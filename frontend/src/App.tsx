import React, { useState } from 'react';
import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScraperUI } from "./Scraper/ScraperUI"


const TorrentUI: React.FC = () => {
  const [activeTab, setActiveTab] = useState("download");

  const switchTab = (tabName: string) => {
    setActiveTab(tabName)
  }

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className='pt-[20px]'>
      <TabsList className="grid w-[400px] grid-cols-3 mx-auto">
        <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white' value="download">Download</TabsTrigger>
        <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white' value="thepiratebay">The Pirate Bay</TabsTrigger>
        <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white' value="rutracker">Rutracker</TabsTrigger>
      </TabsList>
      <TabsContent value="download">
        <TorrentDownloader />
        <TorrentList />
      </TabsContent>
      <TabsContent value="thepiratebay">
        <ScraperUI type='thepiratebay' switchTab={switchTab}/>
      </TabsContent>
      <TabsContent value="rutracker">
        <ScraperUI type='rutracker' switchTab={switchTab}/>
      </TabsContent>
    </Tabs>
  );
};

export default TorrentUI;