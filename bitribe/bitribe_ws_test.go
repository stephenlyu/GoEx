package bitribe

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var bitribeApi = NewBitribe(http.DefaultClient, "", "")

func TestBitribe_GetTradeWithWs(t *testing.T) {
	bitribeApi.GetTradeWithWs("ETH_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	bitribeApi.ws.CloseWs()
}

func TestBitribe_GetDepthWithWs(t *testing.T) {
	bitribeApi.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	bitribeApi.ws.CloseWs()
}
