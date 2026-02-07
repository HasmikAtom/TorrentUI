import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Download, Loader2 } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { MediaTypeSelector } from './MediaTypeSelector';
import { PreparedTorrent, PreparedTorrentStatus, BatchPrepareResponse } from '../Models';
import { ScrollArea } from '@/components/ui/scroll-area';

interface Props {
  selectedCount: number;
  downloadUrls: string[];
  isRuTracker?: boolean;
  onDownloadComplete?: () => void;
}

interface EditableTorrent extends PreparedTorrent {
  editedName: string;
}

type PopupState = 'closed' | 'loading' | 'editing';

export const BatchDownloadPopup: React.FC<Props> = ({
  selectedCount,
  downloadUrls,
  isRuTracker = false,
  onDownloadComplete,
}) => {
  const [open, setOpen] = useState(false);
  const [state, setState] = useState<PopupState>('closed');
  const [mediaType, setMediaType] = useState<string>('');
  const [torrents, setTorrents] = useState<EditableTorrent[]>([]);
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [prepareErrors, setPrepareErrors] = useState<string[]>([]);
  const pollingRef = useRef<Map<number, NodeJS.Timeout>>(new Map());
  const abortControllerRef = useRef<AbortController | null>(null);

  const cleanup = useCallback(() => {
    pollingRef.current.forEach((interval) => clearInterval(interval));
    pollingRef.current.clear();
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  }, []);

  const cancelPreparedTorrents = useCallback(async (torrentIds: number[]) => {
    if (torrentIds.length === 0) return;
    try {
      await fetch('/api/download/cancel', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: torrentIds }),
      });
    } catch (e) {
      console.error('Failed to cancel torrents:', e);
    }
  }, []);

  const pollStatus = useCallback(async (torrentId: number) => {
    try {
      const response = await fetch(`/api/download/prepare/status/${torrentId}`);
      if (!response.ok) return;

      const status: PreparedTorrentStatus = await response.json();
      if (status.ready) {
        // Clear the polling interval for this torrent
        const interval = pollingRef.current.get(torrentId);
        if (interval) {
          clearInterval(interval);
          pollingRef.current.delete(torrentId);
        }

        setTorrents((prev) =>
          prev.map((t) =>
            t.id === torrentId
              ? { ...t, name: status.name, editedName: status.name, ready: true }
              : t
          )
        );
      }
    } catch (e) {
      console.error('Polling error:', e);
    }
  }, []);

  const prepareTorrents = useCallback(async () => {
    setState('loading');
    setError(null);
    setPrepareErrors([]);
    abortControllerRef.current = new AbortController();

    try {
      const endpoint = isRuTracker
        ? '/api/download/file/prepare/batch'
        : '/api/download/prepare/batch';

      const body = isRuTracker
        ? { urls: downloadUrls }
        : { magnetLinks: downloadUrls };

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to prepare torrents');
      }

      const data: BatchPrepareResponse = await response.json();

      if (data.errors && data.errors.length > 0) {
        setPrepareErrors(data.errors);
      }

      if (data.torrents.length === 0) {
        throw new Error('No torrents could be prepared');
      }

      const editableTorrents: EditableTorrent[] = data.torrents.map((t) => ({
        ...t,
        editedName: t.name || `torrent-${t.id}`,
      }));

      setTorrents(editableTorrents);

      // Start polling for any torrents that aren't ready
      const notReady = editableTorrents.filter((t) => !t.ready);
      if (notReady.length === 0) {
        setState('editing');
      } else {
        for (const t of notReady) {
          const interval = setInterval(() => pollStatus(t.id), 1000);
          pollingRef.current.set(t.id, interval);
        }

        // Timeout after 60 seconds - allow proceeding with whatever we have
        setTimeout(() => {
          cleanup();
          setTorrents((prev) =>
            prev.map((t) => ({
              ...t,
              ready: true,
              editedName: t.editedName || t.name || `torrent-${t.id}`,
            }))
          );
          setState('editing');
        }, 60000);
      }
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') {
        return;
      }
      setError(e instanceof Error ? e.message : 'Failed to prepare torrents');
      setState('closed');
      setOpen(false);
    }
  }, [downloadUrls, isRuTracker, pollStatus, cleanup]);

  // Check if all torrents are ready when torrents state changes
  useEffect(() => {
    if (state === 'loading' && torrents.length > 0) {
      const allReady = torrents.every((t) => t.ready);
      if (allReady) {
        cleanup();
        setState('editing');
      }
    }
  }, [torrents, state, cleanup]);

  const handleOpenChange = useCallback(
    (isOpen: boolean) => {
      if (isOpen) {
        setOpen(true);
        prepareTorrents();
      } else {
        // Clean up and cancel if we have prepared torrents
        cleanup();
        if (torrents.length > 0 && state === 'loading') {
          cancelPreparedTorrents(torrents.map((t) => t.id));
        }
        setState('closed');
        setMediaType('');
        setTorrents([]);
        setError(null);
        setPrepareErrors([]);
        setOpen(false);
      }
    },
    [prepareTorrents, cleanup, cancelPreparedTorrents, torrents, state]
  );

  const updateTorrentName = (id: number, newName: string) => {
    setTorrents((prev) =>
      prev.map((t) => (t.id === id ? { ...t, editedName: newName } : t))
    );
  };

  const handleDownload = async () => {
    if (!mediaType || torrents.length === 0) return;

    setDownloading(true);
    try {
      const response = await fetch('/api/download/finalize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          torrents: torrents.map((t) => ({
            id: t.id,
            newName: t.editedName !== t.name ? t.editedName : undefined,
          })),
          contentType: mediaType,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to start downloads');
      }

      setOpen(false);
      setState('closed');
      setMediaType('');
      setTorrents([]);
      onDownloadComplete?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start downloads');
    }
    setDownloading(false);
  };

  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  const allNamesValid = torrents.every((t) => t.editedName.trim() !== '');

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button
          size="sm"
          className="flex items-center gap-1 text-white hover:opacity-90"
          style={{ backgroundColor: 'rgb(37, 99, 235)' }}
        >
          <Download size={14} />
          <span>Download Selected ({selectedCount})</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {state === 'loading'
              ? 'Preparing Downloads'
              : `Download ${torrents.length} Torrents`}
          </DialogTitle>
          <DialogDescription>
            {state === 'loading'
              ? 'Fetching torrent info...'
              : 'Edit names and choose where to save'}
          </DialogDescription>
        </DialogHeader>

        {state === 'loading' && (
          <div className="flex flex-col items-center justify-center py-8 gap-2">
            <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
            {torrents.length > 0 && (
              <p className="text-sm text-muted-foreground">
                {torrents.filter((t) => t.ready).length} / {torrents.length} ready
              </p>
            )}
          </div>
        )}

        {state === 'editing' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Torrent Names</label>
              <ScrollArea className="h-48 rounded-md border p-2">
                <div className="space-y-2">
                  {torrents.map((t, index) => (
                    <Input
                      key={t.id}
                      value={t.editedName}
                      onChange={(e) => updateTorrentName(t.id, e.target.value)}
                      placeholder={`Torrent ${index + 1}`}
                    />
                  ))}
                </div>
              </ScrollArea>
            </div>
            <MediaTypeSelector
              value={mediaType}
              onValueChange={setMediaType}
              idPrefix="batch"
            />
          </div>
        )}

        {prepareErrors.length > 0 && (
          <div className="text-sm text-amber-500">
            <p>Some torrents failed to prepare:</p>
            <ul className="list-disc list-inside">
              {prepareErrors.slice(0, 3).map((err, i) => (
                <li key={i}>{err}</li>
              ))}
              {prepareErrors.length > 3 && (
                <li>...and {prepareErrors.length - 3} more</li>
              )}
            </ul>
          </div>
        )}

        {error && <p className="text-sm text-red-500">{error}</p>}

        <DialogFooter className="sm:justify-start">
          {state === 'loading' && (
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
          )}
          {state === 'editing' && (
            <Button
              disabled={!mediaType || downloading || !allNamesValid}
              onClick={handleDownload}
              className="text-white hover:opacity-90"
              style={{ backgroundColor: 'rgb(37, 99, 235)' }}
            >
              {downloading ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Download className="h-4 w-4 mr-2" />
              )}
              {downloading
                ? 'Starting...'
                : `Download ${torrents.length} Torrents`}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
