package main

import (
	"context"
	"log"
	"mexccrypto/internal/config"
	"mexccrypto/jupiter"
	"mexccrypto/mexc"
	"mexccrypto/types"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func runLoop(ctx context.Context, redisClient *redis.Client, cfg *types.Config) {
	ticker := time.NewTicker(500 * time.Millisecond)
	spreadLogger := config.NewSpreadLogger(cfg.SpreadPrecision)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mexcPrices, err := mexc.FetchMexcFuturesPrices(ctx, redisClient, cfg)
			if err != nil {
				log.Printf("[Mexc] Fetch price error: %v", err)
				continue
			}

			jupPrices, err := jupiter.FetchAllPrices(ctx, redisClient, cfg)
			if err != nil {
				log.Printf("[Jupiter] Fetch price error: %v", err)
				continue
			}

			spreadLogger.CompareAndPrintSpreads(mexcPrices, jupPrices, cfg)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	splMap, err := config.LoadSPLMap("spl.json")
	if err != nil {
		log.Fatalf("load SPL map: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	c := cron.New(cron.WithLocation(time.UTC))
	_, err = c.AddFunc("0 3 * * *", func() {
		if err := mexc.UpdateFuturesTokens(ctx, redisClient, splMap, cfg.Volume24hMin); err != nil {
			log.Printf("[Mexc] update futures tokens error: %v", err)
		} else {
			log.Printf("[Mexc] update futures tokens success")
		}
	})

	if err != nil {
		log.Fatalf("failed to schedule daily update: %v", err)
	}
	c.Start()

	if err := mexc.UpdateFuturesTokens(ctx, redisClient, splMap, cfg.Volume24hMin); err != nil {
		log.Fatalf("update futures tokens error: %v", err)
	}

	go runLoop(ctx, redisClient, cfg)

	select {}

}
