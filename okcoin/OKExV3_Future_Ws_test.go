package okcoin

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var okexFutureV3 = NewOKExV3(http.DefaultClient, "", "", "")

func TestOKExV3_GetDepthWithWs(t *testing.T) {
	okexFutureV3.GetDepthWithWs("EOS-USD-190329", func(depth *goex.Depth) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_GetTradeWithWs(t *testing.T) {
	okexFutureV3.GetTradeWithWs("EOS-USD-SWAP", func(instrumentId string, trades []goex.Trade) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_Login(t *testing.T) {
	ch := make(chan struct{})
	okexV3.Login(func(err error) {
		close(ch)
	})
	<- ch
}

func OnAccount(isSwap bool, account *goex.FutureAccount) {
	log.Printf("OnAccount %+v", account)
}

func OnPosition(positions []goex.FuturePosition) {
	log.Printf("OnPosition %+v", positions)
}

func OnOrder(orders []goex.FutureOrder) {
	log.Printf("OnOrder %+v", orders)
}

func TestOKExV3_Authenticated_Futures(t *testing.T) {
	ch := make(chan struct{})
	okexV3.Login(func(err error) {
		close(ch)
	})
	<- ch

	const instrumentId = "EOS-USD-190329"

	okexV3.GetAccountWithWs(goex.EOS, false, OnAccount)
	okexV3.GetPositionWithWs(instrumentId, OnPosition)
	okexV3.GetOrderWithWs(instrumentId, OnOrder)

	time.Sleep(10 * time.Minute)
}

func TestOKExV3_Authenticated_Swap(t *testing.T) {
	ch := make(chan struct{})
	okexV3.Login(func(err error) {
		close(ch)
	})
	<- ch

	const instrumentId = "EOS-USD-SWAP"

	okexV3.GetAccountWithWs(goex.EOS, true, OnAccount)
	okexV3.GetPositionWithWs(instrumentId, OnPosition)
	okexV3.GetOrderWithWs(instrumentId, OnOrder)

	time.Sleep(10 * time.Minute)
}
