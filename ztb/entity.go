package ztb

import (
	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

// Symbol is coin pair
type Symbol struct {
	Symbol              string
	BaseAssetPrecision  int
	QuoteAssetPrecision int
}

// OrderInfo is order info
type OrderInfo struct {
	Amount    decimal.Decimal
	Ctime     decimal.Decimal
	DealFee   decimal.Decimal `json:"deal_fee"`
	DealMoney decimal.Decimal `json:"deal_money"`
	DealStock decimal.Decimal `json:"deal_stock"`
	ID        decimal.Decimal
	Price     decimal.Decimal
	Side      decimal.Decimal
	Status    int
	AvgPrice  decimal.Decimal
	Type      decimal.Decimal
}

// ToOrderDecimal is convert OrderInfo to OrderDecimal
func (orderInfo *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch orderInfo.Status {
	case OrderStatusInit, OrderStatusQueue:
		status = goex.ORDER_UNFINISH
	case OrderStatusCanceled:
		status = goex.ORDER_CANCEL
	case OrderStatusFilled:
		status = goex.ORDER_FINISH
	case OrderStatusPartiallyFilled:
		status = goex.ORDER_PART_FINISH
	}

	var side goex.TradeSide
	if orderInfo.Side.String() == OrderBuy {
		if orderInfo.Type.String() == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if orderInfo.Type.String() == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	ctime, _ := orderInfo.Ctime.Float64()
	return &goex.OrderDecimal{
		Price:        orderInfo.Price,
		Amount:       orderInfo.Amount,
		AvgPrice:     orderInfo.AvgPrice,
		DealAmount:   orderInfo.DealStock,
		DealNotional: orderInfo.DealMoney,
		OrderID2:     orderInfo.ID.String(),
		Timestamp:    int64(ctime * 1000),
		Status:       status,
		Currency:     goex.NewCurrencyPair2(symbol),
		Side:         side,
	}
}
