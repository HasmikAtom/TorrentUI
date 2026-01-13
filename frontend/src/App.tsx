import React, { useState, useRef, useEffect } from 'react';
import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";
import { TabsList, TabsTrigger } from "@/components/ui/tabs"
import * as TabsPrimitive from "@radix-ui/react-tabs"
import { ScraperUI } from "./Scraper/ScraperUI"
import { Toaster } from "@/components/ui/toaster"
import { Download, Skull, Compass } from "lucide-react"

const TAB_ORDER = ["download", "thepiratebay", "rutracker"] as const;
type TabName = typeof TAB_ORDER[number];

const TorrentUI: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabName>("download");
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [slideDirection, setSlideDirection] = useState<'left' | 'right' | null>(null);
  const prevTabRef = useRef<TabName>("download");

  const getTabIndex = (tab: TabName) => TAB_ORDER.indexOf(tab);

  const switchTab = (tabName: string) => {
    const newTab = tabName as TabName;
    const prevIndex = getTabIndex(prevTabRef.current);
    const newIndex = getTabIndex(newTab);

    setSlideDirection(newIndex > prevIndex ? 'left' : 'right');
    prevTabRef.current = newTab;
    setActiveTab(newTab);

    if (newTab === "download") {
      setRefreshTrigger(prev => prev + 1);
    }
  }

  const handleTabChange = (tabName: string) => {
    switchTab(tabName);
  }

  // Reset slide direction after animation completes
  useEffect(() => {
    if (slideDirection) {
      const timer = setTimeout(() => setSlideDirection(null), 300);
      return () => clearTimeout(timer);
    }
  }, [slideDirection, activeTab]);

  return (
    <>
      <TabsPrimitive.Root value={activeTab} onValueChange={handleTabChange} className='pt-[20px]'>
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

        <div className="overflow-hidden">
          <div
            className={`transition-transform duration-300 ease-out ${
              slideDirection === 'left' ? 'animate-slide-from-right' :
              slideDirection === 'right' ? 'animate-slide-from-left' : ''
            }`}
          >
            <TabsPrimitive.Content value="download" className="mt-2 focus:outline-none">
              <TorrentDownloader onDownloadComplete={() => setRefreshTrigger(prev => prev + 1)} />
              <TorrentList refreshTrigger={refreshTrigger} />
            </TabsPrimitive.Content>
            <TabsPrimitive.Content value="thepiratebay" className="mt-2 focus:outline-none">
              <ScraperUI type='thepiratebay' switchTab={switchTab}/>
            </TabsPrimitive.Content>
            <TabsPrimitive.Content value="rutracker" className="mt-2 focus:outline-none">
              <ScraperUI type='rutracker' switchTab={switchTab}/>
            </TabsPrimitive.Content>
          </div>
        </div>
      </TabsPrimitive.Root>
      <Toaster />
    </>
  );
};

export default TorrentUI;