import React, { useState } from 'react';
import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";
import { ScrapeResults } from "./ScrapeResults"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"



const TorrentUI: React.FC = () => {
  const [activeTab, setActiveTab] = useState("download");

  const switchTab = (tabName: string) => {
    setActiveTab(tabName)
  }

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className='pt-[20px]'>
      <TabsList className="grid w-[400px] grid-cols-2 mx-auto">
        <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white' value="download">Download</TabsTrigger>
        <TabsTrigger className='data-[state=active]:bg-black data-[state=active]:text-white' value="scrape">Find</TabsTrigger>
      </TabsList>
      <TabsContent value="download">
        <TorrentDownloader />
        <TorrentList />
      </TabsContent>
      <TabsContent value="scrape">
        <ScrapeResults switchTab={switchTab}/>
      </TabsContent>
    </Tabs>
  );
};

export default TorrentUI;