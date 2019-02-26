package plo

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var ploWs = NewPloWs("", "")


func TestPloWs_Authenticate(t *testing.T) {
	ploWs = NewPloWs(API_KEY, SECRET_KEY)

	ploWs.SetErrorHandler(func(err error) {
		log.Fatalf("Error: %+v", err)
	})

	ch := make(chan bool)
	err := ploWs.Authenticate(func() {
		ch <- true
	})
	chk(err)

	<- ch
}

func TestPloWs_GetDepthWithWs(t *testing.T) {
	ploWs = NewPloWs(API_KEY, SECRET_KEY)

	ploWs.SetErrorHandler(func(err error) {
		log.Fatalf("Error: %+v", err)
	})

	ch := make(chan bool)
	err := ploWs.Authenticate(func() {
		ch <- true
	})
	chk(err)

	<- ch

	ploWs.GetDepthWithWs(goex.CurrencyPair{goex.Currency{Symbol:"SHE"}, goex.USD}, func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	ploWs.ws.CloseWs()
}

func TestPloWs_GetTradeWithWs(t *testing.T) {
	ploWs = NewPloWs(API_KEY, SECRET_KEY)

	ploWs.SetErrorHandler(func(err error) {
		log.Fatalf("Error: %+v", err)
	})

	ch := make(chan bool)
	err := ploWs.Authenticate(func() {
		ch <- true
	})
	chk(err)

	<- ch

	ploWs.GetTradeWithWs(goex.CurrencyPair{goex.Currency{Symbol:"SHE"}, goex.USD}, false, func(pair goex.CurrencyPair, isIndex bool, trades []goex.TradeDecimal) {
		log.Println(pair, isIndex, trades)
	})
	log.Println("here")
	time.Sleep(10 * time.Minute)
	ploWs.ws.CloseWs()
}

func TestPloWs_GetOrderWithWs(t *testing.T) {
	ploWs = NewPloWs(API_KEY, SECRET_KEY)

	ploWs.SetErrorHandler(func (err error) {
		log.Fatalf("Error: %+v", err)
	})

	ch := make(chan bool)
	err := ploWs.Authenticate(func() {
		ch <- true
	})
	chk(err)

	<- ch

	ploWs.GetOrderWithWs(func(orders []PloOrder) {
		log.Printf("order: %+v", orders)
	})

	ploWs.GetAccountWithWs(func(account *goex.FutureAccount) {
		log.Printf("account: %+v", account)
	})

	ploWs.GetPositionWithWs(func(positions []PloPosition) {
		log.Printf("position: %+v", positions)
	})

	time.Sleep(1000 * time.Minute)
	ploWs.ws.CloseWs()
}
