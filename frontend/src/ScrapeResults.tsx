import React, { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { RotateCw, Search, Delete } from 'lucide-react';
import { FoundTorrents} from './Models';
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Download } from 'lucide-react';

export const ScrapeResults: React.FC = () => {

    const [searchLoading, setSearchLoading] = useState<boolean>(false);
    const [torrentName, setTorrentName] = useState<string>("")
    const [foundTorrents, setFoundTorrents] = useState<FoundTorrents[] | null>(null)


    const handleTorrentSearch = async () => {
        setSearchLoading(true);

        try {
            const response = await fetch(`/api/scrape/${torrentName}`, {
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

    const handleTorrentSearchClear = async () => {
        setSearchLoading(false);
        setFoundTorrents(null);
        setTorrentName("");
    }

    const formatHeader = (header: string) => {
        return header
          .replace(/([A-Z])/g, ' $1')
          .replace(/_/g, ' ')
          .replace(/\b\w/g, (char) => char.toUpperCase());
    };

    const getVisibleColumns = () => {
        if (!foundTorrents || foundTorrents.length === 0) return [];
        return Object.keys(foundTorrents[0]).filter(
        key => key !== "magnet" && key !== "torrent_link"
        );
    };

    return (
        <Card className="w-full max-w-2xl mx-auto mt-8">
          <CardHeader>
            <CardTitle>Transmission Download Manager</CardTitle>
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
            {/* <div className="h-[400px] overflow-y-auto scrollbar scrollbar-thumb-gray-400 scrollbar-track-gray-100"> */}
            <div >
              <Table>
                {/* <TableCaption>Available Torrents</TableCaption> */}
                <TableHeader>
                  <TableRow>
                    {getVisibleColumns().map((column) => (
                        <TableHead key={column} className="font-medium">
                        {formatHeader(column)}
                        </TableHead>
                    ))}
                    {getVisibleColumns().length ? (<TableHead>Actions</TableHead>) : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {foundTorrents && foundTorrents.map((torrent) => (
                    <TableRow key={torrent.torrent_link} className="hover:bg-slate-100">
                      {Object.entries(torrent).map(([key, value], i) => {
                        if (key === "magnet" || key === "torrent_link") return null;

                        return (
                          <TableCell key={`${i}-${key}`}>
                            {key === "se" || key === "le" ? (
                              <span className={`font-medium ${key === "se" ? "text-green-600" : "text-red-600"}`}>
                                {value}
                              </span>
                            ) : (
                              value
                            )}
                          </TableCell>
                        );
                      })}
                      <TableCell>
                        <div className="flex space-x-2">
                          <Button
                            size="sm"
                            variant="outline"
                            className="flex items-center gap-1"
                          >
                            <Download size={16} />
                            <span>Download</span>
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
    )
}