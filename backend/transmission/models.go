package transmission

import "net/http"

type TransmissionRPC struct {
	URL      string
	Username string
	Password string
	Session  string
	Client   *http.Client
}

type RPCRequest struct {
	Method    string `json:"method"`
	Arguments any    `json:"arguments"`
	Tag       int    `json:"tag"`
}

type RPCResponse struct {
	Result    string         `json:"result"`
	Arguments map[string]any `json:"arguments"`
	Tag       int            `json:"tag"`
}
