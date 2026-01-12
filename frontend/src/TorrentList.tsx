import { useState, useEffect, useRef, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "./components/ui/card";
import { Button } from "./components/ui/button";
import { RefreshCw } from "lucide-react";
import { TorrentStatus } from "./Models";

const POLL_INTERVAL = 3000;

interface Props {
  refreshTrigger?: number;
}

export const TorrentList: React.FC<Props> = ({ refreshTrigger }) => {
    const [torrents, setTorrents] = useState<TorrentStatus[] | null>(null);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const intervalRef = useRef<number | null>(null);

    const fetchTorrents = useCallback(async () => {
      try {
        const response = await fetch(`/api/torrents`);
        const data = await response.json();
        if (response.ok) {
          setTorrents(data);
        }
      } catch (error) {
        console.error('Error fetching torrents:', error);
      }
    }, []);

    const handleManualRefresh = async () => {
      setIsRefreshing(true);
      await fetchTorrents();
      setIsRefreshing(false);
    };

    const startPolling = useCallback(() => {
      if (intervalRef.current === null) {
        intervalRef.current = window.setInterval(fetchTorrents, POLL_INTERVAL);
      }
    }, [fetchTorrents]);

    const stopPolling = useCallback(() => {
      if (intervalRef.current !== null) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }, []);

    // Initial fetch and polling with Page Visibility API
    useEffect(() => {
      fetchTorrents();
      startPolling();

      const handleVisibilityChange = () => {
        if (document.hidden) {
          stopPolling();
        } else {
          fetchTorrents(); // Refresh immediately when tab becomes visible
          startPolling();
        }
      };

      document.addEventListener('visibilitychange', handleVisibilityChange);

      return () => {
        stopPolling();
        document.removeEventListener('visibilitychange', handleVisibilityChange);
      };
    }, [fetchTorrents, startPolling, stopPolling]);

    useEffect(() => {
      if (refreshTrigger !== undefined) {
        fetchTorrents();
      }
    }, [refreshTrigger, fetchTorrents]);

    return (
      <Card className="w-full max-w-2xl mx-auto mt-8">
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle>Active Torrents</CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={handleManualRefresh}
              disabled={isRefreshing}
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {torrents && torrents.length > 0 ? (
            <div className="space-y-4">
              {torrents.map((torrent, index) => (
                <div key={index} className="space-y-2 border-b pb-4 last:border-b-0">
                  <div className="flex justify-between">
                    <span className="font-medium">Name:</span>
                    <span>{torrent.name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-medium">Progress:</span>
                    <span>{torrent.percentDone.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-medium">Speed:</span>
                    <span>{(torrent.rateDownload / (1024 * 1024)).toFixed(2)} MB/s</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-medium">Status:</span>
                    <span>{torrent.status}</span>
                  </div>

                  <div className="w-full bg-gray-200 rounded-full h-2.5">
                    <div
                      className="bg-blue-600 h-2.5 rounded-full transition-all duration-300"
                      style={{ width: `${torrent.percentDone}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center text-gray-500">No active torrents</p>
          )}
        </CardContent>
      </Card>
    );
  };
