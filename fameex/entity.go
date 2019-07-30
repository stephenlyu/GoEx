package fameex

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
	"fmt"
)

type Symbol struct {
	BaseCurrency string 	`json:"base-currency"`
	QuoteCurrency string 	`json:"quote-currency"`
	PricePrecision int 		`json:"price-precision"`
	AmountPrecision int 	`json:"amount-precision"`
	MinAmount decimal.Decimal `json:"min-order-amt"`
	Symbol string 			`json:"symbol"`
}

type OrderReq struct {
	Side int					`json:"buyType"`
	Type int					`json:"buyClass"`
	Price decimal.Decimal		`json:"price"`
	Amount decimal.Decimal		`json:"count"`
}

type OrderInfo struct {
	OrderId          string
	TaskId 			 string
	Base			 string
	Quote 			 string
	Coin1 			 string
	Coin2 			 string
	BuyType          int
	State            int
	Price            decimal.Decimal
	Count            decimal.Decimal
	TotalCount       decimal.Decimal
	DealedCount      decimal.Decimal
	DealedMoney        decimal.Decimal
	CreateTime 		 int64
}

func (this *OrderInfo) ToOrderDecimal() *goex.OrderDecimal {
	var symbol string
	var orderId string

	if this.Base != "" {
		symbol = fmt.Sprintf("%s_%s", this.Base, this.Quote)
	} else {
		symbol = fmt.Sprintf("%s_%s", this.Coin1, this.Coin2)
	}

	if this.OrderId != "" {
		orderId = this.OrderId
	} else {
		orderId = this.TaskId
	}

	var status goex.TradeStatus
	switch this.State {
	case 1, 7:
		status = goex.ORDER_UNFINISH
	case 11:
		status = goex.ORDER_CANCEL
	case 4, 10:
		status = goex.ORDER_FINISH
	case 9:
		status = goex.ORDER_PART_FINISH
	}

	var side goex.TradeSide
	switch this.BuyType {
	case SIDE_BUY:
		side = goex.BUY
	case SIDE_SELL:
		side = goex.SELL
	}

	var avgPrice decimal.Decimal
	if this.DealedCount.IsPositive() {
		avgPrice = this.DealedMoney.Div(this.DealedCount)
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.TotalCount,
		AvgPrice: avgPrice,
		DealAmount: this.DealedCount,
		Notinal: this.Price.Mul(this.TotalCount),
		DealNotional: this.DealedMoney,
		OrderID2: orderId,
		Timestamp: this.CreateTime / 1000000,
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
