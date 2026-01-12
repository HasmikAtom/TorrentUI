import React, { useState } from 'react';
import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScraperUI } from "./Scraper/ScraperUI"
import { Toaster } from "@/components/ui/toaster"
import { Download, Skull, Compass } from "lucide-react"


const TorrentUI: React.FC = () => {
  const [activeTab, setActiveTab] = useState("download");
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const switchTab = (tabName: string) => {
    setActiveTab(tabName);
    if (tabName === "download") {
      setRefreshTrigger(prev => prev + 1);
    }
  }

  const handleTabChange = (tabName: string) => {
    setActiveTab(tabName);
    if (tabName === "download") {
      setRefreshTrigger(prev => prev + 1);
    }
  }

  return (
    <>
      <Tabs value={activeTab} onValueChange={handleTabChange} className='pt-[20px]'>
        <TabsList className="grid w-[500px] grid-cols-3 mx-auto">
          <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white flex items-center gap-2' value="download">
            <Download className="w-4 h-4" />
            Download
          </TabsTrigger>
          <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white flex items-center gap-2' value="thepiratebay">
            <Skull className="w-4 h-4" />
            Pirate Bay
          </TabsTrigger>
          <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white flex items-center gap-2' value="rutracker">
            <Compass className="w-4 h-4" />
            Rutracker
          </TabsTrigger>
        </TabsList>
        <TabsContent value="download">
          <TorrentDownloader onDownloadComplete={() => setRefreshTrigger(prev => prev + 1)} />
          <TorrentList refreshTrigger={refreshTrigger} />
        </TabsContent>
        <TabsContent value="thepiratebay">
          <ScraperUI type='thepiratebay' switchTab={switchTab}/>
        </TabsContent>
        <TabsContent value="rutracker">
          <ScraperUI type='rutracker' switchTab={switchTab}/>
        </TabsContent>
      </Tabs>
      <Toaster />
    </>
  );
};

export default TorrentUI;