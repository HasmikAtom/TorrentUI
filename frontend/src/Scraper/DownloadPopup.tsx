import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Download, Loader2 } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { MediaTypeSelector } from './MediaTypeSelector';
import { PreparedTorrent, PreparedTorrentStatus } from '../Models';

interface Props {
  downloadUrl: string;
  isRuTracker?: boolean;
  onDownloadComplete?: () => void;
}

type PopupState = 'closed' | 'loading' | 'editing';

export const TorrentDownloadPopup: React.FC<Props> = ({
  downloadUrl,
  isRuTracker = false,
  onDownloadComplete,
}) => {
  const [open, setOpen] = useState(false);
  const [state, setState] = useState<PopupState>('closed');
  const [mediaType, setMediaType] = useState<string>('');
  const [preparedTorrent, setPreparedTorrent] = useState<PreparedTorrent | null>(null);
  const [editedName, setEditedName] = useState<string>('');
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<NodeJS.Timeout | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  const cleanup = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  }, []);

  const cancelPreparedTorrent = useCallback(async (torrentId: number) => {
    try {
      await fetch('/api/download/cancel', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: [torrentId] }),
      });
    } catch (e) {
      console.error('Failed to cancel torrent:', e);
    }
  }, []);

  const pollStatus = useCallback(async (torrentId: number) => {
    try {
      const response = await fetch(`/api/download/prepare/status/${torrentId}`);
      if (!response.ok) return;

      const status: PreparedTorrentStatus = await response.json();
      if (status.ready) {
        cleanup();
        setPreparedTorrent({ id: status.id, name: status.name, ready: true });
        setEditedName(status.name);
        setState('editing');
      }
    } catch (e) {
      console.error('Polling error:', e);
    }
  }, [cleanup]);

  const prepareTorrent = useCallback(async () => {
    setState('loading');
    setError(null);
    abortControllerRef.current = new AbortController();

    try {
      const formData = new FormData();
      if (isRuTracker) {
        formData.append('url', downloadUrl);
      } else {
        formData.append('magnetLink', downloadUrl);
      }

      const endpoint = isRuTracker ? '/api/download/file/prepare' : '/api/download/prepare';
      const response = await fetch(endpoint, {
        method: 'POST',
        body: formData,
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to prepare torrent');
      }

      const data: PreparedTorrent = await response.json();
      setPreparedTorrent(data);

      if (data.ready) {
        setEditedName(data.name);
        setState('editing');
      } else {
        // Start polling for metadata
        pollingRef.current = setInterval(() => pollStatus(data.id), 1000);

        // Timeout after 60 seconds
        setTimeout(() => {
          if (state === 'loading' && preparedTorrent && !preparedTorrent.ready) {
            cleanup();
            // Allow proceeding with hash-based name
            setEditedName(preparedTorrent.name || `torrent-${preparedTorrent.id}`);
            setState('editing');
          }
        }, 60000);
      }
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') {
        return;
      }
      setError(e instanceof Error ? e.message : 'Failed to prepare torrent');
      setState('closed');
      setOpen(false);
    }
  }, [downloadUrl, isRuTracker, pollStatus, cleanup, state, preparedTorrent]);

  const handleOpenChange = useCallback((isOpen: boolean) => {
    if (isOpen) {
      setOpen(true);
      prepareTorrent();
    } else {
      // Clean up and cancel if we have a prepared torrent
      cleanup();
      if (preparedTorrent && state === 'loading') {
        cancelPreparedTorrent(preparedTorrent.id);
      }
      setState('closed');
      setMediaType('');
      setPreparedTorrent(null);
      setEditedName('');
      setError(null);
      setOpen(false);
    }
  }, [prepareTorrent, cleanup, cancelPreparedTorrent, preparedTorrent, state]);

  const handleDownload = async () => {
    if (!mediaType || !preparedTorrent) return;

    setDownloading(true);
    try {
      const response = await fetch('/api/download/finalize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          torrents: [{
            id: preparedTorrent.id,
            newName: editedName !== preparedTorrent.name ? editedName : undefined,
          }],
          contentType: mediaType,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to start download');
      }

      setOpen(false);
      setState('closed');
      setMediaType('');
      setPreparedTorrent(null);
      setEditedName('');
      onDownloadComplete?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start download');
    }
    setDownloading(false);
  };

  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button
          size="lg"
          className="flex items-center gap-1 w-full justify-center text-white hover:opacity-90"
          style={{ backgroundColor: 'rgb(37, 99, 235)' }}
        >
          <Download size={16} />
          <span>Download</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {state === 'loading' ? 'Preparing Download' : 'Download Torrent'}
          </DialogTitle>
          <DialogDescription>
            {state === 'loading'
              ? 'Fetching torrent info...'
              : 'Edit the name and choose where to save'}
          </DialogDescription>
        </DialogHeader>

        {state === 'loading' && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
          </div>
        )}

        {state === 'editing' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="torrent-name" className="text-sm font-medium">
                Torrent Name
              </label>
              <Input
                id="torrent-name"
                value={editedName}
                onChange={(e) => setEditedName(e.target.value)}
                placeholder="Enter torrent name"
              />
            </div>
            <MediaTypeSelector value={mediaType} onValueChange={setMediaType} />
          </div>
        )}

        {error && (
          <p className="text-sm text-red-500">{error}</p>
        )}

        <DialogFooter className="sm:justify-start">
          {state === 'loading' && (
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
          )}
          {state === 'editing' && (
            <Button
              disabled={!mediaType || downloading || !editedName.trim()}
              onClick={handleDownload}
              className="text-white hover:opacity-90"
              style={{ backgroundColor: 'rgb(37, 99, 235)' }}
            >
              {downloading ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Download className="h-4 w-4 mr-2" />
              )}
              {downloading ? 'Starting...' : 'Start Download'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
