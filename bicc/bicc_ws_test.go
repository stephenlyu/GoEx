package bicc

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var biccApi = NewBicc(http.DefaultClient, "", "")

func TestBicc_GetTradeWithWs(t *testing.T) {
	biccApi.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	biccApi.ws.CloseWs()
}

func TestBicc_GetDepthWithWs(t *testing.T) {
	biccApi.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	biccApi.ws.CloseWs()
}
