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
	Method    string      `json:"method"`
	Arguments interface{} `json:"arguments"`
	Tag       int         `json:"tag"`
}

type RPCResponse struct {
	Result    string                 `json:"result"`
	Arguments map[string]interface{} `json:"arguments"`
	Tag       int                    `json:"tag"`
}
