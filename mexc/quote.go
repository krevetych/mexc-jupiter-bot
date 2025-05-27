package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"mexccrypto/internal/config"
	"mexccrypto/types"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type FuturesTicker struct {
	Symbol string `json:"symbol"`
	Price  float64 `json:"lastPrice"`
}

type FuturesResponse struct {
	Data []FuturesTicker `json:"data"`
}

func FetchMexcFuturesPrices(ctx context.Context, rdb *redis.Client, cfg *types.Config) (map[string]float64, error) {
	tokens, err := config.GetActiveTokens(ctx, rdb, "spl_future_active_tokens")
	if err != nil {
		return nil, fmt.Errorf("failed to get active tokens: %w", err)
	}

	resp, err := http.Get("https://contract.mexc.com/api/v1/contract/ticker")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mexc futures: %w", err)
	}
	defer resp.Body.Close()

	var response FuturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	symbolSet := make(map[string]bool, len(tokens))
	for symbol := range tokens {
		symbolSet[symbol] = true
	}

	price := make(map[string]float64)
	for _, t := range response.Data {
		if symbolSet[t.Symbol] {
			price[t.Symbol] = t.Price
		}
	}

	return price, nil
}
