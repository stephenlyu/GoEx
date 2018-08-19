package okcoin

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var okexFuture = NewOKEx(http.DefaultClient, "", "")

func TestOKEx_GetDepthWithWs(t *testing.T) {
	okexFuture.GetDepthWithWs(goex.BTC_USD, goex.QUARTER_CONTRACT, 0, func(depth *goex.Depth) {
		log.Print(depth)
	})
	time.Sleep(1 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKEx_GetTickerWithWs(t *testing.T) {
	okexFuture.GetTickerWithWs(goex.BTC_USD, goex.QUARTER_CONTRACT, func(ticker *goex.Ticker) {
		log.Print(ticker)
	})
	time.Sleep(1 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKEx_GetTradeWithWs(t *testing.T) {
	okexFuture.GetTradeWithWs(goex.BTC_USD, goex.QUARTER_CONTRACT, func(pair goex.CurrencyPair, contractType string, trades []goex.Trade) {
		log.Print(trades)
	})
	time.Sleep(1 * time.Minute)
	okexFuture.ws.CloseWs()
}
