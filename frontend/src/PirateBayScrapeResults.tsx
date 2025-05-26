import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { RotateCw, Search, Delete } from 'lucide-react';
import { FoundTorrents} from './Models';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Download } from 'lucide-react';
import { Dialog, DialogTrigger, DialogContent, DialogTitle, DialogDescription, DialogClose } from '@/components/ui/dialog';
import { Label } from '@radix-ui/react-label';
import { DialogHeader, DialogFooter } from './components/ui/dialog';
import { RadioGroup, RadioGroupItem } from './components/ui/radio-group';


interface ScrapeResultsProps {
  switchTab: (tabValue: string) => void;
}

export const PirateBayScrapeResults: React.FC<ScrapeResultsProps> = ({ switchTab }) => {

    const [searchLoading, setSearchLoading] = useState<boolean>(false);
    const [_, setDownloadLoading] = useState<boolean>(false);
    const [mediaTypeSelected, setMediaTypeSelected] = useState<boolean>(false);
    const [selectedTorrent, setSelectedTorrent] = useState<string>("");
    const [torrentName, setTorrentName] = useState<string>("");
    const [foundTorrents, setFoundTorrents] = useState<FoundTorrents[] | null>(null);
    const [contentType, setContentType] = useState<string>('Movie');


    const handleTorrentSearch = async () => {
        setSearchLoading(true);

        try {
            const response = await fetch(`/api/scrape/piratebay/${torrentName}`, {
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

    const selectTorrent = (mediaType: string, selectedMagnet: string) => {
      // console.log("Selected stuff ===> ", mediaType, selectedMagnet)
      setMediaTypeSelected(true);


      setContentType(mediaType)
      setSelectedTorrent(selectedMagnet)
    }

    const handleTorrentDownload = async () => {
      // console.log("MEDIA TYPE AND MAGNET LINK ===> ", contentType, selectedTorrent)
      setDownloadLoading(true);

      try {
        const formData = new FormData();
        if (selectedTorrent) {
          formData.append('magnetLink', selectedTorrent);
        }
        formData.append('contentType', contentType);

        const response = await fetch('/api/download', {
          method: 'POST',
          body: formData,
        });

        const data = await response.json();
        if (response.ok) {
          console.log(data)
        } else {
          console.error('Download failed:', data.error);
        }
      } catch (error) {
        console.error('Error:', error);
      }


      setDownloadLoading(false);

      // wait a few seconds before switching the tab
      switchTab("download");
      // setMediaType("")
    }

    // const onClose = (e:any) => {
    //   console.log("on open close", e)
    //   setContentType("")
    // }

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
                          <Dialog onOpenChange={() => setContentType("")}>
                            <DialogTrigger asChild>
                              <Button
                                size="sm"
                                variant="outline"
                                className="flex items-center gap-1"
                              >
                                <Download size={16} />
                                <span>Download</span>
                              </Button>
                            </DialogTrigger>
                            <DialogContent className="sm:max-w-sm">
                              <DialogHeader>
                                <DialogTitle>Select Media Type</DialogTitle>
                                <DialogDescription>
                                  {/* Anyone who has this link will be able to view this. */}
                                </DialogDescription>
                              </DialogHeader>
                              <div className="flex items-center space-x-2">
                                <div className="grid flex-1 gap-2">
                                  <RadioGroup
                                    value={contentType}
                                    onValueChange={(selected) => selectTorrent(selected, torrent.magnet)}
                                    className="flex justify-between mb-4"
                                  >
                                    <div className="flex items-center space-x-2">
                                      <RadioGroupItem value="Movies" id="movie" />
                                      <Label htmlFor="movie">Movie</Label>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                      <RadioGroupItem value="Series" id="series" />
                                      <Label htmlFor="series">Series</Label>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                      <RadioGroupItem value="Music" id="music" />
                                      <Label htmlFor="music">Music</Label>
                                    </div>
                                  </RadioGroup>
                                </div>
                              </div>
                              <DialogFooter className="sm:justify-start">
                                <DialogClose asChild>
                                  <Button
                                    disabled={!mediaTypeSelected}
                                    onClick={handleTorrentDownload}
                                    >
                                    <Download />
                                    Download
                                  </Button>
                                </DialogClose>
                              </DialogFooter>
                            </DialogContent>
                          </Dialog>
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