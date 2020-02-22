package bitribe

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

var bitribe *Bitribe

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
	bitribe = NewBitribe(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBitribe_GetSymbols(t *testing.T) {
	ret, err := bitribe.GetSymbols()
	chk(err)
	output(ret)
}

func TestBitribe_getPairByName(t *testing.T) {
	fmt.Println(bitribe.getPairByName("BTCUSDT"))
}

func TestBitribe_GetTicker(t *testing.T) {
	ret, err := bitribe.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBitribe_GetDepth(t *testing.T) {
	ret, err := bitribe.GetDepth("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBitribe_GetTrades(t *testing.T) {
	ret, err := bitribe.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBitribe_GetAccount(t *testing.T) {
	ret, err := bitribe.GetAccount()
	chk(err)
	output(ret)
}

func TestBitribe_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := bitribe.PlaceOrder(decimal.NewFromFloat(0.01), OrderSell, OrderTypeLimit, code, decimal.NewFromFloat(10000))
	assert.Nil(t, err)
	output(orderId)

	order, err := bitribe.QueryOrder(orderId)
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	err := bitribe.CancelOrder("569461796694531584", "")
	assert.Nil(t, err)
}

func TestOKExV3_GetPendingOrders(t *testing.T) {
	code := "BTC_USDT"
	orders, err := bitribe.QueryPendingOrders(code, "", 100)
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetOrder(t *testing.T) {
	order, err := bitribe.QueryOrder("56937054442685235")
	assert.Nil(t, err)
	output(order)
}