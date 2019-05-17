package okcoin

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
	"os"
	"runtime/pprof"
)

var okexFutureV3 = NewOKExV3(http.DefaultClient, "", "", "")

func TestOKExV3_GetDepthWithWs(t *testing.T) {
	okexFutureV3.GetDepthWithWs("EOS-USD-SWAP", func(depth *goex.Depth) {
		log.Printf("ask1: %f bid1: %f\n", depth.AskList[0].Price, depth.BidList[0].Price)
	})
	writer, err := os.Create("cpu.prof")
	chk(err)
	pprof.StartCPUProfile(writer)
	time.Sleep(time.Minute)
	pprof.StopCPUProfile()
	okexFuture.CloseWs()
}

func TestOKExV3_GetTradeWithWs(t *testing.T) {
	okexFutureV3.GetTradeWithWs("EOS-USD-SWAP", func(instrumentId string, trades []goex.Trade) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_GetIndexTickerWithWs(t *testing.T) {
	okexFutureV3.GetIndexTickerWithWs("EOS-USD", func(instrumentId string, tickers []goex.Ticker) {
		log.Printf("%+v\n", tickers)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_GetFundingRateWithWs(t *testing.T) {
	okexFutureV3.GetFundingRateWithWs("EOS-USD-SWAP", func(fundingRate SWAPFundingRate) {
		log.Printf("%+v\n", fundingRate)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_Login(t *testing.T) {
	okexV3.Login()
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
	okexV3.Login()

	const instrumentId = "EOS-USD-190628"

	okexV3.GetAccountWithWs(goex.EOS, false, OnAccount)
	okexV3.GetPositionWithWs(instrumentId, OnPosition)
	okexV3.GetOrderWithWs(instrumentId, OnOrder)

	time.Sleep(10 * time.Minute)
}

func TestOKExV3_Authenticated_Swap(t *testing.T) {
	okexV3.Login()

	const instrumentId = "EOS-USD-SWAP"

	okexV3.GetAccountWithWs(goex.EOS, true, OnAccount)
	okexV3.GetPositionWithWs(instrumentId, OnPosition)
	okexV3.GetOrderWithWs(instrumentId, OnOrder)

	time.Sleep(10 * time.Minute)
}
