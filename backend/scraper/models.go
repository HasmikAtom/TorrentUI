package scraper

type PirateBayTorrent struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Uploader       string `json:"uploader"`
	Size           string `json:"size"`
	UploadDate     string `json:"upload_date"`
	Se             int    `json:"se"`
	Le             int    `json:"le"`
	DescriptionURL string `json:"description_url"`

	Magnet string `json:"magnet"`
}

type RutrackerTorrent struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Uploader       string `json:"uploader"`
	Size           string `json:"size"`
	UploadDate     string `json:"upload_date"`
	Se             string `json:"se"`
	Le             string `json:"le"`
	DescriptionURL string `json:"description_url"`

	DownloadURL string `json:"download_url"`

	Downloads string `json:"downloads"`
}
