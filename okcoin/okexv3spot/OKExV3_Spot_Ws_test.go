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
//
//func TestOKExV3_Login(t *testing.T) {
//	ch := make(chan struct{})
//	okexV3.Login(func(err error) {
//		close(ch)
//	})
//	<- ch
//}
//
//func OnAccount(isSwap bool, account *goex.FutureAccount) {
//	log.Printf("OnAccount %+v", account)
//}
//
//func OnPosition(positions []goex.FuturePosition) {
//	log.Printf("OnPosition %+v", positions)
//}
//
//func OnOrder(orders []goex.FutureOrder) {
//	log.Printf("OnOrder %+v", orders)
//}
//
//func TestOKExV3_Authenticated_Futures(t *testing.T) {
//	ch := make(chan struct{})
//	okexV3.Login(func(err error) {
//		close(ch)
//	})
//	<- ch
//
//	const instrumentId = "EOS-USD-190329"
//
//	okexV3.GetAccountWithWs(goex.EOS, false, OnAccount)
//	okexV3.GetPositionWithWs(instrumentId, OnPosition)
//	okexV3.GetOrderWithWs(instrumentId, OnOrder)
//
//	time.Sleep(10 * time.Minute)
//}
