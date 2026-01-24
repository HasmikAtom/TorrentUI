import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

import { TDownloadPopup } from './DownloadPopup';
import { BatchDownloadPopup } from './BatchDownloadPopup';
import { ScrapedTorrents } from '../Models';
import { DownloadSource } from './ScraperUI';

interface Props {
  foundTorrents: ScrapedTorrents[] | null;
  downloadSource: DownloadSource;
  handleSingleDownload: (downloadUrl: string, mediaType: string) => Promise<void>;
  selectedTorrents: Map<string, string>;
  onToggleSelection: (id: string, downloadUrl: string) => void;
  onSelectAll: () => void;
  onClearSelection: () => void;
  onBatchDownload: (mediaType: string) => Promise<void>;
  downloading: boolean;
}

export const ScrapedTorrentsCards: React.FC<Props> = React.memo(({
  foundTorrents,
  downloadSource,
  handleSingleDownload,
  selectedTorrents,
  onToggleSelection,
  onSelectAll,
  onClearSelection,
  onBatchDownload,
  downloading,
}) => {

  const selectedCount = selectedTorrents.size;

  return (
    <Card className="w-full max-w-2xl mx-auto mt-8">
      <CardHeader>
        <div className="flex justify-between items-center">
          <CardTitle>Search Results</CardTitle>
          {foundTorrents && foundTorrents.length > 0 && (
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={onSelectAll}
              >
                Select All
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onClearSelection}
                disabled={selectedCount === 0}
              >
                Clear
              </Button>
              {selectedCount > 0 && (
                <BatchDownloadPopup
                  selectedCount={selectedCount}
                  onBatchDownload={onBatchDownload}
                  downloading={downloading}
                />
              )}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div>
          {foundTorrents && foundTorrents.length > 0 ? (
            <div>
              <div className="space-y-4">
                {foundTorrents.map((torrent) => {
                  const downloadUrl = torrent[downloadSource] || '';
                  const isSelected = selectedTorrents.has(torrent.id);

                  return (
                    <div
                      key={torrent.id}
                      className={`space-y-2 border-b pb-4 last:border-b-0 ${isSelected ? 'bg-accent -mx-4 px-4 py-2 rounded' : ''}`}
                    >
                      <div className="flex items-start gap-3">
                        <input
                          type="checkbox"
                          id={`torrent-${torrent.id}`}
                          checked={isSelected}
                          onChange={() => onToggleSelection(torrent.id, downloadUrl)}
                          className="mt-1 h-4 w-4 rounded border-gray-300 cursor-pointer"
                          aria-label={`Select ${torrent.title}`}
                        />
                        <div className="flex-1 space-y-2">
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
                              downloadUrl={downloadUrl}
                              handleDownload={handleSingleDownload}
                              downloading={downloading}
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : (
            <p className="text-center text-gray-500">No active torrents</p>
          )}
        </div>
      </CardContent>
    </Card>
  );
});
