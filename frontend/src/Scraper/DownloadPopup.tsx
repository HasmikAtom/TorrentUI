import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Download } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { MediaTypeSelector } from './MediaTypeSelector';

interface Props {
  downloadUrl: string;
  handleDownload: (downloadUrl: string, mediaType: string) => Promise<void>;
  downloading: boolean;
}

export const TDownloadPopup: React.FC<Props> = ({
  downloadUrl,
  handleDownload,
  downloading,
}) => {
  const [mediaType, setMediaType] = useState<string>('');

  const onDownload = async () => {
    if (mediaType && downloadUrl) {
      await handleDownload(downloadUrl, mediaType);
    }
  };

  return (
    <Dialog onOpenChange={() => setMediaType('')}>
      <DialogTrigger asChild>
        <Button
          size="lg"
          className="flex items-center gap-1 w-full justify-center"
        >
          <Download size={16} />
          <span>Download</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Select Media Type</DialogTitle>
          <DialogDescription>
            Choose where to save this torrent
          </DialogDescription>
        </DialogHeader>
        <MediaTypeSelector value={mediaType} onValueChange={setMediaType} />
        <DialogFooter className="sm:justify-start">
          <DialogClose asChild>
            <Button
              disabled={!mediaType || downloading}
              onClick={onDownload}
            >
              <Download />
              {downloading ? 'Downloading...' : 'Download'}
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
