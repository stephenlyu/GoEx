package zingex

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"net/http"
	"github.com/stretchr/testify/assert"
	"github.com/shopspring/decimal"
)

var zingEx *ZingEx

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		ApiKey    string    `json:"api-key"`
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
	zingEx = NewZingEx(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestZingEx_GetSymbols(t *testing.T) {
	ret, err := zingEx.GetSymbols()
	chk(err)
	output(ret)
}

func TestZingEx_GetTicker(t *testing.T) {
	ret, err := zingEx.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZingEx_GetDepth(t *testing.T) {
	ret, err := zingEx.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZingEx_GetTrades(t *testing.T) {
	ret, err := zingEx.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZingEx_GetAccount(t *testing.T) {
	ret, err := zingEx.GetAccount()
	chk(err)
	output(ret)
}

func TestZingEx_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	for i := 0; i < 100; i++ {
		orderId, err := zingEx.PlaceOrder(decimal.NewFromFloat(0.1), OrderBuy, code, decimal.NewFromFloat(5400))
		assert.Nil(t, err)
		output(orderId)
	}

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestZingEx_Sell(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := zingEx.PlaceOrder(decimal.NewFromFloat(0.01), OrderSell, code, decimal.NewFromFloat(5400))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestZingExCancelOrder(t *testing.T) {
	err := zingEx.CancelOrder("3")
	assert.Nil(t, err)
}

func TestZingExGetPendingOrders(t *testing.T) {
	code := "BTC_USDT"
	orders, err := zingEx.QueryPendingOrders(code)
	assert.Nil(t, err)
	output(orders)
}

func TestZingExGetOrder(t *testing.T) {
	code := "BTC_USDT"
	order, err := zingEx.QueryOrder(code, "4")
	assert.Nil(t, err)
	output(order)
}

func TestZingExCancelAll(t *testing.T) {
	code := "BTC_USDT"
	orders, err := zingEx.QueryPendingOrders(code)
	assert.Nil(t, err)
	for _, o := range orders {
		err = zingEx.CancelOrder(o.OrderID2)
		assert.Nil(t, err)
	}
}
