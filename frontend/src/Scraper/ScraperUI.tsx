import React, { useState } from 'react';
import { ScrapedTorrents } from '../Models';
import { ScrapeSearch } from './ScrapeSearch';
import { ScrapedTorrentsCards } from './ScrapedTorrents';

const ScraperConfig = {
  thepiratebay: {
    scrapeEndpoint: '/api/scrape/piratebay/',
    downloadEndpoint: '/api/download/batch',
    downloadKey: 'magnetLinks',
    downloadSource: 'magnet' as const,
  },
  rutracker: {
    scrapeEndpoint: '/api/scrape/rutracker/',
    downloadEndpoint: '/api/download/file/batch',
    downloadKey: 'urls',
    downloadSource: 'download_url' as const,
  }
} as const

export type DownloadSource = typeof ScraperConfig[keyof typeof ScraperConfig]['downloadSource'];

interface props {
  type: keyof typeof ScraperConfig
  switchTab: (tabValue: string) => void;
}

export const ScraperUI: React.FC<props> = ({
  type,
  switchTab,
}) => {

    const [searchLoading, setSearchLoading] = useState<boolean>(false);
    const [downloading, setDownloading] = useState<boolean>(false);
    const [torrentName, setTorrentName] = useState<string>("");
    const [foundTorrents, setFoundTorrents] = useState<ScrapedTorrents[] | null>(null);
    const [selectedTorrents, setSelectedTorrents] = useState<Map<string, string>>(new Map());

    const config = ScraperConfig[type];
    const downloadSource = config.downloadSource;

    const handleScrapeSearch = async () => {
      setSearchLoading(true);

        try {
            const response = await fetch(`${config.scrapeEndpoint}${torrentName}`, {
                method: "POST",
            });

            const data = await response.json();
            if(response.ok) {
                console.log(data)
                setFoundTorrents(data)
            } else {
                console.error("Search Failed", data.Error)
            }

        }
        catch(error) {
            console.error("Error", error)
        }
        setSearchLoading(false);
    }

    // Unified download handler - works for both single and batch downloads
    const handleDownload = async (downloadUrls: string[], mediaType: string) => {
      if (downloadUrls.length === 0) return;

      setDownloading(true);
      try {
        const body = {
          [config.downloadKey]: downloadUrls,
          contentType: mediaType,
        };

        const response = await fetch(config.downloadEndpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(body),
        });

        const data = await response.json();
        if (response.ok) {
          console.log('Download started:', data);
        } else {
          console.error('Download failed:', data.error);
        }
      } catch (error) {
        console.error('Error:', error);
      }

      setDownloading(false);
      clearSelection();
      switchTab("download");
    };

    // Single torrent download (from individual Download button)
    const handleSingleDownload = async (downloadUrl: string, mediaType: string) => {
      await handleDownload([downloadUrl], mediaType);
    };

    // Batch download (from Download Selected button)
    const handleBatchDownload = async (mediaType: string) => {
      const downloadUrls = Array.from(selectedTorrents.values());
      await handleDownload(downloadUrls, mediaType);
    };

    const handleTorrentSearchClear = async () => {
      setSearchLoading(false);
      setFoundTorrents(null);
      setTorrentName("");
      setSelectedTorrents(new Map());
    }

    const toggleTorrentSelection = (id: string, downloadUrl: string) => {
      setSelectedTorrents(prev => {
        const newMap = new Map(prev);
        if (newMap.has(id)) {
          newMap.delete(id);
        } else {
          newMap.set(id, downloadUrl);
        }
        return newMap;
      });
    };

    const selectAllTorrents = () => {
      if (!foundTorrents) return;
      const newMap = new Map<string, string>();
      foundTorrents.forEach(torrent => {
        const downloadUrl = torrent[downloadSource] || '';
        if (downloadUrl) {
          newMap.set(torrent.id, downloadUrl);
        }
      });
      setSelectedTorrents(newMap);
    };

    const clearSelection = () => {
      setSelectedTorrents(new Map());
    };

  return (
    <>
      <ScrapeSearch
        torrentName={torrentName}
        searchLoading={searchLoading}
        setTorrentName={setTorrentName}
        handleTorrentSearch={handleScrapeSearch}
        handleTorrentSearchClear={handleTorrentSearchClear}
      />

      <ScrapedTorrentsCards
        foundTorrents={foundTorrents}
        downloadSource={downloadSource}
        handleSingleDownload={handleSingleDownload}
        selectedTorrents={selectedTorrents}
        onToggleSelection={toggleTorrentSelection}
        onSelectAll={selectAllTorrents}
        onClearSelection={clearSelection}
        onBatchDownload={handleBatchDownload}
        downloading={downloading}
      />
    </>
  );
}
