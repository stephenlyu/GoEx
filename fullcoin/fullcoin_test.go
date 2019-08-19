package fullcoin

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var fullCoin *FullCoin

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
	fullCoin = NewFullCoin(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestFullCoin_GetSymbols(t *testing.T) {
	ret, err := fullCoin.GetSymbols()
	chk(err)
	output(ret)
}

func TestFullCoin_getPairByName(t *testing.T) {
	fmt.Println(fullCoin.getPairByName("btcusdt"))
}

func TestFullCoin_GetTicker(t *testing.T) {
	ret, err := fullCoin.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestFullCoin_GetDepth(t *testing.T) {
	api := NewFullCoin("", "")
	ret, err := api.GetDepth("PDRR_USDT")
	chk(err)
	output(ret)
}

func TestFullCoin_GetTrades(t *testing.T) {
	api := NewFullCoin("", "")
	ret, err := api.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestFullCoin_GetAccounts(t *testing.T) {
	ret, err := fullCoin.GetAccounts()
	chk(err)
	output(ret)
}

func TestFullCoin_PlaceOrder(t *testing.T) {
	code := "PDRR_USDT"
	orderId, err := fullCoin.PlaceOrder(decimal.NewFromFloat32(10), SIDE_BUY, TYPE_LIMIT, code, decimal.NewFromFloat(0.015))
	assert.Nil(t, err)
	output(orderId)
}

func TestFullCoin_CancelOrder(t *testing.T) {
	err := fullCoin.CancelOrder("PDRR_USDT", "6999")
	assert.Nil(t, err)
}

func TestFullCoin_GetPendingOrders(t *testing.T) {
	code := "PDRR_USDT"
	orders, err := fullCoin.QueryPendingOrders(code, 1, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestFullCoin_GetOrder(t *testing.T) {
	order, err := fullCoin.QueryOrder("PDRR_USDT", "69999")
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelAll(t *testing.T) {
	code := "sht_usdt"
	orders, err := fullCoin.QueryPendingOrders(code, 1, 100)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		err = fullCoin.CancelOrder(code, o.OrderID2)
		fmt.Println(err)
	}
}
