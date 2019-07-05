package binance

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"time"
	"log"
)

func OnDepth (depth *goex.DepthDecimal) {
	log.Printf("%+v\n", depth)
}

func OnTrade(symbol string, trades []goex.TradeDecimal) {
	log.Printf("%s %+v\n", symbol, trades)
}

func TestBinance_GetDepthTradeWithWs(t *testing.T) {
	ba.GetDepthTradeWithWs([]string{"EOS_USDT"}, OnDepth, OnTrade)
	time.Sleep(time.Hour * 4)
}
