package binancefuture

import (
	"github.com/stephenlyu/GoEx"
	"net/http"
	"testing"
	"fmt"
	"encoding/json"
)

var ba = New(http.DefaultClient, "", "")

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBinance_GetExchangeInfo(t *testing.T) {
	exchange, err := ba.GetExchangeInfo()
	chk(err)
	output(exchange)
}

func TestBinance_GetTicker(t *testing.T) {
	ticker, _ := ba.GetTicker(goex.BTC_USDT)
	output(ticker)
}

func TestBinance_LimitSell(t *testing.T) {
	order, err := ba.LimitSell("1", "1", goex.LTC_BTC)
	t.Log(order, err)
}

func TestBinance_GetDepth(t *testing.T) {
	dep, err := ba.GetDepth(5, goex.BTC_USDT)
	t.Log(err)
	output(dep)
}

func TestBinance_GetTrades(t *testing.T) {
	dep, err := ba.GetTrades(goex.BTC_USDT)
	t.Log(err)
	output(dep)
}

func TestBinance_GetAccount(t *testing.T) {
	account, err := ba.GetAccount()
	t.Log(account, err)
}

func TestBinance_GetUnfinishOrders(t *testing.T) {
	orders, err := ba.GetUnfinishOrders(goex.ETH_BTC)
	t.Log(orders, err)
}
