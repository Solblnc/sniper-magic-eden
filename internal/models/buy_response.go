package models

// Response - struct of a response
type Response struct {
	Tx struct {
		Type string `json:"type"`
		Data []byte `json:"data"`
	} `json:"tx"`

	TxSigned struct {
		Type string `json:"type"`
		Data []byte `json:"data"`
	} `json:"tx_signed"`
}
