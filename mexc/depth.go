package mexc

import (
	"encoding/json"
	"fmt"
	"mexccrypto/types"
	"net/http"
	"time"
)

type DepthResponse struct {
	Data struct {
		Asks [][]float64 `json:"asks"`
		Bids [][]float64 `json:"bids"`
	} `json:"data"`
}

func FetchOrderBook(symbol string, depth int) (asks, bids []types.Order, err error) {
	url := fmt.Sprintf("https://contract.mexc.com/api/v1/contract/depth/%s?limit=%d", symbol, depth)
	client := &http.Client{Timeout: 5 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	var d DepthResponse
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return nil, nil, err
	}

	for _, a := range d.Data.Asks {
		asks = append(asks, types.Order{Price: a[0], Amount: a[1]})
	}
	for _, b := range d.Data.Bids {
		bids = append(bids, types.Order{Price: b[0], Amount: b[1]})
	}

	return asks, bids, nil
}

func VWAP(orders []types.Order, targetUSD float64, side string) (float64, error) {
	var totalCost, totalAmount float64

	for _, o := range orders {
		orderValue := o.Price * o.Amount
		remaining := targetUSD - totalCost

		if remaining <= 0 {
			break
		}

		if orderValue >= remaining {
			partialAmount := remaining / o.Price
			totalCost += partialAmount * o.Price
			totalAmount += partialAmount
		} else {
			totalCost += orderValue
			totalAmount += o.Amount
		}
	}

	if totalAmount == 0 {
		return 0, fmt.Errorf("not enough liquidity on %s side for %.2f USD", side, targetUSD)
	}

	return totalCost / totalAmount, nil
}
