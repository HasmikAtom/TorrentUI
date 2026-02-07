import React, { useState, useRef, useEffect } from 'react';
import { ScrapedTorrents } from '../Models';
import { ScrapeSearch } from './ScrapeSearch';
import { ScrapedTorrentsCards } from './ScrapedTorrents';
import { useToast } from '@/hooks/use-toast';

const ScraperConfig = {
  thepiratebay: {
    scrapeEndpoint: '/api/scrape/piratebay/',
    scrapeStreamEndpoint: '/api/scrape/piratebay/',
    downloadSource: 'magnet' as const,
  },
  rutracker: {
    scrapeEndpoint: '/api/scrape/rutracker/',
    scrapeStreamEndpoint: '/api/scrape/rutracker/',
    downloadSource: 'download_url' as const,
  }
} as const

interface SSEEvent {
  type: 'trying' | 'success' | 'error' | 'complete';
  message: string;
  host?: string;
  label?: string;
  data?: any;
}

export type DownloadSource = typeof ScraperConfig[keyof typeof ScraperConfig]['downloadSource'];

interface Props {
  type: keyof typeof ScraperConfig
  switchTab: (tabValue: string) => void;
}

export const ScraperUI: React.FC<Props> = ({
  type,
  switchTab,
}) => {

    const [searchLoading, setSearchLoading] = useState<boolean>(false);
    const [torrentName, setTorrentName] = useState<string>("");
    const [foundTorrents, setFoundTorrents] = useState<ScrapedTorrents[] | null>(null);
    const [selectedTorrents, setSelectedTorrents] = useState<Map<string, string>>(new Map());
    const eventSourceRef = useRef<EventSource | null>(null);
    const { toast } = useToast();

    // Cleanup EventSource on unmount
    useEffect(() => {
      return () => {
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
        }
      };
    }, []);

    const config = ScraperConfig[type];
    const downloadSource = config.downloadSource;

    const handleScrapeSearch = async () => {
      // Close any existing EventSource
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }

      setSearchLoading(true);
      setFoundTorrents(null);

      // Use SSE for real-time progress updates
      const streamUrl = `${config.scrapeStreamEndpoint}${encodeURIComponent(torrentName)}/stream`;
      const eventSource = new EventSource(streamUrl);
      eventSourceRef.current = eventSource;

      eventSource.onmessage = (event) => {
        try {
          const data: SSEEvent = JSON.parse(event.data);

          switch (data.type) {
            case 'trying':
              toast({
                title: `Searching ${data.label}...`,
                description: data.message,
              });
              break;

            case 'success':
              toast({
                title: "Source found!",
                description: data.message,
              });
              break;

            case 'error':
              toast({
                variant: "destructive",
                title: `${data.label} failed`,
                description: data.message,
              });
              break;

            case 'complete':
              eventSource.close();
              eventSourceRef.current = null;
              setSearchLoading(false);

              if (data.data && Array.isArray(data.data) && data.data.length > 0) {
                setFoundTorrents(data.data);
                toast({
                  title: "Search complete",
                  description: data.message,
                });
              } else {
                toast({
                  variant: "destructive",
                  title: "No results",
                  description: data.message || "No torrents found",
                });
              }
              break;
          }
        } catch (e) {
          console.error('Failed to parse SSE event:', e);
        }
      };

      eventSource.onerror = () => {
        eventSource.close();
        eventSourceRef.current = null;
        setSearchLoading(false);
        toast({
          variant: "destructive",
          title: "Connection error",
          description: "Lost connection to server",
        });
      };
    }

    const handleDownloadComplete = () => {
      clearSelection();
      switchTab("download");
      toast({
        title: "Download started",
        description: "Torrent(s) added to queue",
      });
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
        selectedTorrents={selectedTorrents}
        onToggleSelection={toggleTorrentSelection}
        onSelectAll={selectAllTorrents}
        onClearSelection={clearSelection}
        onDownloadComplete={handleDownloadComplete}
      />
    </>
  );
}
