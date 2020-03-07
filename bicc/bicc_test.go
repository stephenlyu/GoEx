package bicc

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

var bicc *Bicc

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
	bicc = NewBicc(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBicc_GetSymbols(t *testing.T) {
	ret, err := bicc.GetSymbols()
	chk(err)
	output(ret)
}

func TestBicc_getPairByName(t *testing.T) {
	fmt.Println(bicc.getPairByName("btcusdt"))
}

func TestBicc_GetTicker(t *testing.T) {
	ret, err := bicc.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBicc_GetDepth(t *testing.T) {
	ret, err := bicc.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBicc_GetTrades(t *testing.T) {
	ret, err := bicc.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBicc_GetAccount(t *testing.T) {
	ret, err := bicc.GetAccount()
	chk(err)
	output(ret)
}

func TestBicc_PlaceOrder(t *testing.T) {
	code := "EOS_USDT"
	orderId, err := bicc.PlaceOrder(decimal.NewFromFloat(0.528), OrderSell, OrderTypeMarket, code, decimal.NewFromFloat(3.6))
	assert.Nil(t, err)
	output(orderId)

	//order, err := bitribe.QueryOrder(orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestBiccCancelOrder(t *testing.T) {
	code := "EOS_USDT"
	err := bicc.CancelOrder(code, "42948624")
	assert.Nil(t, err)
}

func TestBiccGetPendingOrders(t *testing.T) {
	code := "EOS_USDT"
	orders, err := bicc.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestBiccGetOrder(t *testing.T) {
	code := "EOS_USDT"
	order, err := bicc.QueryOrder(code, "42958597")
	assert.Nil(t, err)
	output(order)
}

func TestBicc_BatchReplace(t *testing.T) {
	code := "EOS_USDT"

	reqList := []OrderReq{
		{
			Price: decimal.NewFromFloat(3.6),
			Volume: decimal.NewFromFloat(10),
			Type: OrderTypeLimit,
			Side: OrderBuy,
		},
		{
			Price: decimal.NewFromFloat(3.601),
			Volume: decimal.NewFromFloat(10),
			Type: OrderTypeLimit,
			Side: OrderBuy,
		},
	}

	cancelOrderIds := []string {
		//"42950600",
	}

	cErrList, orderIds, pErrList, err := bicc.BatchReplace(code, cancelOrderIds, reqList)
	assert.Nil(t, err)
	fmt.Println(cErrList)
	output(orderIds)
	fmt.Println(pErrList)
}

func TestBiccCancelAll(t *testing.T) {
	code := "EOS_USDT"
	orders, err := bicc.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)

	var orderIds []string
	for _, o := range orders {
		orderIds = append(orderIds, o.OrderID2)
	}

	cErrList, orderIds, pErrList, err := bicc.BatchReplace(code, orderIds, nil)
	assert.Nil(t, err)
	fmt.Println(cErrList)
	output(orderIds)
	fmt.Println(pErrList)
}

