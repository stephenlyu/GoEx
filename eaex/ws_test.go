package eaex

import (
	"log"
	"testing"
	"time"

	goex "github.com/stephenlyu/GoEx"
)

func TestEAEX_GetDepthWithWs(t *testing.T) {
	api.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}

func TestEAEX_GetTradeWithWs(t *testing.T) {
	api.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}
