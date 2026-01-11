import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Download } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { Label } from '@radix-ui/react-label';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { RadioGroup, RadioGroupItem } from '../components/ui/radio-group';

interface props {
  selectedCount: number;
  onBatchDownload: (mediaType: string) => Promise<void>;
  downloading: boolean;
}

export const BatchDownloadPopup: React.FC<props> = ({
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
        <div className="flex items-center space-x-2">
          <div className="grid flex-1 gap-2">
            <RadioGroup
              value={mediaType}
              onValueChange={setMediaType}
              className="flex justify-between mb-4"
            >
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="Movies" id="batch-movie" />
                <Label htmlFor="batch-movie">Movie</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="Series" id="batch-series" />
                <Label htmlFor="batch-series">Series</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="Music" id="batch-music" />
                <Label htmlFor="batch-music">Music</Label>
              </div>
            </RadioGroup>
          </div>
        </div>
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
