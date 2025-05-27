package main

type Config struct {
	AppPort              string
	TransmissionHost     string
	TransmissionPort     string
	TransmissionUsername string
	TransmissionPassword string
	RutrackerUrl         string
	ThepiratebayURL      string
	RutrackerUsername    string
	RutrackerPassword    string
}

type TorrentStatus struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	PercentDone  float64 `json:"percentDone"`
	RateDownload int64   `json:"rateDownload"`
	Status       string  `json:"status"`
	Error        int     `json:"error"`
	ErrorString  string  `json:"errorString"`
}
