import { TorrentDownloader } from "./TorrentDownloader";
import { TorrentList } from "./TorrentList";

const TorrentUI: React.FC = () => {
  return (
    <div className="space-y-4">
      <TorrentDownloader />
      <TorrentList />
    </div>
  );
};

export default TorrentUI;