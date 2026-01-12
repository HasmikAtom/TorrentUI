import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Download } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { MediaTypeSelector } from './MediaTypeSelector';

interface Props {
  selectedCount: number;
  onBatchDownload: (mediaType: string) => Promise<void>;
  downloading: boolean;
}

export const BatchDownloadPopup: React.FC<Props> = ({
  selectedCount,
  onBatchDownload,
  downloading,
}) => {
  const [mediaType, setMediaType] = useState<string>('');

  const handleDownload = async () => {
    if (mediaType) {
      await onBatchDownload(mediaType);
    }
  };

  return (
    <Dialog onOpenChange={() => setMediaType('')}>
      <DialogTrigger asChild>
        <Button
          size="sm"
          className="flex items-center gap-1"
        >
          <Download size={14} />
          <span>Download Selected ({selectedCount})</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Download {selectedCount} Torrents</DialogTitle>
          <DialogDescription>
            Select media type for all selected torrents
          </DialogDescription>
        </DialogHeader>
        <MediaTypeSelector value={mediaType} onValueChange={setMediaType} idPrefix="batch" />
        <DialogFooter className="sm:justify-start">
          <DialogClose asChild>
            <Button
              disabled={!mediaType || downloading}
              onClick={handleDownload}
            >
              <Download />
              {downloading ? 'Downloading...' : `Download ${selectedCount} Torrents`}
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
