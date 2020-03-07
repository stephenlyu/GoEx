package bicc

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type Symbol struct {
	Symbol          string
	CountCoin       string `json:"count_coin"`
	AmountPrecision int `json:"amount_precision"`
	BaseCoin        string    `json:"base_coin"`
	PricePrecision  int `json:"price_precision"`
}

type OrderInfo struct {
	ID         decimal.Decimal
	Side       string
	CreatedAt  decimal.Decimal        `json:"created_at"`
	Price      decimal.Decimal
	Volume     decimal.Decimal
	DealVolume decimal.Decimal    `json:"deal_volume"`
	TotalPrice decimal.Decimal    `json:"total_price"`
	DealPrice  decimal.Decimal    `json:"deal_price"`
	Type       decimal.Decimal
	Fee        decimal.Decimal
	AvgPrice   decimal.Decimal    `json:"avg_price"`
	Status     int
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
	case OrderStatusRejected:
		status = goex.ORDER_REJECT
	case OrderStatusInit, OrderStatusNew:
		status = goex.ORDER_UNFINISH
	case OrderStatusCanceled:
		status = goex.ORDER_CANCEL
	case OrderStatusFilled:
		status = goex.ORDER_FINISH
	case OrderStatusPartiallyFilled:
		status = goex.ORDER_PART_FINISH
	case OrderStatusPendingCancel:
		status = goex.ORDER_CANCEL_ING
	}

	var side goex.TradeSide
	if this.Side == OrderBuy {
		if this.Type.String() == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if this.Type.String() == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Volume,
		AvgPrice: this.AvgPrice,
		DealAmount: this.DealVolume,
		Notinal: this.Volume.Mul(this.Price),
		DealNotional: this.DealPrice,
		OrderID2: this.ID.String(),
		Timestamp: this.CreatedAt.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}

type OrderReq struct {
	Side   string    `json:"side"`
	Type   string    `json:"type"`
	Volume decimal.Decimal    `json:"volume"`
	Price  decimal.Decimal    `json:"price"`
}