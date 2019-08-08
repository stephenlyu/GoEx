package huobifuture

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestHuobi_GetDepthWithWs(t *testing.T) {
	huobi.GetDepthWithWs("BTC_CQ", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	huobi.publicWs.CloseWs()
}

func TestAppex_GetTradeWithWs(t *testing.T) {
	huobi.GetTradeWithWs("BTC_CQ", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	huobi.publicWs.CloseWs()
}
