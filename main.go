package main

import (
	"fmt"
	"log"
	"math"
	"mexccrypto/internal/config"
	"mexccrypto/jupiter"
	"mexccrypto/mexc"
	"mexccrypto/tg"
	"mexccrypto/types"
	"time"
)

type lastState struct {
	price    float64
	spread   float64
	loggedAt time.Time
}

var lastLogged = make(map[string]lastState)

func shouldLog(symbol string, newPrice, newSpread float64, cfg *types.Config, cooldown time.Duration) bool {
	last, ok := lastLogged[symbol]
	if ok {
		samePrice := math.Abs(last.price-newPrice) < 1e-8
		spreadDelta := math.Abs(last.spread - newSpread)
		recent := time.Since(last.loggedAt) < cooldown

		if samePrice && spreadDelta < cfg.SpreadPrecision && recent {
			return false
		}
	}

	lastLogged[symbol] = lastState{
		price:    newPrice,
		spread:   newSpread,
		loggedAt: time.Now(),
	}

	return true
}

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	splMap, err := config.LoadSPLMap("spl.json")
	if err != nil {
		log.Fatalf("load SPL map: %v", err)
	}

	priceCh := make(chan types.PriceInfo)
	go mexc.RunWS(priceCh, cfg, splMap)

	const cooldown = 10 * time.Second

	for price := range priceCh {
		splToken, ok := splMap[price.Symbol]
		if !ok {
			continue
		}

		asks, _, err := mexc.FetchOrderBook(price.Symbol, cfg.ContractDepth)
		if err != nil {
			log.Printf("orderbook fetch error for %s: %v", price.Symbol, err)
			continue
		}

		vwapAsk, err := mexc.VWAP(asks, cfg.VWAP, "ask")
		if err != nil {
			log.Printf("VWAP calc error for %s: %v", price.Symbol, err)
			continue
		}

		jupPrice, err := jupiter.FetchPrice(price.Symbol, splToken, cfg)
		if err != nil || jupPrice == 0 {
			log.Printf("jupiter error for %s: %v", price.Symbol, err)
			continue
		}

		spread := 100 * math.Abs(vwapAsk-jupPrice) / ((vwapAsk + jupPrice) / 2)
		if spread >= cfg.SpreadThresholdPercent {
			if shouldLog(price.Symbol, jupPrice, spread, cfg, cooldown) {
				log.Printf("[ARBITRAGE] %s: MEXC VWAPAsk %.8f | Jupiter %.8f | Spread %.2f%% | Volume %.2fM",
					price.Symbol, vwapAsk, jupPrice, spread, price.Volume24h/1_000_000)
				msg := fmt.Sprintf(
					`<b>ARBITRAGE ALERT</b>
<code>%s</code>
<b>MEXC VWAPAsk:</b> <code>%.8f</code>
<b>Jupiter:</b> <code>%.8f</code>
<b>Spread:</b> %.2f%%
<b>Volume:</b> %.2fM

<b>MEXC:</b> <code>https://www.mexc.com/ru-RU/futures/%s?_from=search</code>
<b>Jupiter:</b> <code>%s</code>`,
					price.Symbol,
					vwapAsk,
					jupPrice,
					spread,
					price.Volume24h/1_000_000,
					price.Symbol,
					splToken,
				)

				for _, chatID := range cfg.TelegramChatIDs {
					err := tg.SendMessage(cfg.TelegramBotToken, chatID, msg)
					if err != nil {
						log.Printf("telegram send error to %d: %v", chatID, err)
					}
				}
			}
		}
	}
}
