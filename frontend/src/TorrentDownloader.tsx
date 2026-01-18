import React, { useState, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Download, RotateCw, FileUp } from 'lucide-react';
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

const MAX_FILE_SIZE = 1024 * 1024; // 1MB

interface Props {
  onDownloadComplete?: () => void;
}

export const TorrentDownloader: React.FC<Props> = ({ onDownloadComplete }) => {
  const [magnetLink, setMagnetLink] = useState<string>('');
  const [torrentFile, setTorrentFile] = useState<File | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [isDragOver, setIsDragOver] = useState<boolean>(false);
  const [mediaType, setMediaType] = useState<string>('');
  const [showDialog, setShowDialog] = useState<boolean>(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { toast } = useToast();

  const validateFile = (file: File): boolean => {
    if (!file.name.endsWith('.torrent')) {
      toast({
        variant: "destructive",
        title: "Invalid file type",
        description: "Please select a .torrent file",
      });
      return false;
    }
    if (file.size > MAX_FILE_SIZE) {
      toast({
        variant: "destructive",
        title: "File too large",
        description: "Torrent file must be less than 1MB",
      });
      return false;
    }
    return true;
  };

  const handleDownload = async () => {
    setLoading(true);
    try {
      const formData = new FormData();
      if (magnetLink) {
        formData.append('magnetLink', magnetLink);
      }
      if (torrentFile) {
        formData.append('torrentFile', torrentFile);
      }
      formData.append('contentType', mediaType);

      const response = await fetch('/api/download', {
        method: 'POST',
        body: formData,
      });

      const data = await response.json();
      if (response.ok) {
        setMagnetLink('');
        setTorrentFile(null);
        setShowDialog(false);
        setMediaType('');
        onDownloadComplete?.();
      } else {
        console.error('Download failed:', data.error);
      }
    } catch (error) {
      console.error('Error:', error);
    }
    setLoading(false);
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
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMagnetLink(e.target.value)}
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
            onClick={() => setShowDialog(true)}
            disabled={!magnetLink && !torrentFile}
            className="w-full"
          >
            <Download className="w-4 h-4" />
            <span className="ml-2">Download</span>
          </Button>
        </div>
      </CardContent>

      <Dialog open={showDialog} onOpenChange={(open) => {
        setShowDialog(open);
        if (!open) setMediaType('');
      }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Select Media Type</DialogTitle>
            <DialogDescription>
              Choose where to save this torrent
            </DialogDescription>
          </DialogHeader>
          <MediaTypeSelector value={mediaType} onValueChange={setMediaType} idPrefix="downloader" />
          <DialogFooter>
            <Button
              onClick={handleDownload}
              disabled={!mediaType || loading}
              className="w-full"
            >
              {loading ? (
                <RotateCw className="w-4 h-4 animate-spin" />
              ) : (
                <Download className="w-4 h-4" />
              )}
              <span className="ml-2">{loading ? 'Downloading...' : 'Download'}</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
};
