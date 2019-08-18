package fullcoin

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

func TestFullCoin_GetDepthWithWs(t *testing.T) {
	fullCoin.GetDepthWithWs("EOS_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	fullCoin.ws.CloseWs()
}

func TestFullCoin_GetTradeWithWs(t *testing.T) {
	fullCoin.GetTradeWithWs("EOS_USDT", func(instrumentId string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	fullCoin.ws.CloseWs()
}
