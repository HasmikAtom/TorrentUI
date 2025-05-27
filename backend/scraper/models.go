package scraper

type PirateBayTorrent struct {
	Title       string `json:"title"`
	Magnet      string `json:"magnet"`
	UploadDate  string `json:"upload_date"`
	Size        string `json:"size"`
	Se          int    `json:"se"`
	Le          int    `json:"le"`
	Category    string `json:"category"`
	Uploader    string `json:"uploader"`
	TorrentLink string `json:"torrent_link"`
}

type RutrackerTorrent struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Size        string `json:"size"`
	DownloadURL string `json:"downloadURL"`
	Se          string `json:"se"`
	Le          string `json:"le"`
	Downloads   string `json:"downloads"`
	DateAdded   string `json:"dateAdded"`
	MagnetLink  string `json:"magnetLink"`
}
