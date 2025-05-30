import React from 'react';
import { Button } from '@/components/ui/button';

import { Download } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { Label } from '@radix-ui/react-label';
import { DialogHeader, DialogFooter } from '../components/ui/dialog';
import { RadioGroup, RadioGroupItem } from '../components/ui/radio-group';
import { ScrapedTorrents } from '../Models';
import { DownloadSource } from './ScraperUI';


interface props {
  contentType: string;
  torrent: ScrapedTorrents,
  mediaTypeSelected?: boolean;
  downloadSource: DownloadSource;
  setContentType: (type: string) => void;
  selectTorrent: (type: string, downloadUrl: string) => void;
  handleDownload?: () => Promise<void>;
}


export const TDownloadPopup: React.FC<props> = ({
  contentType,
  torrent,
  downloadSource,
  mediaTypeSelected,
  setContentType,
  selectTorrent,
  handleDownload,
}) => {

  return (
    <Dialog onOpenChange={() => setContentType("")}>
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
          </DialogDescription>
        </DialogHeader>
        <div className="flex items-center space-x-2">
          <div className="grid flex-1 gap-2">
            <RadioGroup
              value={contentType}
              onValueChange={(selected) => selectTorrent(selected, torrent[downloadSource])}
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
          </div>
        </div>
        <DialogFooter className="sm:justify-start">
          <DialogClose asChild>
            <Button
              disabled={!mediaTypeSelected}
              onClick={handleDownload}
            >
              <Download />
              Download
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}