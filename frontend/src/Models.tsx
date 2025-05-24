export interface TorrentStatus {
  name: string;
  percentDone: number;
  rateDownload: number;
  status: string;
}

export interface FoundTorrents {
  title:       string;
  magnet:      string;
  upload_date:  string;
  size:        string;
  seeders:     number;
  leechers:    number;
  category:    string;
  uploader:    string;
  torrent_link: string;
}

