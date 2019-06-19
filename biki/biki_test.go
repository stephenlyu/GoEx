package biki

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
	"github.com/shopspring/decimal"
)

var biki *Biki

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		ApiKey string 	`json:"api-key"`
		SecretKey string `json:"secret-key"`
	}

	var configFile = os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "key.json"
	}

	bytes, err := ioutil.ReadFile(configFile)
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)
	biki = NewBiki(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBiki_GetSymbols(t *testing.T) {
	ret, err := biki.GetSymbols()
	chk(err)
	output(ret)
}

func TestBiki_getPairByName(t *testing.T) {
	fmt.Println(biki.getPairByName("bikiusdt"))
}

func TestBiki_GetTicker(t *testing.T) {
	ret, err := biki.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBiki_GetDepth(t *testing.T) {
	api := NewBiki("", "")
	ret, err := api.GetDepth("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBiki_GetTrades(t *testing.T) {
	api := NewBiki("", "")
	ret, err := api.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBiki_GetAccount(t *testing.T) {
	ret, err := biki.GetAccount()
	chk(err)
	output(ret)
}

func TestBiki_PlaceOrder(t *testing.T) {
	code := "SHT_USDT"
	orderId, err := biki.PlaceOrder(decimal.NewFromFloat32(20), ORDER_BUY, ORDER_TYPE_LIMIT, code, decimal.NewFromFloat(0.041))
	assert.Nil(t, err)
	output(orderId)

	order, err := biki.QueryOrder(code, orderId)
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	code := "sht_usdt"
	err := biki.CancelOrder(code, "8603629")
	assert.Nil(t, err)
}

func TestOKExV3_GetPendingOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := biki.QueryPendingOrders(code, 0, 0)
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetOrder(t *testing.T) {
	code := "sht_usdt"
	order, err := biki.QueryOrder(code, "8603703")
	assert.Nil(t, err)
	output(order)
}
