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
	Symbol string
	Fee decimal.Decimal
	AvgPrice decimal.Decimal		`json:"avg_price"`
	Type string
	MTime decimal.Decimal
	Volume decimal.Decimal
	Price decimal.Decimal
	CTime decimal.Decimal
	DealVolume decimal.Decimal		`json:"deal_volume"`
	Id decimal.Decimal
	DealMoney decimal.Decimal		`json:"deal_money"`
	Status decimal.Decimal
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status.IntPart() {
	case 6:
		status = goex.ORDER_REJECT
	case 1:
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
	switch this.Type {
	case OrderTypeBuyLimit:
		side = goex.BUY
	case OrderTypeBuyMarket:
		side = goex.BUY_MARKET
	case OrderTypeSellLimit:
		side = goex.SELL
	case OrderTypeSellMarket:
		side = goex.SELL_MARKET
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Volume,
		AvgPrice: this.AvgPrice,
		DealAmount: this.DealVolume,
		Notinal: this.Price.Mul(this.Volume),
		DealNotional: this.DealMoney,
		OrderID2: this.Id.String(),
		Timestamp: this.CTime.IntPart(),
		Status: status,
		Fee: this.Fee,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
