import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { RotateCw, Search, Delete } from 'lucide-react';


interface props {
  torrentName: string;
  searchLoading: boolean;
  setTorrentName: (name: string) => void;
  handleTorrentSearch: () => void;
  handleTorrentSearchClear: () => void;
}

export const ScrapeSearch: React.FC<props> = ({
  torrentName,
  setTorrentName,
  handleTorrentSearch,
  searchLoading,
  handleTorrentSearchClear,
}) => {

  return (
    <Card className="w-full max-w-2xl mx-auto mt-8">
      <CardHeader>
        <CardTitle>Pirate Search</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex space-x-2 mb-4">
          <Input
            type="text"
            placeholder="Enter torrent name..."
            value={torrentName}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTorrentName(e.target.value)}
            className="flex-1"
          />
        </div>
        <div className="flex space-x-2 mb-4">
          <Button
            onClick={handleTorrentSearch}
            disabled={searchLoading}
          >
            {searchLoading ? (
              <RotateCw className="w-4 h-4 animate-spin" />
            ) : (
              <Search className="w-4 h-4" />
            )}
            <span className="ml-2">Search</span>
          </Button>

          <Button onClick={handleTorrentSearchClear}>
            <Delete className="w-4 h-4" />
            <span className="ml-2">Clear Search</span>
          </Button>
        </div>
      </ CardContent>
    </ Card>
      )
}