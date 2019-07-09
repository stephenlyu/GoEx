package ceohk

import (
	"github.com/stephenlyu/GoEx"
	"github.com/shopspring/decimal"
)

type OrderInfo struct {
	Currency string
	Id decimal.Decimal
	Price decimal.Decimal
	Status decimal.Decimal
	TotalAmount decimal.Decimal		`json:"total_amount"`
	DealAmount decimal.Decimal		`json:"trade_amount"`
	TradeMoney decimal.Decimal		`json:"trade_money"`
	TradeTime decimal.Decimal		`json:"trade_time"`
	Type int
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status.IntPart() {
	case 0:
		status = goex.ORDER_UNFINISH
	case 1:
		status = goex.ORDER_FINISH
	case 2:
		status = goex.ORDER_CANCEL
	case 3:
		status = goex.ORDER_CANCEL
	}

	var side goex.TradeSide
	if this.Type == TRADE_TYPE_BUY {
		side = goex.BUY
	} else {
		side = goex.SELL
	}

	var avgPrice decimal.Decimal
	if this.DealAmount.IsPositive() {
		avgPrice = this.TradeMoney.Div(this.DealAmount)
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.TotalAmount,
		AvgPrice: avgPrice,
		DealAmount: this.DealAmount,
		Notinal: this.Price.Mul(this.TotalAmount),
		DealNotional: this.TradeMoney,
		OrderID2: this.Id.String(),
		Timestamp: this.TradeTime.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
