package ztb

import (
	"log"
	"net/http"
	"testing"
	"time"

	goex "github.com/stephenlyu/GoEx"
)

var ztbAPI = NewZtb(http.DefaultClient, "", "")

func TestZtb_GetTradeWithWs(t *testing.T) {
	ztbAPI.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	ztbAPI.ws.CloseWs()
}

func TestZtb_GetDepthWithWs(t *testing.T) {
	ztbAPI.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	ztbAPI.ws.CloseWs()
}
