package bibull

import (
	"log"
	"net/http"
	"testing"
	"time"

	goex "github.com/stephenlyu/GoEx"
)

var bibullAPI = NewBiBull(http.DefaultClient, "", "")

func TestBiBull_GetTradeWithWs(t *testing.T) {
	bibullAPI.GetTradeWithWs("BTC_USDT", func(symbol string, trades []goex.TradeDecimal) {
		log.Printf("%+v\n", trades)
	})
	time.Sleep(10 * time.Minute)
	bibullAPI.ws.CloseWs()
}

func TestBiBull_GetDepthWithWs(t *testing.T) {
	bibullAPI.GetDepthWithWs("BTC_USDT", func(depth *goex.DepthDecimal) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(10 * time.Minute)
	bibullAPI.ws.CloseWs()
}
