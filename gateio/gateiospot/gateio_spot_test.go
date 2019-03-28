package gateiospot

import (
	"testing"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stephenlyu/GoEx"
	"github.com/shopspring/decimal"
	"sync"
)

var (
	gateioSpot *GateIOSpot
)

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

	bytes, err := ioutil.ReadFile("key.json")
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)
	gateioSpot = NewGateIOSpot(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestGateIOSpot_GetPairs(t *testing.T) {
	ret, err := gateioSpot.GetPairs()
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetMarketInfo(t *testing.T) {
	ret, err := gateioSpot.GetMarketInfo()
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetTicker(t *testing.T) {
	ret, err := gateioSpot.GetTicker(goex.EOS_USDT)
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetOrderBook(t *testing.T) {
	ret, err := gateioSpot.GetOrderBook(goex.EOS_USDT)
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetTrades(t *testing.T) {
	ret, err := gateioSpot.GetTrades(goex.EOS_USDT)
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetAccount(t *testing.T) {
	ret, err := gateioSpot.GetAccount()
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_PlaceOrder(t *testing.T) {
	pair := goex.CurrencyPair{
		CurrencyA: goex.Currency{Symbol:"BU"},
		CurrencyB: goex.Currency{Symbol:"ETH"},
	}
	ret, err := gateioSpot.PlaceOrder("buy", pair, decimal.NewFromFloat(0.0002893), decimal.NewFromFloat(10))
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_SelfTrade(t *testing.T) {
	pair := goex.CurrencyPair{
		CurrencyA: goex.Currency{Symbol:"BU"},
		CurrencyB: goex.Currency{Symbol:"ETH"},
	}

	price := decimal.NewFromFloat(0.0002850)
	amount := decimal.NewFromFloat(5)
	wg := sync.WaitGroup{}
	go func() {
		ret, err := gateioSpot.PlaceOrder("buy", pair, price, amount)
		assert.Nil(t, err)
		output(ret)
		wg.Done()
	} ()
	go func() {
		ret, err := gateioSpot.PlaceOrder("sell", pair, price, amount)
		assert.Nil(t, err)
		output(ret)
		wg.Done()
	} ()

	wg.Add(2)
	wg.Wait()
}

func TestGateIOSpot_CancelOrder(t *testing.T) {
	err := gateioSpot.CancelOrder(goex.ETH_USDT, "3036638558")
	assert.Nil(t, err)
}

func TestGateIOSpot_CancelOrders(t *testing.T) {
	err := gateioSpot.CancelOrders(goex.ETH_USDT, []string{"3024214382"})
	assert.Nil(t, err)
}

func TestGateIOSpot_CancelAllOrders(t *testing.T) {
	err := gateioSpot.CancelAllOrders(goex.ETH_USDT, CancelAllOrdersTypeSell)
	assert.Nil(t, err)
}

func TestGateIOSpot_GetOrder(t *testing.T) {
	ret, err := gateioSpot.GetOrder(goex.ETH_USDT, "3024225306")
	assert.Nil(t, err)
	output(ret)
}

func TestGateIOSpot_GetOpenOrders(t *testing.T) {
	ret, err := gateioSpot.GetOpenOrders(goex.ETH_USDT)
	assert.Nil(t, err)
	output(ret)
}
