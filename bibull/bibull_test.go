package bibull

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

var bibull *BiBull

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
	bibull = NewBiBull(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBiBull_GetSymbols(t *testing.T) {
	ret, err := bibull.GetSymbols()
	chk(err)
	output(ret)
}

func TestBiBull_getPairByName(t *testing.T) {
	fmt.Println(bibull.getPairByName("btcusdt"))
}

func TestBiBull_GetTicker(t *testing.T) {
	ret, err := bibull.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBiBull_GetDepth(t *testing.T) {
	ret, err := bibull.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBiBull_GetTrades(t *testing.T) {
	ret, err := bibull.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBiBull_GetAccount(t *testing.T) {
	ret, err := bibull.GetAccount()
	chk(err)
	output(ret)
}

func TestBiBull_PlaceOrder(t *testing.T) {
	code := "EOS_USDT"
	orderId, err := bibull.PlaceOrder(decimal.NewFromFloat(0.1), OrderBuy, OrderTypeLimit, code, decimal.NewFromFloat(2))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestBiBullCancelOrder(t *testing.T) {
	code := "EOS_USDT"
	err := bibull.CancelOrder(code, "63523150424935438")
	assert.Nil(t, err)
}

func TestBiBullGetPendingOrders(t *testing.T) {
	code := "EOS_USDT"
	orders, err := bibull.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestBiBullGetOrder(t *testing.T) {
	code := "EOS_USDT"
	order, err := bibull.QueryOrder(code, "63523150424935438")
	assert.Nil(t, err)
	output(order)
}

func TestBiBull_BatchReplace(t *testing.T) {
	code := "EOS_USDT"

	reqList := []OrderReq{
		{
			Price:  decimal.NewFromFloat(3.6),
			Volume: decimal.NewFromFloat(10),
			Type:   OrderTypeLimit,
			Side:   OrderBuy,
		},
		{
			Price:  decimal.NewFromFloat(3.601),
			Volume: decimal.NewFromFloat(10),
			Type:   OrderTypeLimit,
			Side:   OrderBuy,
		},
	}

	cancelOrderIds := []string{
		//"42950600",
	}

	cErrList, orderIds, pErrList, err := bibull.BatchReplace(code, cancelOrderIds, reqList)
	assert.Nil(t, err)
	fmt.Println(cErrList)
	output(orderIds)
	fmt.Println(pErrList)
}

func TestBiBullCancelAll(t *testing.T) {
	code := "EOS_USDT"
	orders, err := bibull.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)

	var orderIds []string
	for _, o := range orders {
		orderIds = append(orderIds, o.OrderID2)
	}

	cErrList, orderIds, pErrList, err := bibull.BatchReplace(code, orderIds, nil)
	assert.Nil(t, err)
	fmt.Println(cErrList)
	output(orderIds)
	fmt.Println(pErrList)
}
