package eaex

import (
	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

type Symbol struct {
	Symbol             string
	BaseAsset          string
	QuoteAsset         string
	BaseAssetPrecision decimal.Decimal
	QuotePrecision     decimal.Decimal
	AmountMin          decimal.Decimal
	Filters            []struct {
		FilterType string
		MinQty     decimal.Decimal
		TickSize   decimal.Decimal
		StepSize   decimal.Decimal
	}
}

type OrderInfo struct {
	Msg  string
	Code decimal.Decimal

	Symbol      string
	OrderId     decimal.Decimal
	Price       decimal.Decimal
	OrigQty     decimal.Decimal
	ExecutedQty decimal.Decimal
	Status      string
	Type        string
	Side        string
	Time        decimal.Decimal
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
	case "REJECTED":
		status = goex.ORDER_REJECT
	case "NEW":
		status = goex.ORDER_UNFINISH
	case "CANCELED":
		status = goex.ORDER_CANCEL
	case "FILLED":
		status = goex.ORDER_FINISH
	case "PARTIALLY_FILLED":
		status = goex.ORDER_PART_FINISH
	case "PENDING_CANCEL":
		status = goex.ORDER_CANCEL_ING
	}

	var side goex.TradeSide
	switch this.Side {
	case ORDER_BUY:
		switch this.Type {
		case ORDER_TYPE_LIMIT:
			side = goex.BUY
		case ORDER_TYPE_MARKET:
			side = goex.BUY_MARKET
		}
	case ORDER_SELL:
		switch this.Type {
		case ORDER_TYPE_LIMIT:
			side = goex.SELL
		case ORDER_TYPE_MARKET:
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price:        this.Price,
		Amount:       this.OrigQty,
		DealAmount:   this.ExecutedQty,
		Notinal:      this.Price.Mul(this.OrigQty),
		DealNotional: this.Price.Mul(this.ExecutedQty),
		AvgPrice:     this.Price,
		OrderID2:     this.OrderId.String(),
		Timestamp:    this.Time.IntPart(),
		Status:       status,
		Currency:     goex.NewCurrencyPair2(symbol),
		Side:         side,
	}
}

type Fill struct {
	Symbol          string
	Id              decimal.Decimal
	OrderId         decimal.Decimal
	Price           decimal.Decimal
	Qty             decimal.Decimal
	Commission      decimal.Decimal
	CommissionAsset string
	Time            decimal.Decimal
	IsBuyer         bool
	IsMaker         bool
}
