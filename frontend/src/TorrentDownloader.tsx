import React, { useState, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Download, RotateCw, FileUp } from 'lucide-react';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { useToast } from '@/hooks/use-toast';

const MAX_FILE_SIZE = 1024 * 1024; // 1MB

interface Props {
  onDownloadComplete?: () => void;
}

export const TorrentDownloader: React.FC<Props> = ({ onDownloadComplete }) => {
  const [magnetLink, setMagnetLink] = useState<string>('');
  const [torrentFile, setTorrentFile] = useState<File | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [isDragOver, setIsDragOver] = useState<boolean>(false);
  const [contentType, setContentType] = useState<string>('Movie');
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
      formData.append('contentType', contentType);

      const response = await fetch('/api/download', {
        method: 'POST',
        body: formData,
      });

      const data = await response.json();
      if (response.ok) {
        setMagnetLink('');
        setTorrentFile(null);
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
          <RadioGroup 
            value={contentType} 
            onValueChange={setContentType} 
            className="flex justify-between mb-4"
          >
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="Movies" id="movie" />
              <Label htmlFor="movie">Movie</Label>
            </div>
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="Series" id="series" />
              <Label htmlFor="series">Series</Label>
            </div>
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="Music" id="music" />
              <Label htmlFor="music">Music</Label>
            </div>
          </RadioGroup>

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
            onClick={handleDownload}
            disabled={loading || (!magnetLink && !torrentFile)}
            className="w-full"
          >
            {loading ? (
              <RotateCw className="w-4 h-4 animate-spin" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            <span className="ml-2">Download</span>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
