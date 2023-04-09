package sniper

import (
	"Sniper-Magic-Eden/internal/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const storageTime = time.Duration(time.Second * 30)

var (
	client = &http.Client{}
	cache  = make(map[string]*models.Floor)
)

func GetFloor(symbol string) float64 {

	if _, ok := cache[symbol]; ok && time.Since(cache[symbol].Time) < storageTime {
		return -1
	} else {
		client = &http.Client{}
		req, _ := http.NewRequest("GET", fmt.Sprintf("https://api-mainnet.magiceden.dev/v2/collections/%s/stats", symbol), nil)

		res, _ := client.Do(req)

		defer res.Body.Close()

		body, _ := ioutil.ReadAll(res.Body)

		var floorResponse models.Stats
		json.Unmarshal(body, &floorResponse)

		cache[symbol] = &models.Floor{
			Value: floorResponse.FloorPrice / 1e9,
			Time:  time.Now(),
		}

	}
	return cache[symbol].Value

}
