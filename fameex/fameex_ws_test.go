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

	fameex.GetDepthWithWs("OMG_ETH", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	},func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	}, func (orders []goex.OrderDecimal) {
		log.Printf("%+v\n", orders)
	})
	time.Sleep(10 * time.Minute)
	fameex.ws.CloseWs()
}

func Test_Login(t *testing.T) {
	fameex.Login()
}