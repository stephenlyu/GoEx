package biki

import (
	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

// OrderReq Order request
type OrderReq struct {
	Side   string  `json:"side"`
	Type   string  `json:"type"`
	Volume float64 `json:"volume"`
	Price  float64 `json:"price"`
}

// Symbol symbol
type Symbol struct {
	Symbol          string
	CountCoin       string `json:"count_coin"`
	BaseCoin        string `json:"base_coin"`
	AmountPrecision int    `json:"amount_precision"`
	PricePrecision  int    `json:"price_precision"`
}

// OrderInfo order info
type OrderInfo struct {
	Side         string
	TotalPrice   decimal.Decimal `json:"total_price"`
	AvgPrice     decimal.Decimal `json:"avg_price"`
	Type         int             `json:"type"`
	ID           decimal.Decimal
	Volume       decimal.Decimal
	Price        decimal.Decimal
	DealVolume   decimal.Decimal `json:"deal_volume"`
	DealPrice    decimal.Decimal `json:"deal_price"`
	RemainVolume decimal.Decimal `json:"remain_volume"`
	Status       decimal.Decimal
	CreateAt     decimal.Decimal `json:"created_at"`
}

// ToOrderDecimal translate to OrderDecimal
func (oi *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch oi.Status.IntPart() {
	case 6:
		status = goex.ORDER_REJECT
	case 0, 1:
		status = goex.ORDER_UNFINISH
	case 4:
		status = goex.ORDER_CANCEL
	case 2:
		status = goex.ORDER_FINISH
	case 3:
		status = goex.ORDER_PART_FINISH
	case 5:
		status = goex.ORDER_CANCEL_ING
	}

	var side goex.TradeSide
	if oi.Side == "BUY" {
		if oi.Type == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if oi.Type == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price:        oi.Price,
		Amount:       oi.Volume,
		AvgPrice:     oi.AvgPrice,
		DealAmount:   oi.DealVolume,
		Notinal:      oi.TotalPrice,
		DealNotional: oi.DealPrice,
		OrderID2:     oi.ID.String(),
		Timestamp:    oi.CreateAt.IntPart(),
		Status:       status,
		Currency:     goex.NewCurrencyPair2(symbol),
		Side:         side,
	}
}
