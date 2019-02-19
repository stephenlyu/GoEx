package bitmex

import (
	"testing"
	"github.com/stephenlyu/GoEx"
	"log"
	"time"
)

var bitmexWs = NewBitMexWs("", "")

func TestBitMexWs_GetDepthWithWs(t *testing.T) {
	bitmexWs.GetDepthWithWs("XBTM19", func(depth *goex.Depth) {
		log.Printf("%+v\n", depth)
	})
	time.Sleep(1 * time.Minute)
	bitmexWs.ws.CloseWs()
}

func TestBitMexWs_GetTradeWithWs(t *testing.T) {
	bitmexWs.GetTradeWithWs("XBTM19", func(symbol string, trades []goex.Trade) {
		log.Println(trades)
	})
	time.Sleep(10 * time.Minute)
	bitmexWs.ws.CloseWs()
}

func TestBitMexWs_GetOrderWithWs(t *testing.T) {
	bitmexWs = NewBitMexWs(API_KEY, SECRET_KEY)

	bitmexWs.SetErrorHandler(func (err error) {
		log.Fatalf("Error: %+v", err)
	})

	err := bitmexWs.Authenticate()
	chk(err)

	bitmexWs.GetOrderWithWs(func(orders []goex.FutureOrder) {
		log.Printf("%+v", orders)
	})

	bitmexWs.GetAccountWithWs(func(account *goex.FutureAccount) {
		log.Printf("%+v", account)
	})
	//
	bitmexWs.GetFillWithWs(func(fills []goex.FutureFill) {
		log.Printf("%+v", fills)
	})

	bitmexWs.GetPositionWithWs(func(positions []goex.FuturePosition) {
		log.Printf("%+v", positions)
	})

	time.Sleep(1000 * time.Minute)
	bitmexWs.ws.CloseWs()
}
