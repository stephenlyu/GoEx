package zingex

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
	Price      decimal.Decimal
	Prize      decimal.Decimal
	Volume     decimal.Decimal    `json:"count"`
	DealVolume decimal.Decimal    `json:"success_count"`
	Side       decimal.Decimal    `json:"type"`
	Status     int
	TotalPrice decimal.Decimal    `json:"amount"`
	DealPrice  decimal.Decimal    `json:"success_amount"`
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
	case OrderStatusNew:
		status = goex.ORDER_UNFINISH
	case OrderStatusCanceled:
		status = goex.ORDER_CANCEL
	case OrderStatusFilled:
		status = goex.ORDER_FINISH
	case OrderStatusPartiallyFilled:
		status = goex.ORDER_PART_FINISH
	}

	var side goex.TradeSide
	if this.Side.String() == OrderBuy {
		side = goex.BUY
	} else {
		side = goex.SELL
	}

	var avgPrice decimal.Decimal
	if this.DealVolume.IsPositive() {
		avgPrice = this.DealPrice.Div(this.DealVolume)
	}

	var price = this.Price
	if price.IsZero() {
		price = this.Prize
	}

	return &goex.OrderDecimal{
		Price: price,
		Amount: this.Volume,
		AvgPrice: avgPrice,
		DealAmount: this.DealVolume,
		Notinal: this.Volume.Mul(this.Price),
		DealNotional: this.DealPrice,
		OrderID2: this.ID.String(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
