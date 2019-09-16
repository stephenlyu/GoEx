package binancefuture

import "github.com/shopspring/decimal"

type RateLimit struct {
	Interval string
	IntervalNum int
	Limit int
	RateLimitType string
}

type Filter struct {
	FilterType string
	MaxPrice decimal.Decimal
	MinPrice decimal.Decimal
	TickSize decimal.Decimal
	MaxQty decimal.Decimal
	MinQty decimal.Decimal
	StepSize decimal.Decimal
	Limit int
	MultiplierDown decimal.Decimal
	MultiplierUp decimal.Decimal
	MultiplierDecimal decimal.Decimal
}

type Symbol struct {
	Filters []Filter
	MaintMarginPercent decimal.Decimal
	PricePrecision int
	QuantityPrecision int
	RequiredMarginPercent decimal.Decimal
	Status string
	OrderType []string
	BaseAsset string
	QuoteAsset string
	Symbol string
	TimeInForce []string
}

type Exchange struct {
	Code            int
	Msg             string

	ExchangeFilters []Filter
	RateLimits      []RateLimit
	ServerTime      int64
	Symbols         []Symbol
	TimeZone        string
}

type DepthData struct {
	Code int
	LastUpdateId int64
	Asks [][]decimal.Decimal
	Bids [][]decimal.Decimal
}

type DepthUpdate struct {
	Event string 					`json:"e"`
	EventTs int64 					`json:"E"`
	Symbol string 					`json:"s"`
	UFirst int64 					`json:"U"`
	ULast int64 					`json:"u"`
	PrevU int64 					`json:"pu"`
	Bids [][]decimal.Decimal		`json:"b"`
	Asks [][]decimal.Decimal		`json:"a"`
}
