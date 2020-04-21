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
	"github.com/stephenlyu/GoEx"
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
	ret, err := zingEx.GetTicker("ODIN_USDT")
	chk(err)
	output(ret)
}

func TestZingEx_GetDepth(t *testing.T) {
	ret, err := zingEx.GetDepth("ODIN_USDT")
	chk(err)
	output(ret)
}

func TestZingEx_GetTrades(t *testing.T) {
	ret, err := zingEx.GetTrades("LEEE_ETH")
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
	orderId, err := zingEx.PlaceOrder(decimal.NewFromFloat(0.0015), OrderBuy, code, decimal.NewFromFloat(5400))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestZingEx_Sell(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := zingEx.PlaceOrder(decimal.NewFromFloat(0.002), OrderSell, code, decimal.NewFromFloat(5400))
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
	code := "ODIN_USDT"
	orders, err := zingEx.QueryPendingOrders(code)
	assert.Nil(t, err)

	var bids []goex.OrderDecimal
	var asks []goex.OrderDecimal
	for _, order := range orders {
		if order.Side == goex.BUY || order.Side == goex.BUY_MARKET {
			bids = append(bids, order)
		} else {
			asks = append(asks, order)
		}
	}
	fmt.Printf("bid count: %d ask count: %d", len(bids), len(asks))

	//output(orders)
}

func TestZingExGetOrder(t *testing.T) {
	code := "ODIN_USDT"
	order, err := zingEx.QueryOrder(code, "22534753")
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

func TestGetPositionStatistics(t *testing.T) {
	stat, err := zingEx.GetPositionStatistics()
	assert.Nil(t, err)
	output(stat)
}
