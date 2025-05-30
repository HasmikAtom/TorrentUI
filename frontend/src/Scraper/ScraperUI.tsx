import React, { useState } from 'react';
import { ScrapedTorrents } from '../Models';
import { ScrapeSearch } from './ScrapeSearch';
import { ScrapedTorrentsCards } from './ScrapedTorrents';



const ScraperConfig = {
  thepiratebay: {
    scrapeEndpoint: '/api/scrape/piratebay/',
    downloadEndpoint: '/api/download',
    downloadKey: 'magnetLink',
    downloadSource: 'magnet' as const,
  },
  rutracker: {
    scrapeEndpoint: '/api/scrape/rutracker/',
    downloadEndpoint: '/api/download/file',
    downloadKey: 'url',
    downloadSource: 'download_url' as const,
  }
} as const

// TODO: do this the proper way, create a type for the piratebay and rutracker config
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
    const [_, setDownloadLoading] = useState<boolean>(false);
    const [mediaTypeSelected, setMediaTypeSelected] = useState<boolean>(false);
    const [selectedTorrent, setSelectedTorrent] = useState<string>("");
    const [torrentName, setTorrentName] = useState<string>("");
    const [foundTorrents, setFoundTorrents] = useState<ScrapedTorrents[] | null>(null);
    const [contentType, setContentType] = useState<string>('Movie');
  

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

    const handleScrapeDownload = async () => {
      setDownloadLoading(true);
      try {
        const formData = new FormData();
        if (selectedTorrent) {
          formData.append(config.downloadKey, selectedTorrent);
        }
        formData.append('contentType', contentType);

        const response = await fetch(config.downloadEndpoint, {
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
    

      const handleTorrentSearchClear = async () => {
      setSearchLoading(false);
      setFoundTorrents(null);
      setTorrentName("");
    }
  
    const selectTorrent = (mediaType: string, selectedMagnet: string) => {
      console.log("selecting torrent", selectedMagnet)
      setMediaTypeSelected(true);
      setContentType(mediaType)
      setSelectedTorrent(selectedMagnet)
    }
  

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
        contentType={contentType}
        mediaTypeSelected={mediaTypeSelected}
        downloadSource={downloadSource}
        setContentType={setContentType}
        selectTorrent={selectTorrent}
        handleTorrentDownload={handleScrapeDownload}
      />
    </>
  );
}