package okexv3spot

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var okexSpotV3 = NewOKExV3Spot(http.DefaultClient, "", "", "")

func TestOKExV3_GetDepthWithWs(t *testing.T) {
	okexSpotV3.GetDepthWithWs("EOS-USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	okexSpotV3.ws.CloseWs()
}

func TestOKExV3_GetTradeWithWs(t *testing.T) {
	okexSpotV3.GetTradeWithWs("EOS-USDT", func(instrumentId string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	okexSpotV3.ws.CloseWs()
}

func TestOKExV3_Login(t *testing.T) {
	ch := make(chan struct{})
	okexV3.Login(func(err error) {
		close(ch)
	})
	<- ch
}

func OnAccount(account *goex.SubAccountDecimal) {
	log.Printf("OnAccount %+v", account)
}

func OnOrder(orders []goex.OrderDecimal) {
	log.Printf("OnOrder %+v", orders)
}

func TestOKExV3_Authenticated_Spot(t *testing.T) {
	ch := make(chan struct{})
	okexV3.Login(func(err error) {
		close(ch)
	})
	<- ch

	println("login success.")

	const instrumentId = "EOS-USDT"

	okexV3.GetAccountWithWs(goex.EOS, OnAccount)
	okexV3.GetOrderWithWs(instrumentId, OnOrder)

	time.Sleep(10 * time.Minute)
}
