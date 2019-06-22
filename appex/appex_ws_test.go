package appex

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestAppex_GetDepthWithWs(t *testing.T) {
	appex.GetDepthWithWs("EOS_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	appex.ws.CloseWs()
}

func TestAppex_GetTradeWithWs(t *testing.T) {
	appex.GetTradeWithWs("EOS_USDT", func(instrumentId string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	appex.ws.CloseWs()
}
