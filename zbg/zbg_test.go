package zbg

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
)

var zbg *ZBG

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
	zbg = NewZBG(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestZBG_GetMarketList(t *testing.T) {
	api := NewZBG("", "")
	ret, err := api.GetMarketList()
	chk(err)
	output(ret)
}

func TestZBG_GetCurrencyList(t *testing.T) {
	api := NewZBG("", "")
	ret, err := api.GetCurrencyList()
	chk(err)
	output(ret)
}

func TestZBG_GetTicker(t *testing.T) {
	api := NewZBG("", "")
	ret, err := api.GetTicker("ETC_USDT")
	chk(err)
	output(ret)
}

func TestZBG_GetDepth(t *testing.T) {
	api := NewZBG("", "")
	ret, err := api.GetDepth("ETC_USDT", 5)
	chk(err)
	output(ret)
}

func TestZBG_GetTrades(t *testing.T) {
	api := NewZBG("", "")
	ret, err := api.GetTrades("ETC_USDT", 5)
	chk(err)
	output(ret)
}

func TestZBG_GetAccount(t *testing.T) {
	ret, err := zbg.GetAccount(0, 0)
	chk(err)
	output(ret)
}

func TestZBG_PlaceOrder(t *testing.T) {
	code := "sht_usdt"
	orderId, err := zbg.PlaceOrder(10, ORDER_TYPE_BUY, code, 0.06)
	assert.Nil(t, err)
	output(orderId)

	order, err := zbg.QueryOrder(code, orderId)
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelOrder(t *testing.T) {
	code := "sht_usdt"
	err := zbg.CancelOrder(code, "E6542270567591006208")
	assert.Nil(t, err)
}

func TestZBG_QueryPendingOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := zbg.QueryPendingOrders(code)
	assert.Nil(t, err)
	output(orders)
}

func TestZBG_QueryOrder(t *testing.T) {
	code := "sht_usdt"
	order, err := zbg.QueryOrder(code, "E6542270567591006208")
	assert.Nil(t, err)
	output(order)
}
