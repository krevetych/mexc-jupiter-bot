package mexc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"mexccrypto/types"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const mexcWSURL = "wss://contract.mexc.com"

type tickerData struct {
	Symbol    string  `json:"symbol"`
	LastPrice float64 `json:"lastPrice"`
	Bid1      float64 `json:"bid1"`
	Ask1      float64 `json:"ask1"`
	Volume24  float64 `json:"volume24"`
}

type wsMessage struct {
	Channel string     `json:"channel"`
	Data    tickerData `json:"data"`
	Ts      int64      `json:"ts"`
}

type subParams struct {
	Method string            `json:"method"`
	Param  map[string]string `json:"param"`
	ID     int64             `json:"id"`
}

func RunWS(out chan<- types.PriceInfo, cfg *types.Config, splMap map[string]string) {
	u := url.URL{Scheme: "wss", Host: "contract.mexc.com", Path: "/edge"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("MEXC dial error: %v", err)
		return
	}
	defer conn.Close()

	symbols := make([]string, 0, len(splMap))
	for symbol := range splMap {
		symbols = append(symbols, symbol)
	}

	for _, s := range symbols {
		sub := subParams{
			Method: "sub.ticker",
			Param:  map[string]string{"symbol": s},
			ID:     time.Now().UnixNano(),
		}

		if err := conn.WriteJSON(sub); err != nil {
			log.Fatalf("sub error: %v", err)
		}
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			sub := map[string]string{
				"method": "ping",
			}
			if err := conn.WriteJSON(sub); err != nil {
				log.Fatalf("ping error: %v", err)
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read error: %v", err)
			return
		}

		var reader io.Reader = bytes.NewReader(msg)
		if msg[0] == 0x1f && msg[1] == 0x8b {
			gzipReader, err := gzip.NewReader(reader)
			if err != nil {
				log.Printf("gzip error: %v", err)
				continue
			}
			defer gzipReader.Close()
			reader = gzipReader
		}

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			continue
		}

		var parsed wsMessage
		if err := json.Unmarshal(decompressed, &parsed); err != nil {
			continue
		}

		if parsed.Channel != "push.ticker" {
			continue
		}

		ticker := parsed.Data

		if ticker.Volume24 < cfg.Volume24hMin {
			continue
		}

		out <- types.PriceInfo{
			Symbol:    ticker.Symbol,
			Ask:       ticker.Ask1,
			Bid:       ticker.Bid1,
			Volume24h: ticker.Volume24,
		}
	}
}
