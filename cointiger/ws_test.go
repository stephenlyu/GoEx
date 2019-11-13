package cointiger

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestHuobi_GetDepthWithWs(t *testing.T) {
	api.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}

func TestHuobi_GetTradeWithWs(t *testing.T) {
	api.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}
