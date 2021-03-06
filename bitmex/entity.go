package bitmex

import (
	"github.com/stephenlyu/GoEx"
	"time"
	"strings"
)

const (
	ORDER_STATUS_NEW = "new"
	ORDER_STATUS_PARTIALLY_FILLED = "partiallyfilled"
	ORDER_STATUS_FILLED = "filled"
	ORDER_STATUS_CANCELED = "canceled"
	ORDER_STATUS_REJECTED = "rejected"
	ORDER_STATUS_EXPIRED = "expired"
)
const SATOSHI = 0.00000001

const UTC_FORMAT = "2006-01-02T15:04:05.999Z"

type BitmexOrder struct {
	OrderId string 			`json:"orderID"`
	ClientOrderId string 	`json:"clOrdID"`
	Symbol string 			`json:"symbol"`
	Side string 			`json:"side"`
	OrderQty int64 			`json:"orderQty"`
	Price float64 			`json:"price"`
	AvgPrice float64 		`json:"avgPx"`
	Currency string 		`json:"currency"`
	OrderType string 		`json:"ordType"`
	TimeInForce string 		`json:"timeInForce"`
	Status string 			`json:"ordStatus"`
	RejectReason string 	`json:"ordRejReason"`
	LeavesQty int64			`json:"leavesQty"`
	CumQty int64 			`json:"cumQty"`
	Timestamp string 		`json:"timestamp"`
}

func (order *BitmexOrder) ToFutureOrder() *goex.FutureOrder {
	ret := new(goex.FutureOrder)

	ret.Price = order.Price
	ret.AvgPrice = order.AvgPrice
	ret.Amount = float64(order.OrderQty)
	ret.DealAmount = float64(order.CumQty)
	ret.OrderID2 = order.OrderId
	ret.ClientOrderID = order.ClientOrderId
	_, ts := ParseTimestamp(order.Timestamp)
	ret.OrderTime = ts
	switch strings.ToLower(order.Status) {
	case ORDER_STATUS_NEW:
		ret.Status = goex.ORDER_UNFINISH
	case ORDER_STATUS_PARTIALLY_FILLED:
		ret.Status = goex.ORDER_PART_FINISH
	case ORDER_STATUS_FILLED:
		ret.Status = goex.ORDER_FINISH
	case ORDER_STATUS_CANCELED:
		ret.Status = goex.ORDER_CANCEL
	case ORDER_STATUS_REJECTED:
		ret.Status = goex.ORDER_REJECT
	case ORDER_STATUS_EXPIRED:
		ret.Status = goex.ORDER_CANCEL
	}
	ret.ContractName = order.Symbol
	if order.Side == "Buy" {
		if order.OrderType == "Limit" {
			ret.Side = goex.BUY
		} else {
			ret.Side = goex.BUY_MARKET
		}
	} else {
		if order.OrderType == "Limit" {
			ret.Side = goex.SELL
		} else {
			ret.Side = goex.SELL_MARKET
		}
	}

	return ret
}

type Execution struct {
	ExecId string 			`json:"execID"`
	OrderId string 			`json:"orderID"`
	ClientOrderId string 	`json:"clOrdID"`
	Symbol string 			`json:"symbol"`
	Side string 			`json:"side"`
	LastQty int64 			`json:"lastQty"`
	LastPrice float64 		`json:"lastPx"`
	Price float64 			`json:"price"`
	AvgPrice float64 		`json:"avgPx"`
	Commission int64 		`json:"execComm"`			// 手续费，单位聪
	TransactionTime string 	`json:"transactTime"`
	TrdMatchId string 		`json:"trdMatchID"`
}

func (e *Execution) ToFill() *goex.FutureFill {
	f := new(goex.FutureFill)
	f.FillId = e.TrdMatchId
	f.OrderId = e.OrderId
	f.ClientOrderId = e.ClientOrderId
	f.Symbol = e.Symbol
	if e.Side == "Buy" {
		f.Side = goex.BUY
	} else {
		f.Side = goex.SELL
	}
	f.LastQty = e.LastQty
	f.LastPrice = e.LastPrice
	f.Price = e.Price
	f.AvgPrice = e.AvgPrice
	f.Commission = e.Commission
	_, ts := ParseTimestamp(e.TransactionTime)
	f.TransactionTime = ts

	return f
}

type BitmexPosition struct {
	Symbol string 			`json:"symbol"`
	CurrentQty float64		`json:"currentQty"`
	AveragePrice float64	`json:"avgCostPrice"`
	RealisedPnl float64 	`json:"realisedPnl"`
	UnrealisedPnl float64 	`json:"unrealisedPnl"`
}

func (p *BitmexPosition) ToFuturePosition() *goex.FuturePosition {
	ret := new(goex.FuturePosition)
	ret.InstrumentId = p.Symbol
	if p.CurrentQty < 0 {
		ret.SellAmount = -p.CurrentQty
		ret.SellPriceAvg = p.AveragePrice
		ret.SellProfitUnReal = p.UnrealisedPnl
		ret.SellProfitReal = p.RealisedPnl
	} else if p.CurrentQty > 0 {
		ret.BuyAmount = p.CurrentQty
		ret.BuyPriceAvg = p.AveragePrice
		ret.BuyProfitUnReal = p.UnrealisedPnl
		ret.BuyProfitReal = p.RealisedPnl
	}
	return ret
}

type Margin struct {
	Account int64 			`json:"account"`
	Currency string 		`json:"currency"`
	RealizedPnl int64 		`json:"realisedPnl"`
	UnrealizedPnl int64 	`json:"unrealisedPnl"`
	WalletBalance int64 	`json:"walletBalance"`
	MarginBalance int64 	`json:"marginBalance"`
	AvailableMargin int64 	`json:"availableMargin"`
	WithdrawableMargin int64 `json:"withdrawableMargin"`
	Amount int64 			`json:"amount"`
}

func (m *Margin) ToFutureAccount() *goex.FutureAccount {
	if m.Currency == "XBt" {
		m.Currency = "BTC"
	}
	currency := goex.Currency{m.Currency, ""}

	rights := float64(m.MarginBalance)
	keepDeposit := float64(m.MarginBalance)
	amount := float64(m.Amount)

	if rights == 0 && keepDeposit == 0 && amount > 0 {
		rights = amount
		keepDeposit = amount
	}

	return &goex.FutureAccount{
		FutureSubAccounts: map[goex.Currency]goex.FutureSubAccount {
			currency: goex.FutureSubAccount{
				Currency: currency,
				AccountRights: rights * SATOSHI,
				KeepDeposit: keepDeposit * SATOSHI,
				ProfitReal: float64(m.RealizedPnl) * SATOSHI,
				ProfitUnreal: float64(m.UnrealizedPnl) * SATOSHI,
			},
		},
	}
}

func ParseTimestamp(ts string) (error, int64) {
	t, err := time.Parse(UTC_FORMAT, ts)
	if err != nil {
		return err, 0
	}
	return nil, t.UnixNano() / int64(time.Millisecond)
}

func FormatTimestamp(ts int64) string {
	t := time.Unix(ts / 1000, ts % 1000 * int64(time.Millisecond)).In(time.UTC)
	return t.Format(UTC_FORMAT)
}
