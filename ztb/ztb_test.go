package ztb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var ztb *Ztb

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		ApiKey    string `json:"api-key"`
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
	ztb = NewZtb(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestZtb_GetSymbols(t *testing.T) {
	ret, err := ztb.GetSymbols()
	chk(err)
	output(ret)
}

func TestZtb_GetTicker(t *testing.T) {
	ret, err := ztb.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZtb_GetDepth(t *testing.T) {
	ret, err := ztb.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZtb_GetTrades(t *testing.T) {
	ret, err := ztb.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestZtb_GetAccount(t *testing.T) {
	ret, err := ztb.GetAccount()
	chk(err)
	output(ret)
}

func TestZtb_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := ztb.PlaceOrder(decimal.NewFromFloat(0.001), OrderBuy, OrderTypeLimit, code, decimal.NewFromFloat(8500))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestZtbCancelOrder(t *testing.T) {
	code := "BTC_USDT"
	err := ztb.CancelOrder(code, "159002570098514")
	assert.Nil(t, err)
}

func TestZtbGetPendingOrders(t *testing.T) {
	code := "BTC_USDT"
	orders, err := ztb.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestZtbGetOrder(t *testing.T) {
	code := "BTC_USDT"
	order, err := ztb.QueryOrder(code, "159002427396873")
	assert.Nil(t, err)
	output(order)
}

func TestZtb_BatchReplace(t *testing.T) {
	code := "BTC_USDT"

	reqList := []OrderReq{
		{
			Price:  8500,
			Amount: 0.001,
			Type:   OrderBuy,
		},
		{
			Price:  8501,
			Amount: 0.001,
			Type:   OrderBuy,
		},
	}

	orderIds, err := ztb.BatchPlace(code, reqList)
	assert.Nil(t, err)
	output(orderIds)
}

func TestZtbCancelAll(t *testing.T) {
	code := "BTC_USDT"
	orders, err := ztb.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)

	var orderIds []string
	for _, o := range orders {
		orderIds = append(orderIds, o.OrderID2)
	}

	cErrList, err := ztb.BatchCancel(code, orderIds)
	assert.Nil(t, err)
	fmt.Println(cErrList)
}

func TestZtbCancelAll1(t *testing.T) {
	code := "BTC_USDT"
	var orderIds = []string{
		"159002427396873",
	}

	cErrList, err := ztb.BatchCancel(code, orderIds)
	assert.Nil(t, err)
	fmt.Println(cErrList)
}
