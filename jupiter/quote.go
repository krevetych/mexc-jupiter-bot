package jupiter

import (
	"encoding/json"
	"errors"
	"fmt"
	"mexccrypto/types"
	"net/http"
	"sync"
	"time"
)

const jupiterQuoteURL = "https://lite-api.jup.ag/price/v2"

var (
	cache           map[string]float64
	cacheTimestamp  map[string]time.Time
	lastRequestTime time.Time
	throttleMutex   = &sync.Mutex{}
)

func init() {
	cache = make(map[string]float64)
	cacheTimestamp = make(map[string]time.Time)
}

type priceInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Price string `json:"price"`
}

type PriceResponse struct {
	Data map[string]priceInfo `json:"data"`
}

func FetchPrice(mexcSymbol, splToken string, cfg *types.Config) (float64, error) {
	if cachedPrice, found := cache[splToken]; found {
		if time.Since(cacheTimestamp[splToken]) < time.Duration(cfg.JupiterQuoteIntervalSec)*time.Second {
			return cachedPrice, nil
		}
	}

	throttleMutex.Lock()
	defer throttleMutex.Unlock()

	elapsed := time.Since(lastRequestTime)
	wait := time.Duration(cfg.JupiterQuoteIntervalSec)*time.Second - elapsed

	if wait > 0 {
		time.Sleep(wait)
	}

	url := fmt.Sprintf("%s?ids=%s&vsToken=Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", jupiterQuoteURL, splToken)

	client := &http.Client{Timeout: 5 * time.Second}
	var outAmount float64

	for attempt := 1; attempt <= 5; attempt++ {
		res, err := client.Get(url)
		if err != nil {
			return 0, err
		}
		defer res.Body.Close()

		lastRequestTime = time.Now()

		if res.StatusCode == http.StatusOK {
			var pr PriceResponse
			if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
				return 0, err
			}

			info, ok := pr.Data[splToken]
			if !ok {
				return 0, fmt.Errorf("no price data found for %s in response", splToken)
			}

			_, err := fmt.Sscanf(info.Price, "%f", &outAmount)
			if err != nil {
				return 0, fmt.Errorf("failed to parse price for %s: %w", splToken, err)
			}

			cache[splToken] = outAmount
			cacheTimestamp[splToken] = time.Now()

			return outAmount, nil
		}

		if res.StatusCode == http.StatusTooManyRequests {
			waitTime := time.Duration(attempt*attempt) * time.Second
			fmt.Printf("Received 429. Retrying in %v...\n", waitTime)
			time.Sleep(waitTime)
		} else {
			return 0, fmt.Errorf("bad status: %s", res.Status)
		}
	}

	return 0, errors.New("failed to fetch price after 5 attempts")
}
