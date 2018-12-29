package bitfinex

import (
	"github.com/stephenlyu/GoEx"
	"net/http"
	"testing"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

var bfx = New(http.DefaultClient, "", "")

type Key struct {
	ApiKey string 	`json:"api-key"`
	SecretKey string `json:"secret-key"`
}

var (
	API_KEY = ""
	SECRET_KEY = ""
)

func init() {
	bytes, err := ioutil.ReadFile("key.json")
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)
	API_KEY = key.ApiKey
	SECRET_KEY = key.SecretKey
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func Output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBitfinex_GetTicker(t *testing.T) {
	ticker, _ := bfx.GetTicker(goex.ETH_BTC)
	t.Log(ticker)
}

func TestBitfinex_GetDepth(t *testing.T) {
	dep, _ := bfx.GetDepth(2, goex.ETH_BTC)
	t.Log(dep.AskList)
	t.Log(dep.BidList)
}

func TestBitfinex_OffersHistory(t *testing.T) {
	bfx := New(http.DefaultClient, API_KEY, SECRET_KEY)
	err, orders := bfx.OffersHistory(100)
	chk(err)
	Output(orders)
}

func TestBitfinex_OrderHistory(t *testing.T) {
	bfx := New(http.DefaultClient, API_KEY, SECRET_KEY)
	orders, err := bfx.GetOrderHistory(100)
	chk(err)
	Output(orders)
}
