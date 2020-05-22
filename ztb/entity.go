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
	Number         decimal.Decimal
	Price          decimal.Decimal
	AvgPrice       decimal.Decimal
	ID             decimal.Decimal
	Time           decimal.Decimal
	Type           decimal.Decimal
	Status         int
	CompleteNumber decimal.Decimal
	CompleteMoney  decimal.Decimal
	EntrustType    decimal.Decimal
	Fee            decimal.Decimal
}

// ToOrderDecimal is convert OrderInfo to OrderDecimal
func (orderInfo *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch orderInfo.Status {
	case OrderStatusInit:
		status = goex.ORDER_UNFINISH
	case OrderStatusCanceled:
		status = goex.ORDER_CANCEL
	case OrderStatusFilled, OrderStatusSettle:
		status = goex.ORDER_FINISH
	case OrderStatusPartiallyFilled:
		if orderInfo.CompleteNumber.IsPositive() {
			status = goex.ORDER_PART_FINISH
		} else {
			status = goex.ORDER_UNFINISH
		}
	}

	var side goex.TradeSide
	if orderInfo.Type.IntPart() == OrderBuy {
		if orderInfo.EntrustType.IntPart() == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if orderInfo.EntrustType.IntPart() == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price:        orderInfo.Price,
		Amount:       orderInfo.Number,
		AvgPrice:     orderInfo.AvgPrice,
		DealAmount:   orderInfo.CompleteNumber,
		DealNotional: orderInfo.CompleteMoney,
		OrderID2:     orderInfo.ID.String(),
		Timestamp:    orderInfo.Time.IntPart(),
		Status:       status,
		Currency:     goex.NewCurrencyPair2(symbol),
		Side:         side,
	}
}

type OrderReq struct {
	Type   int     `json:"type"`
	Amount float64 `json:"amount"`
	Price  float64 `json:"price"`
}
