package collections

import (
	"Sniper-Magic-Eden/internal/models"
	"encoding/json"
	"os"
)

func LoadCollections() (map[string]*models.Token, error) {
	var collections map[string]*models.Token
	req, _ := os.ReadFile("./data/collections.json")
	err := json.Unmarshal(req, &collections)
	if err != nil {
		return nil, err
	}

	return collections, nil
}
