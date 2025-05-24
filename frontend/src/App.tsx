import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";
import { ScrapeResults } from "./ScrapeResults"
// import { TorrentTabs } from "./TorrentTabs";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"


const TorrentUI: React.FC = () => {
  return (
    // <div className="space-y-4">
    //   <ScrapeResults />
    //   <TorrentDownloader />
    //   {/* <TorrentTabs /> */}
    //   <TorrentList />
    // </div>


    // <Tabs defaultValue="download" className="w-[400px]">
    <Tabs defaultValue="download">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="download">Download</TabsTrigger>
        <TabsTrigger value="scrape">Find</TabsTrigger>
      </TabsList>
      <TabsContent value="download">
        <TorrentDownloader />
        <TorrentList />
      </TabsContent>
      <TabsContent value="scrape">
        <ScrapeResults />
      </TabsContent>
    </Tabs>
  );
};

export default TorrentUI;