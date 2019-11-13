package cointiger

import (
	"github.com/stephenlyu/GoEx"
	"github.com/shopspring/decimal"
)

type Symbol struct {
	Symbol string
	BaseCurrency string
	QuoteCurrency string
	AmountPrecision int
	PricePrecision int
	AmountMin decimal.Decimal
}

type OrderInfo struct {
	Side string
	TotalPrice decimal.Decimal		`json:"total_price"`
	AvgPrice decimal.Decimal		`json:"avg_price"`
	Type int 						`json:"type"`
	Id decimal.Decimal
	Volume decimal.Decimal
	Price decimal.Decimal
	DealVolume decimal.Decimal		`json:"deal_volume"`
	DealPrice decimal.Decimal		`json:"deal_price"`
	RemainVolume decimal.Decimal	`json:"remain_volume"`
	Status decimal.Decimal
	CreateAt decimal.Decimal			`json:"created_at"`
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status.IntPart() {
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
	if this.Side == "BUY" {
		if this.Type == ORDER_TYPE_LIMIT {
			side = goex.BUY
		} else {
			side = goex.BUY_MARKET
		}
	} else {
		if this.Type == ORDER_TYPE_LIMIT {
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
		Notinal: this.TotalPrice,
		DealNotional: this.DealPrice,
		OrderID2: this.Id.String(),
		Timestamp: this.CreateAt.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
