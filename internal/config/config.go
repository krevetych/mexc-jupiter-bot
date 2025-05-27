package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mexccrypto/types"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type SpreadLogger struct {
	lastSpread map[string]float64
	threshold  float64
}

type SPLInfo struct {
	Mint string `json:"mint"`
	Decimals string `json:"decimals"`
}

func LoadConfig(path string) (*types.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg types.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return &cfg, nil
}

func LoadSPLMap(path string) (map[string]SPLInfo, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spl map: %w", err)
	}

	if len(f) == 0 {
		return nil, errors.New("empty spl.json")
	}

	m := make(map[string]SPLInfo)
	if err := json.Unmarshal(f, &m); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return m, nil
}

func GetActiveTokens(ctx context.Context, rdb *redis.Client, key string) (map[string]string, error) {
	rawTokens, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active tokens from redis: %v", err)
	}

	tokens := make(map[string]string, len(rawTokens))
	for symbol, valueStr := range rawTokens {
		tokens[symbol] = valueStr
	}

	return tokens, nil
}

func NewSpreadLogger(threshold float64) *SpreadLogger {
	return &SpreadLogger{
		lastSpread: make(map[string]float64),
		threshold:  threshold,
	}
}

func (s *SpreadLogger) CompareAndPrintSpreads(mexcPrices, jupPrices map[string]float64, cfg *types.Config) {
	for symbol, priceMexc := range mexcPrices {
		priceJup, ok := jupPrices[symbol]
		if !ok {
			continue
		}

		spread := (priceMexc - priceJup) / priceJup * 100

		if spread <= cfg.SpreadPrecision {
			continue
		}

		last, found := s.lastSpread[symbol]
		if found {
			diff := spread - last
			if diff < 0 {
				diff = -diff
			}

			if diff < s.threshold {
				continue
			}
		}

		fmt.Printf("Spread for %s is %.2f%% (Mexc: %.8f, Jupiter: %.8f)\n",
			symbol, spread, priceMexc, priceJup)

		s.lastSpread[symbol] = spread
	}
}
