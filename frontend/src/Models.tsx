export interface TorrentStatus {
  name: string;
  percentDone: number;
  rateDownload: number;
  status: string;
}

export interface FoundTorrents {
  id:          string;
  title:       string;
  category:    string;
  uploader:    string;
  size:        string;
  upload_date:  string;
  se:     number;
  le:    number;
  description_url: string;
  
  magnet:      string;
}

export interface FoundRuTorrents {
  id:          string;
  title:       string;
  category:    string;
  uploader:    string;
  size:        string;
  upload_date:  string;
  se:     number;
  le:    number;
  description_url: string;
  
  downloadURL: string;
  
  downloads: string;
}
