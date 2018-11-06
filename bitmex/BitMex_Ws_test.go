package bitmex

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var bitmexWs = NewBitMexWs("", "")

func TestBitMexWs_GetDepthWithWs(t *testing.T) {
	bitmexWs.GetDepthWithWs(goex.CurrencyPair{goex.XBT, goex.USD}, func(depth *goex.Depth) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(1 * time.Minute)
	bitmexWs.ws.CloseWs()
}

func TestBitMexWs_GetTradeWithWs(t *testing.T) {
	bitmexWs.GetTradeWithWs(goex.CurrencyPair{goex.XBT, goex.USD}, func(pair goex.CurrencyPair, trades []goex.Trade) {
		log.Println(trades)
	})
	time.Sleep(10 * time.Minute)
	bitmexWs.ws.CloseWs()
}
