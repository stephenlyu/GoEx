package appex

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type Symbol struct {
	BaseCurrency string 	`json:"base-currency"`
	QuoteCurrency string 	`json:"quote-currency"`
	PricePrecision int 		`json:"price-precision"`
	AmountPrecision int 	`json:"amount-precision"`
	MinAmount decimal.Decimal `json:"min-order-amt"`
	MaxAmount decimal.Decimal `json:"max-order-amt"`
	Symbol string 			`json:"symbol"`
}


type OrderInfo struct {
	Id decimal.Decimal
	Symbol string
	Price decimal.Decimal
	CreateAt decimal.Decimal		`json:"created-at"`
	Type string 					`json:"type"`
	Amount decimal.Decimal
	FilledAmount decimal.Decimal	`json:"field-amount"`
	FilledCashAmount decimal.Decimal`json:"field-cash-amount"`
	FilledFees decimal.Decimal		`json:"field-fees"`
	Source string
	State string
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.State {
	case "submitted":
		status = goex.ORDER_UNFINISH
	case "canceled":
		status = goex.ORDER_CANCEL
	case "filled":
		status = goex.ORDER_FINISH
	case "partial-filled":
		status = goex.ORDER_PART_FINISH
	case "cancelling":
		status = goex.ORDER_CANCEL_ING
	}

	var side goex.TradeSide
	switch this.Type {
	case "buy-market":
		side = goex.BUY_MARKET
	case "buy-limit":
		side = goex.BUY
	case "sell-market":
		side = goex.SELL_MARKET
	case "sell-limit":
		side = goex.SELL
	}

	var avgPrice decimal.Decimal
	if this.FilledAmount.IsPositive() {
		avgPrice = this.FilledCashAmount.Div(this.FilledAmount)
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Amount,
		AvgPrice: avgPrice,
		DealAmount: this.FilledAmount,
		Notinal: this.Price.Mul(this.Amount),
		DealNotional: this.FilledCashAmount,
		OrderID2: this.Id.String(),
		Timestamp: this.CreateAt.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
