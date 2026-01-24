import React, { useState, useEffect, useRef, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "./components/ui/card";
import { Button } from "./components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { RefreshCw, X, Trash2 } from "lucide-react";
import { TorrentStatus } from "./Models";
import { useToast } from "@/hooks/use-toast";

const POLL_INTERVAL = 3000;
const BYTES_PER_MB = 1024 * 1024;

interface Props {
  refreshTrigger?: number;
}

export const TorrentList: React.FC<Props> = React.memo(({ refreshTrigger }) => {
    const [torrents, setTorrents] = useState<TorrentStatus[] | null>(null);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
    const intervalRef = useRef<number | null>(null);
    const { toast } = useToast();

    const fetchTorrents = useCallback(async (showError = false) => {
      try {
        const response = await fetch(`/api/torrents`);
        const data = await response.json();
        if (response.ok) {
          setTorrents(data);
        } else if (showError) {
          toast({
            variant: "destructive",
            title: "Failed to fetch torrents",
            description: data.error || "Could not load torrent list",
          });
        }
      } catch (error) {
        if (showError) {
          toast({
            variant: "destructive",
            title: "Connection error",
            description: error instanceof Error ? error.message : "Failed to connect to server",
          });
        }
      }
    }, [toast]);

    const handleManualRefresh = async () => {
      setIsRefreshing(true);
      await fetchTorrents(true);
      setIsRefreshing(false);
    };

    const handleDelete = async (id: number, deleteData: boolean) => {
      try {
        const response = await fetch(`/api/torrents/${id}?deleteData=${deleteData}`, {
          method: 'DELETE',
        });
        if (response.ok) {
          await fetchTorrents();
          toast({
            title: deleteData ? "Torrent deleted" : "Torrent removed",
            description: deleteData ? "Torrent and files have been deleted" : "Torrent removed from list",
          });
        } else {
          const data = await response.json();
          toast({
            variant: "destructive",
            title: "Delete failed",
            description: data.error || "Could not delete torrent",
          });
        }
      } catch (error) {
        toast({
          variant: "destructive",
          title: "Connection error",
          description: error instanceof Error ? error.message : "Failed to connect to server",
        });
      }
    };

    const handleConfirmDelete = async () => {
      if (deleteConfirmId !== null) {
        await handleDelete(deleteConfirmId, true);
        setDeleteConfirmId(null);
      }
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
      <>
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
              {torrents.map((torrent) => (
                <div key={torrent.id} className="space-y-2 border-b pb-4 last:border-b-0">
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
                    <span>{(torrent.rateDownload / BYTES_PER_MB).toFixed(2)} MB/s</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-medium">Status:</span>
                    <span>{torrent.status}</span>
                  </div>

                  <div
                    role="progressbar"
                    aria-valuenow={torrent.percentDone}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-label={`Download progress for ${torrent.name}`}
                    className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5"
                  >
                    <div
                      className="bg-blue-600 h-2.5 rounded-full transition-all duration-300"
                      style={{ width: `${torrent.percentDone}%` }}
                    />
                  </div>

                  <div className="flex gap-2 pt-2 justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDelete(torrent.id, false)}
                      aria-label={`Remove ${torrent.name} from list`}
                    >
                      <X className="w-4 h-4 mr-1" />
                      Remove
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="bg-slate-700 text-white hover:bg-slate-800 border-slate-700"
                      onClick={() => setDeleteConfirmId(torrent.id)}
                      aria-label={`Delete ${torrent.name} and its files`}
                    >
                      <Trash2 className="w-4 h-4 mr-1" />
                      Delete
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center text-gray-500">No active torrents</p>
          )}
        </CardContent>
      </Card>

      <Dialog open={deleteConfirmId !== null} onOpenChange={(open) => !open && setDeleteConfirmId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Torrent and Files</DialogTitle>
            <DialogDescription>
              This will permanently delete the torrent and all downloaded files from your disk. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
              Cancel
            </Button>
            <Button
              className="bg-slate-700 text-white hover:bg-slate-800"
              onClick={handleConfirmDelete}
            >
              I Understand, Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
    );
  });
