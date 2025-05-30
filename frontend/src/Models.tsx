export interface TorrentStatus {
  name: string;
  percentDone: number;
  rateDownload: number;
  status: string;
}

export interface ScrapedTorrents {
  id:          string;
  title:       string;
  category:    string;
  uploader:    string;
  size:        string;
  upload_date:  string;
  se:     number;
  le:    number;
  description_url: string;
  
  magnet:      string; // for piratebay
  download_url: string; // for rutracker

  downloads: string;
}
