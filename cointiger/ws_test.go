package cointiger

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestCoinTiger_GetDepthWithWs(t *testing.T) {
	api.GetDepthWithWs("LEEE_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}

func TestCoinTiger_GetTradeWithWs(t *testing.T) {
	api.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	api.publicWs.CloseWs()
}
