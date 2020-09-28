package biki

import (
	"log"
	"testing"
	"time"

	goex "github.com/stephenlyu/GoEx"
)

var bikiAPI = NewBiki("", "")

func TestBiki_GetTradeWithWs(t *testing.T) {
	bikiAPI.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	bikiAPI.ws.CloseWs()
}

func TestBiki_GetDepthWithWs(t *testing.T) {
	bikiAPI.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	bikiAPI.ws.CloseWs()
}
