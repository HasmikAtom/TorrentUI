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

export interface FoundRuTorrents {
  author: string;
  category:    string;
  dateAdded: string;
  downloadURL: string;
  downloads: string;
  id: string;
  le: string;
  se: string;
  magnetLink: string;
  size: string;
  title: string
}
