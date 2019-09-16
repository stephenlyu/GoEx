package binancefuture

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"time"
	"log"
)

func OnDepth (depth *goex.DepthDecimal) {
	log.Printf("askLen: %d bidLen: %d %+v\n", len(depth.AskList), len(depth.BidList), depth)
}

func OnTrade(symbol string, trades []goex.TradeDecimal) {
	log.Printf("%s %+v\n", symbol, trades)
}

func TestBinance_GetDepthTradeWithWs(t *testing.T) {
	ba.GetDepthTradeWithWs([]string{"BTC_USDT"}, OnDepth, OnTrade)
	time.Sleep(time.Hour * 4)
}
