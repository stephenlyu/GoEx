package huobifuture

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
	"fmt"
	"github.com/Sirupsen/logrus"
)

func TestHuobi_GetDepthWithWs(t *testing.T) {
	huobi.GetDepthWithWs("BTC_CQ", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	huobi.publicWs.CloseWs()
}

func TestHuobi_GetTradeWithWs(t *testing.T) {
	huobi.GetTradeWithWs("BTC_CQ", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	huobi.publicWs.CloseWs()
}

func TestHuobiFuture_Login(t *testing.T) {
	err := huobi.Login()
	fmt.Println(err)
	ch := make(chan struct{})
	<- ch
}

func TestHuobiFuture_GetOrderWithWs(t *testing.T) {
	err := huobi.Login()
	fmt.Println("loged in", err)

	err = huobi.GetOrderWithWs("eth", func(orders []goex.FutureOrderDecimal) {
		logrus.Printf("%+v", orders)
	})
	fmt.Println(err)

	ch := make(chan struct{})
	<- ch
}
