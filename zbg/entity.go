package zbg

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
	"strings"
)

type Market struct {
	AmountDecimal int
	MinAmount decimal.Decimal
	BuyerCurrencyId string
	PriceDecimal int
	MarketId string
	SellerCurrencyId string
	DefaultFee decimal.Decimal
	Name string
	State int
}

type CurrencyInfo struct {
	TotalNumber decimal.Decimal
	CurrencyId string
	Name string
	DefaultDecimal int
}

type OrderInfo struct {
	Amount decimal.Decimal
	TotalMoney decimal.Decimal
	EntrustId string
	Type decimal.Decimal
	CompleteAmount decimal.Decimal
	MarketId string
	DealTimes decimal.Decimal
	Price decimal.Decimal
	CompleteTotalMoney decimal.Decimal
	Status decimal.Decimal
	CreateTime decimal.Decimal
}

func (this *OrderInfo) ToOrderDecimal(marketName string) *goex.OrderDecimal {
	var avgPrice decimal.Decimal
	if this.CompleteAmount.IsPositive() {
		avgPrice = this.CompleteTotalMoney.Div(this.CompleteAmount)
	}

	var status goex.TradeStatus
	switch this.Status.IntPart() {
	case -2, -1:
		status = goex.ORDER_REJECT
	case 0:
		status = goex.ORDER_UNFINISH
	case 1:
		status = goex.ORDER_CANCEL
	case 2:
		status = goex.ORDER_FINISH
	case 3:
		status = goex.ORDER_PART_FINISH
	case 4:
		status = goex.ORDER_CANCEL_ING
	}

	var side goex.TradeSide
	if this.Type.IntPart() == 0 {
		side = goex.SELL
	} else if this.Type.IntPart() == 1 {
		side = goex.BUY
	}

	return &goex.OrderDecimal{
		Price: this.Price,
		Amount: this.Amount,
		AvgPrice: avgPrice,
		DealAmount: this.CompleteAmount,
		Notinal: this.TotalMoney,
		DealNotional: this.CompleteTotalMoney,
		OrderID2: this.EntrustId,
		Timestamp: this.CreateTime.IntPart(),
		Status: status,
		Currency: goex.NewCurrencyPair2(strings.ToUpper(marketName)),
		Side: side,
	}
}