package types

type Config struct {
	SpreadThresholdPercent  float64 `yaml:"spread_threshold_percent"`
	SpreadPrecision         float64 `yaml:"spread_precision"`
	ContractDepth           int     `yaml:"contract_depth"`
	VWAP                    float64 `yaml:"vwap"`
	Volume24hMin            float64 `yaml:"volume_24h_min"`
	JupiterQuoteIntervalSec int     `yaml:"jupiter_quote_interval_sec"`
	TelegramBotToken        string  `yaml:"telegram_bot_token"`
	TelegramChatIDs          []int64     `yaml:"telegram_chat_ids"`
}

type PriceInfo struct {
	Symbol    string
	Ask       float64
	Bid       float64
	Volume24h float64
}

type Order struct {
	Price  float64
	Amount float64
}
