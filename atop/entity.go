package atop

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type Symbol struct {
	Symbol     string
	MinAmount  decimal.Decimal `json:"minAmount"`
	CoinPoint  int `json:"coinPoint"`
	PricePoint int `json:"pricePoint"`
}

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

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
	case OrderStatusInit:
		status = goex.ORDER_UNFINISH
	case OrderStatusCanceled:
		status = goex.ORDER_CANCEL
	case OrderStatusFilled, OrderStatusSettle:
		status = goex.ORDER_FINISH
	case OrderStatusPartiallyFilled:
		if this.CompleteNumber.IsPositive() {
			status = goex.ORDER_PART_FINISH
		} else {
			status = goex.ORDER_UNFINISH
		}
	}

	var side goex.TradeSide
	if this.Type.IntPart() == OrderBuy {
		if this.EntrustType.IntPart() == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if this.EntrustType.IntPart() == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Number,
		AvgPrice: this.AvgPrice,
		DealAmount: this.CompleteNumber,
		DealNotional: this.CompleteMoney,
		OrderID2: this.ID.String(),
		Timestamp: this.Time.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}

type OrderReq struct {
	Type   int    `json:"type"`
	Amount float64    `json:"amount"`
	Price  float64    `json:"price"`
}