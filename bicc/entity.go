package bicc

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type Symbol struct {
	Symbol string
	CountCoin string `json:"count_coin"`
	AmountPrecision int `json:"amount_precision"`
	BaseCoin string 	`json:"base_coin"`
	PricePrecision int `json:"price_precision"`
}

type OrderInfo struct {
	Msg string
	Code decimal.Decimal

	Symbol string
	OrderId decimal.Decimal
	ClientOrderId string
	Price decimal.Decimal
	OrigQty decimal.Decimal
	ExecuteQty decimal.Decimal
	CummulativeQuoteQty decimal.Decimal
	Status string
	TimeInForce string
	Type string
	Side string
	StopPrice decimal.Decimal
	IcebergQty decimal.Decimal
	Time decimal.Decimal
	UpdateTime decimal.Decimal
	IsWorking bool
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
	case OrderStatusRejected:
		status = goex.ORDER_REJECT
	case OrderStatusNew:
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
		if this.Type == OrderTypeLimit {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if this.Type == OrderTypeLimit {
			side = goex.SELL
		} else {
			side = goex.SELL_MARKET
		}
	}

	var avgPrice decimal.Decimal
	if this.ExecuteQty.IsPositive() {
		avgPrice = this.CummulativeQuoteQty.Div(this.ExecuteQty)
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.OrigQty,
		AvgPrice: avgPrice,
		DealAmount: this.ExecuteQty,
		Notinal: this.Price.Mul(this.OrigQty),
		DealNotional: this.CummulativeQuoteQty,
		OrderID2: this.OrderId.String(),
		ClientOid: this.ClientOrderId,
		Timestamp: this.Time.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
