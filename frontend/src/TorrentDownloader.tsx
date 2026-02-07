import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Download, FileUp, Loader2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { MediaTypeSelector } from './Scraper/MediaTypeSelector';
import { PreparedTorrent, PreparedTorrentStatus } from './Models';

const MAX_FILE_SIZE = 1024 * 1024; // 1MB

interface Props {
  onDownloadComplete?: () => void;
}

type DialogState = 'closed' | 'loading' | 'editing';

export const TorrentDownloader: React.FC<Props> = ({ onDownloadComplete }) => {
  const [magnetLink, setMagnetLink] = useState<string>('');
  const [torrentFile, setTorrentFile] = useState<File | null>(null);
  const [isDragOver, setIsDragOver] = useState<boolean>(false);
  const [mediaType, setMediaType] = useState<string>('');
  const [dialogState, setDialogState] = useState<DialogState>('closed');
  const [preparedTorrent, setPreparedTorrent] = useState<PreparedTorrent | null>(null);
  const [editedName, setEditedName] = useState<string>('');
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const pollingRef = useRef<NodeJS.Timeout | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const { toast } = useToast();

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

  const validateFile = (file: File): boolean => {
    if (!file.name.endsWith('.torrent')) {
      toast({
        variant: 'destructive',
        title: 'Invalid file type',
        description: 'Please select a .torrent file',
      });
      return false;
    }
    if (file.size > MAX_FILE_SIZE) {
      toast({
        variant: 'destructive',
        title: 'File too large',
        description: 'Torrent file must be less than 1MB',
      });
      return false;
    }
    return true;
  };

  const pollStatus = useCallback(async (torrentId: number) => {
    try {
      const response = await fetch(`/api/download/prepare/status/${torrentId}`);
      if (!response.ok) return;

      const status: PreparedTorrentStatus = await response.json();
      if (status.ready) {
        cleanup();
        setPreparedTorrent({ id: status.id, name: status.name, ready: true });
        setEditedName(status.name);
        setDialogState('editing');
      }
    } catch (e) {
      console.error('Polling error:', e);
    }
  }, [cleanup]);

  const prepareTorrent = useCallback(async () => {
    setDialogState('loading');
    setError(null);
    abortControllerRef.current = new AbortController();

    try {
      const formData = new FormData();
      if (magnetLink) {
        formData.append('magnetLink', magnetLink);
      }
      if (torrentFile) {
        formData.append('torrentFile', torrentFile);
      }

      const response = await fetch('/api/download/prepare', {
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
        setDialogState('editing');
      } else {
        // Start polling for metadata
        pollingRef.current = setInterval(() => pollStatus(data.id), 1000);

        // Timeout after 60 seconds
        setTimeout(() => {
          if (dialogState === 'loading' && preparedTorrent && !preparedTorrent.ready) {
            cleanup();
            setEditedName(preparedTorrent.name || `torrent-${preparedTorrent.id}`);
            setDialogState('editing');
          }
        }, 60000);
      }
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') {
        return;
      }
      setError(e instanceof Error ? e.message : 'Failed to prepare torrent');
      toast({
        variant: 'destructive',
        title: 'Prepare failed',
        description: e instanceof Error ? e.message : 'Failed to prepare torrent',
      });
      setDialogState('closed');
    }
  }, [magnetLink, torrentFile, pollStatus, cleanup, dialogState, preparedTorrent, toast]);

  const handleDownloadClick = () => {
    prepareTorrent();
  };

  const handleDialogClose = useCallback((isOpen: boolean) => {
    if (!isOpen) {
      cleanup();
      if (preparedTorrent && dialogState === 'loading') {
        cancelPreparedTorrent(preparedTorrent.id);
      }
      setDialogState('closed');
      setMediaType('');
      setPreparedTorrent(null);
      setEditedName('');
      setError(null);
    }
  }, [cleanup, cancelPreparedTorrent, preparedTorrent, dialogState]);

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

      setMagnetLink('');
      setTorrentFile(null);
      setDialogState('closed');
      setMediaType('');
      setPreparedTorrent(null);
      setEditedName('');
      onDownloadComplete?.();
      toast({
        title: 'Download started',
        description: 'Torrent added to queue',
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start download');
      toast({
        variant: 'destructive',
        title: 'Download failed',
        description: e instanceof Error ? e.message : 'Failed to start download',
      });
    }
    setDownloading(false);
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const file = e.target.files[0];
      if (validateFile(file)) {
        setTorrentFile(file);
      }
    }
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => {
    setIsDragOver(false);
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(false);
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      const file = files[0];
      if (validateFile(file)) {
        setTorrentFile(file);
      }
    }
  };

  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  const isOpen = dialogState !== 'closed';

  return (
    <Card className="w-full max-w-2xl mx-auto mt-8">
      <CardHeader>
        <CardTitle>Transmission Download Manager</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <Input
            type="text"
            placeholder="Enter magnet link..."
            value={magnetLink}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setMagnetLink(e.target.value)
            }
            className="w-full"
          />

          <Input
            type="file"
            accept=".torrent"
            onChange={handleFileChange}
            className="hidden"
            ref={fileInputRef}
          />
          <Button
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            className="w-full"
          >
            <FileUp className="w-4 h-4 mr-2" />
            Select File
          </Button>

          <div
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            className={`border-2 border-dashed p-4 text-center ${
              isDragOver ? 'border-blue-500 bg-blue-50' : 'border-gray-300'
            }`}
          >
            {torrentFile ? (
              <p>Selected file: {torrentFile.name}</p>
            ) : (
              <p>Drag and drop .torrent file here</p>
            )}
          </div>

          <Button
            onClick={handleDownloadClick}
            disabled={!magnetLink && !torrentFile}
            className="w-full text-white hover:opacity-90"
            style={{ backgroundColor: 'rgb(37, 99, 235)' }}
          >
            <Download className="w-4 h-4" />
            <span className="ml-2">Download</span>
          </Button>
        </div>
      </CardContent>

      <Dialog open={isOpen} onOpenChange={handleDialogClose}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {dialogState === 'loading' ? 'Preparing Download' : 'Download Torrent'}
            </DialogTitle>
            <DialogDescription>
              {dialogState === 'loading'
                ? 'Fetching torrent info...'
                : 'Edit the name and choose where to save'}
            </DialogDescription>
          </DialogHeader>

          {dialogState === 'loading' && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
            </div>
          )}

          {dialogState === 'editing' && (
            <div className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="downloader-torrent-name" className="text-sm font-medium">
                  Torrent Name
                </label>
                <Input
                  id="downloader-torrent-name"
                  value={editedName}
                  onChange={(e) => setEditedName(e.target.value)}
                  placeholder="Enter torrent name"
                />
              </div>
              <MediaTypeSelector
                value={mediaType}
                onValueChange={setMediaType}
                idPrefix="downloader"
              />
            </div>
          )}

          {error && <p className="text-sm text-red-500">{error}</p>}

          <DialogFooter>
            {dialogState === 'loading' && (
              <Button variant="outline" onClick={() => handleDialogClose(false)}>
                Cancel
              </Button>
            )}
            {dialogState === 'editing' && (
              <Button
                onClick={handleDownload}
                disabled={!mediaType || downloading || !editedName.trim()}
                className="w-full text-white hover:opacity-90"
                style={{ backgroundColor: 'rgb(37, 99, 235)' }}
              >
                {downloading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Download className="w-4 h-4" />
                )}
                <span className="ml-2">
                  {downloading ? 'Starting...' : 'Start Download'}
                </span>
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
};
