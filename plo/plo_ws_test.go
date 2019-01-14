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

	ploWs.GetDepthWithWs(goex.CurrencyPair{goex.EOS, goex.USD}, func(depth *goex.Depth) {
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

	ploWs.GetTradeWithWs(goex.CurrencyPair{goex.EOS, goex.USD}, false, func(pair goex.CurrencyPair, isIndex bool, trades []goex.Trade) {
		log.Println(pair, isIndex, trades)
	})
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
		log.Printf("%+v", orders)
	})

	ploWs.GetAccountWithWs(func(account *goex.FutureAccount) {
		log.Printf("%+v", account)
	})

	ploWs.GetPositionWithWs(func(positions []PloPosition) {
		log.Printf("%+v", positions)
	})

	time.Sleep(1000 * time.Minute)
	ploWs.ws.CloseWs()
}
