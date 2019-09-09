package fameex

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
	"github.com/gorilla/websocket"
	"crypto/tls"
)

func init() {
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
}

func TestFameex_GetDepthWithWs(t *testing.T) {
	fameex.Login()

	fameex.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("depth: %+v\n", depth)
	},func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("trades: %+v\n", trades)
	}, func (orders []goex.OrderDecimal) {
		log.Printf("orders: %+v\n", orders)
	})
	time.Sleep(10 * time.Minute)
	fameex.ws.CloseWs()
}

func Test_Login(t *testing.T) {
	fameex.Login()
}