package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/redis/go-redis/v9"
)

func UpdateFuturesTokens(ctx context.Context, redisClient *redis.Client, splMap map[string]string, threshold float64) error {
	res, err := http.Get("https://contract.mexc.com/api/v1/contract/ticker")
	if err != nil {
		return fmt.Errorf("fetch future ticker: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var response struct {
		Data []struct {
			Symbol string  `json:"symbol"`
			Volume float64 `json:"volume24"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	key := "spl_future_active_tokens"
	active := make(map[string]string)

	for _, ticker := range response.Data {
		spl, ok := splMap[ticker.Symbol]
		if !ok || ticker.Volume < threshold {
			continue
		}
		active[ticker.Symbol] = spl
	}

	if len(active) == 0 {
		return nil
	}

	if err := redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	fields := make(map[string]interface{})
	for k, v := range active {
		fields[k] = v
	}

	if err := redisClient.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("redis hset error: %w", err)
	}

	return nil
}

