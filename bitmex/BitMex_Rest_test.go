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
	err, ret := bitmex.GetTrade("XBTUSD")
	chk(err)
	fmt.Printf("%+v", ret)
}

func TestBitMexRest_GetOrderBook(t *testing.T) {
	bitmex := NewBitMexRest("", "")
	err, ret := bitmex.GetOrderBook("XBTUSD")
	chk(err)
	fmt.Printf("%+v", ret)
}

func TestBitMexRest_GetMargin(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.GetMargin()
	chk(err)
	fmt.Printf("%+v", ret)
}

func TestBitMexRest_GetPosition(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.GetPosition("XBTUSD", 10)
	chk(err)
	Output(ret)
}

func TestBitMexRest_ListOrders(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.ListOrders("XBTUSD", false, "", "", 50)
	chk(err)
	fmt.Printf("%+v\n", ret)
}

func TestBitMexRest_ListExecutions(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.ListFills("XBTUSD", "", "", 50)
	chk(err)
	fmt.Printf("%+v\n", ret)
}

func TestBitMexRest_PlaceOrder(t *testing.T) {
	bitmex := NewBitMexRest(API_KEY, SECRET_KEY)
	err, ret := bitmex.PlaceOrder("XBTUSD", goex.SELL, 6600, 10, "")
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
