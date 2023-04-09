package models

// Stats - response (description) of a nft token
type Stats struct {
	Symbol      string  `json:"symbol"`
	FloorPrice  float64 `json:"floorPrice"`
	ListedCount int     `json:"listedCount"`
	VolumeAll   float64 `json:"volumeAll"`
}
