package bitmex

import (
	"testing"
	"fmt"
	"github.com/stephenlyu/GoEx"
	"io/ioutil"
	"encoding/json"
)

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

func TestBitMexRest_GetTrade(t *testing.T) {
	bitmex := NewBitMexRest("", "")
	err, ret := bitmex.GetTrade(goex.NewCurrencyPair(goex.XBT, goex.USD), true)
	chk(err)
	fmt.Printf("%+v", ret)
}

func TestBitMexRest_GetOrderBook(t *testing.T) {
	bitmex := NewBitMexRest("", "")
	err, ret := bitmex.GetOrderBook(goex.NewCurrencyPair(goex.XBT, goex.USD))
	chk(err)
	fmt.Printf("%+v", ret)
}

func TestBitMexRest_GetMargin(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.GetAccount()
	chk(err)
	fmt.Println(ret)
}

func TestBitMexRest_GetPosition(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.GetPosition(goex.NewCurrencyPair(goex.XBT, goex.USD), 10)
	chk(err)
	Output(ret)
}

func TestBitMexRest_ListOrders(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.ListOrders(goex.NewCurrencyPair(goex.XBT, goex.USD), false, "", "", 50)
	chk(err)
	Output(ret)
}

func TestBitMexRest_ListExecutions(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.ListFills(goex.NewCurrencyPair(goex.XBT, goex.USD), "", "", 50)
	chk(err)
	Output(ret)
}

func TestBitMexRest_PlaceOrder(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.PlaceOrder(goex.NewCurrencyPair(goex.XBT, goex.USD), goex.SELL_MARKET, 0, 10, "")
	chk(err)
	fmt.Printf("%+v\n", ret)
}

func TestBitMexRest_CancelOrder(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.CancelOrder("b625db43-c6b4-b70c-3bac-564d1626721f", "")
	chk(err)
	fmt.Printf("%+v\n", ret)
}

func TestBitMexRest_CancelAll(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.CancelAll()
	chk(err)
	Output(ret)
}
