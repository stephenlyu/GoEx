package fcoin

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestFCoin_GetDepthWithWs(t *testing.T) {
	ft.GetDepthWithWs("EOS_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	ft.ws.CloseWs()
}

func TestFCoin_GetTradeWithWs(t *testing.T) {
	ft.GetTradeWithWs("EOS_USDT", func(instrumentId string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	ft.ws.CloseWs()
}
