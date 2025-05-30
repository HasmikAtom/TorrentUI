import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

import { TDownloadPopup } from './DownloadPopup';
import { ScrapedTorrents } from '../Models';
import { DownloadSource } from './ScraperUI';

interface props {
  foundTorrents: ScrapedTorrents[] | null;
  contentType: string;
  downloadSource: DownloadSource;
  mediaTypeSelected: boolean;
  handleTorrentDownload: () => Promise<void>;
  setContentType: (type: string) => void;
  selectTorrent: (mediaType: string, selectedMagnet: string) => void;
}



export const ScrapedTorrentsCards: React.FC<props> = ({
  foundTorrents,
  contentType,
  mediaTypeSelected,
  downloadSource,
  handleTorrentDownload,
  setContentType,
  selectTorrent,
  }) => {
    
  
  
  return (
    <Card className="w-full max-w-2xl mx-auto mt-8">
      <CardHeader>
        <CardTitle>Pirate Search Results</CardTitle>
      </CardHeader>
      <CardContent>
        <div >
          {foundTorrents && foundTorrents.length > 0 ? (
            <div >
              <div className="space-y-4">
                {foundTorrents.map((torrent, index) => (
                  
                  <div key={index} className="space-y-2 border-b pb-4 last:border-b-0">
                    <div className="flex justify-between truncate">
                      <span className="font-medium">Title: </span>
                      <span className='w-[400px]'>{torrent.title}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="font-medium">Category: </span>
                      <span>{torrent.category}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="font-medium">Size: </span>
                      <span>{torrent.size} </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="font-medium">Seeders: </span>
                      <span>{torrent.se}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="font-medium">Leechers: </span>
                      <span>{torrent.le}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="font-medium">Uploader: </span>
                      <span>{torrent.uploader}</span>
                    </div>

                    <div className="flex space-x-2">
                      <TDownloadPopup
                        torrent={torrent}
                        contentType={contentType}
                        mediaTypeSelected={mediaTypeSelected}
                        downloadSource={downloadSource}
                        setContentType={setContentType}
                        selectTorrent={selectTorrent}
                        handleDownload={handleTorrentDownload}
                      /> 
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <p className="text-center text-gray-500">No active torrents</p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}