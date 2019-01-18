package okcoin

import (
	"testing"
	"net/http"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var okexFutureV3 = NewOKExV3(http.DefaultClient, "", "", "")

func TestOKExV3_GetDepthWithWs(t *testing.T) {
	okexFutureV3.GetDepthWithWs("EOS-USD-190329", func(depth *goex.Depth) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}

func TestOKExV3_GetTradeWithWs(t *testing.T) {
	okexFutureV3.GetTradeWithWs("EOS-USD-SWAP", func(instrumentId string, trades []goex.Trade) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	okexFuture.ws.CloseWs()
}
