package fullcoin

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type Symbol struct {
	Symbol string 			`json:"symbol"`
	BaseCurrency string 	`json:"base_coin"`
	QuoteCurrency string 	`json:"count_coin"`
	PricePrecision int 		`json:"price_precision"`
	AmountPrecision int 	`json:"amount_precision"`
}

//{side:"BUY",type:"1",volume:"0.01",price:"6400",fee_is_user_exchange_coin:"0"}
type OrderReq struct {
	Side string 			`json:"side"`
	Type decimal.Decimal	`json:"type"`
	Volume decimal.Decimal	`json:"volume"`
	Price decimal.Decimal	`json:"price"`
	FeeIsUserExchangeCoin decimal.Decimal `json:"fee_is_user_exchange_coin"`
}

type OrderInfo struct {
	Id decimal.Decimal
	Side string
	Symbol string
	Price decimal.Decimal
	CreateAt decimal.Decimal		`json:"created_at"`
	Type int
	AvgPrice decimal.Decimal		`json:"avg_price"`
	Amount decimal.Decimal			`json:"volume"`
	FilledAmount decimal.Decimal	`json:"deal_volume"`
	Status int
}

func (this *OrderInfo) ToOrderDecimal(symbol string) *goex.OrderDecimal {
	var status goex.TradeStatus
	switch this.Status {
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
	switch this.Side {
	case SIDE_BUY:
		switch this.Type {
		case TYPE_LIMIT:
			side = goex.BUY
		case TYPE_MARKET:
			side = goex.BUY_MARKET
		}
	case SIDE_SELL:
		switch this.Type {
		case TYPE_LIMIT:
			side = goex.SELL
		case TYPE_MARKET:
			side = goex.SELL_MARKET
		}
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Amount,
		AvgPrice: this.AvgPrice,
		DealAmount: this.FilledAmount,
		Notinal: this.Price.Mul(this.Amount),
		DealNotional: this.AvgPrice.Mul(this.FilledAmount),
		OrderID2: this.Id.String(),
		Timestamp: this.CreateAt.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(symbol),
		Side: side,
	}
}
