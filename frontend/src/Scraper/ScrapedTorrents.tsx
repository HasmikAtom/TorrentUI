import React, { useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

import { TorrentDownloadPopup } from './DownloadPopup';
import { BatchDownloadPopup } from './BatchDownloadPopup';
import { ScrapedTorrents } from '../Models';
import { DownloadSource } from './ScraperUI';

interface Props {
  foundTorrents: ScrapedTorrents[] | null;
  downloadSource: DownloadSource;
  selectedTorrents: Map<string, string>;
  onToggleSelection: (id: string, downloadUrl: string) => void;
  onSelectAll: () => void;
  onClearSelection: () => void;
  onDownloadComplete?: () => void;
  filterText: string;
  onFilterChange: (value: string) => void;
  filterText2: string;
  onFilterChange2: (value: string) => void;
  selectedUploaders: Set<string>;
  onToggleUploader: (uploader: string) => void;
}

export const ScrapedTorrentsCards: React.FC<Props> = React.memo(({
  foundTorrents,
  downloadSource,
  selectedTorrents,
  onToggleSelection,
  onSelectAll,
  onClearSelection,
  onDownloadComplete,
  filterText,
  onFilterChange,
  filterText2,
  onFilterChange2,
  selectedUploaders,
  onToggleUploader,
}) => {

  const filteredTorrents = useMemo(() => {
    if (!foundTorrents) return foundTorrents;
    let results = foundTorrents;
    if (filterText.trim()) {
      const lower = filterText.toLowerCase();
      results = results.filter(t => t.title.toLowerCase().includes(lower));
    }
    if (filterText2.trim()) {
      const lower2 = filterText2.toLowerCase();
      results = results.filter(t => t.title.toLowerCase().includes(lower2));
    }
    if (selectedUploaders.size > 0) {
      results = results.filter(t => selectedUploaders.has(t.uploader));
    }
    return results;
  }, [foundTorrents, filterText, filterText2, selectedUploaders]);

  const uploaders = useMemo(() => {
    if (!foundTorrents) return [];
    const set = new Set<string>();
    foundTorrents.forEach(t => { if (t.uploader) set.add(t.uploader); });
    return Array.from(set).sort();
  }, [foundTorrents]);

  const selectedCount = selectedTorrents.size;
  const isRuTracker = downloadSource === 'download_url';

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
                  downloadUrls={Array.from(selectedTorrents.values())}
                  isRuTracker={isRuTracker}
                  onDownloadComplete={onDownloadComplete}
                />
              )}
            </div>
          )}
        </div>
        {foundTorrents && foundTorrents.length > 0 && (
          <div className="flex gap-6 !mt-6">
            <div className="w-1/2 flex flex-col gap-2">
              <Input
                placeholder="Primary filter..."
                value={filterText}
                onChange={(e) => onFilterChange(e.target.value)}
              />
              <Input
                placeholder="Secondary filter..."
                value={filterText2}
                onChange={(e) => onFilterChange2(e.target.value)}
              />
            </div>
            <div className="w-1/2 flex flex-wrap gap-1 content-start overflow-auto max-h-20">
              {uploaders.map(uploader => (
                <button
                  key={uploader}
                  onClick={() => onToggleUploader(uploader)}
                  className={`px-2 py-0.5 rounded-full text-xs border cursor-pointer transition-colors ${
                    selectedUploaders.has(uploader)
                      ? 'bg-primary text-primary-foreground border-primary'
                      : 'bg-muted text-muted-foreground border-border hover:bg-accent'
                  }`}
                >
                  {uploader}
                </button>
              ))}
            </div>
          </div>
        )}
      </CardHeader>
      <CardContent>
        <div>
          {filteredTorrents && filteredTorrents.length > 0 ? (
            <div>
              <div className="space-y-4">
                {filteredTorrents.map((torrent) => {
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
                            <TorrentDownloadPopup
                              downloadUrl={downloadUrl}
                              isRuTracker={isRuTracker}
                              onDownloadComplete={onDownloadComplete}
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
            <p className="text-center text-gray-500">
              {foundTorrents && foundTorrents.length > 0 ? "No matching results" : "No active torrents"}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
});
