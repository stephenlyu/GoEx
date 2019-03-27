package gateiospot

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestOKExV3_GetDepthWithWs(t *testing.T) {
	gateioSpot.GetDepthWithWs([]goex.CurrencyPair{goex.EOS_USDT}, []float64{0.001}, 30, func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	gateioSpot.ws.CloseWs()
}

func TestOKExV3_GetTradeWithWs(t *testing.T) {
	gateioSpot.GetTradeWithWs([]goex.CurrencyPair{goex.BTC_USDT}, func(pair goex.CurrencyPair, trades []goex.TradeDecimal) {
		log.Printf("pair: %+v %+v\n", pair, trades)
	})
	time.Sleep(10 * time.Minute)
	gateioSpot.ws.CloseWs()
}

func TestOKExV3_Login(t *testing.T) {
	ch := make(chan struct{})
	gateioSpot.Login(func(err error) {
		close(ch)
	})
	<- ch
}

func OnAccount(account *goex.AccountDecimal) {
	log.Printf("OnAccount %+v", account)
}

func OnOrder(order *goex.OrderDecimal) {
	log.Printf("OnOrder %+v", order)
}

func TestOKExV3_Authenticated_Spot(t *testing.T) {
	ch := make(chan struct{})
	gateioSpot.Login(func(err error) {
		close(ch)
	})
	<- ch

	println("login success.")

	currencyA, currencyB := "ETH", "USDT"

	ca := goex.Currency{Symbol: currencyA}
	cb := goex.Currency{Symbol: currencyB}
	gateioSpot.GetAccountWithWs([]goex.Currency{ca, cb}, OnAccount)
	gateioSpot.GetOrderWithWs([]goex.CurrencyPair{goex.NewCurrencyPair(ca, cb)}, OnOrder)

	time.Sleep(10 * time.Minute)
}
