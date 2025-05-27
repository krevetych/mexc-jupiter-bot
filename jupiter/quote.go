package jupiter

import (
	"context"
	"encoding/json"
	"fmt"
	"mexccrypto/internal/config"
	"mexccrypto/types"
	"net/http"

	"github.com/redis/go-redis/v9"
)

const jupiterQuoteURL = "https://lite-api.jup.ag/swap/v1/quote"

type priceInfo struct {
	ID    string `json:"id"`
	Price string `json:"price"`
}

type PriceResponse struct {
	Data map[string]priceInfo `json:"data"`
}

func FetchAllPrices(ctx context.Context, rdb *redis.Client, cfg *types.Config, splMap map[string]config.SPLInfo) (map[string]float64, error) {
	tokens, err := config.GetActiveTokens(ctx, rdb, "spl_future_active_tokens")
	if err != nil {
		return nil, fmt.Errorf("get active tokens: %v", err)
	}

	client := &http.Client{Timeout: 5 * 1e9}
	outputMint := "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"

	prices := make(map[string]float64)

	for mexcSym, splKey := range tokens {
		splInfo, ok := splMap[splKey]
		if !ok {
			continue
		}

		amount := fmt.Sprintf("%d", splInfo.Decimals)
		url := fmt.Sprintf("%s?inputMint=%s&outputMint=%s&amount=%s&slippage=1", jupiterQuoteURL, splInfo.Mint, outputMint, amount)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		res, err := client.Do(req)
		if err != nil || res.StatusCode != http.StatusOK {
			continue
		}

		var pr PriceResponse
		if err := json.NewDecoder(res.Body).Decode(&pr)
	}
}
