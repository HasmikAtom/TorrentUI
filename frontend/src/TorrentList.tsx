import { useState, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "./components/ui/card";
import { TorrentStatus } from "./Models";



export const TorrentList: React.FC = () => {
    const [torrents, setTorrents] = useState<TorrentStatus[] | null>(null);
  
    const fetchTorrents = async () => {
      try {
        const response = await fetch(`/api/torrents`);
        const data = await response.json();
        if (response.ok) {
          setTorrents(data);
        }
      } catch (error) {
        console.error('Error fetching torrents:', error);
      }
    };
  
    useEffect(() => {
      fetchTorrents();
    }, []);
  
    return (
      <Card className="w-full max-w-2xl mx-auto mt-8">
        <CardHeader>
          <CardTitle>Active Torrents</CardTitle>
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
  
  