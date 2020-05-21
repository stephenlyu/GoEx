package atop

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

var atop *Atop

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
	atop = NewAtop(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestAtop_GetSymbols(t *testing.T) {
	ret, err := atop.GetSymbols()
	chk(err)
	output(ret)
}

func TestAtop_GetTicker(t *testing.T) {
	ret, err := atop.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestAtop_GetDepth(t *testing.T) {
	ret, err := atop.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestAtop_GetTrades(t *testing.T) {
	ret, err := atop.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestAtop_GetAccount(t *testing.T) {
	ret, err := atop.GetAccount()
	chk(err)
	output(ret)
}

func TestAtop_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := atop.PlaceOrder(decimal.NewFromFloat(0.001), OrderBuy, OrderTypeLimit, code, decimal.NewFromFloat(8500))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestAtopCancelOrder(t *testing.T) {
	code := "BTC_USDT"
	err := atop.CancelOrder(code, "159002570098514")
	assert.Nil(t, err)
}

func TestAtopGetPendingOrders(t *testing.T) {
	code := "BTC_USDT"
	orders, err := atop.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestAtopGetOrder(t *testing.T) {
	code := "BTC_USDT"
	order, err := atop.QueryOrder(code, "159002427396873")
	assert.Nil(t, err)
	output(order)
}

func TestAtop_BatchReplace(t *testing.T) {
	code := "BTC_USDT"

	reqList := []OrderReq{
		{
			Price: 8500,
			Amount: 0.001,
			Type: OrderBuy,
		},
		{
			Price: 8501,
			Amount: 0.001,
			Type: OrderBuy,
		},
	}

	orderIds, err := atop.BatchPlace(code, reqList)
	assert.Nil(t, err)
	output(orderIds)
}

func TestAtopCancelAll(t *testing.T) {
	code := "BTC_USDT"
	orders, err := atop.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)

	var orderIds []string
	for _, o := range orders {
		orderIds = append(orderIds, o.OrderID2)
	}

	cErrList, err := atop.BatchCancel(code, orderIds)
	assert.Nil(t, err)
	fmt.Println(cErrList)
}

func TestAtopCancelAll1(t *testing.T) {
	code := "BTC_USDT"
	var orderIds = []string{
		"159002427396873",
	}

	cErrList, err := atop.BatchCancel(code, orderIds)
	assert.Nil(t, err)
	fmt.Println(cErrList)
}

