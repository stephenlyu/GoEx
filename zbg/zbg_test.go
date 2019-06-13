package zbg

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
	"github.com/shopspring/decimal"
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
	fmt.Println(len(ret))
	output(ret)
}

func TestZBG_PlaceOrder(t *testing.T) {
	code := "sht_usdt"
	orderId, err := zbg.PlaceOrder(decimal.NewFromFloat(21.1804), ORDER_TYPE_BUY, code, decimal.NewFromFloat(0.0547))
	assert.Nil(t, err)
	output(orderId)

	order, err := zbg.QueryOrder(code, orderId)
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelOrder(t *testing.T) {
	code := "sht_usdt"
	err := zbg.CancelOrder(code, "E6542313405284438016")
	assert.Nil(t, err)
}

func TestZBG_QueryPendingOrders(t *testing.T) {
	code := "sht_qc"
	orders, err := zbg.QueryPendingOrders(code)
	assert.Nil(t, err)
	fmt.Println(len(orders))
	output(orders)
}

func TestZBG_QueryPagedPendingOrders(t *testing.T) {
	code := "sht_qc"
	orders, err := zbg.QueryPagedPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	fmt.Println(len(orders))
	output(orders)
}

func TestZBG_QueryDoneOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := zbg.QueryDoneOrders(code)
	assert.Nil(t, err)
	output(orders)
}

func TestZBG_CancelAll(t *testing.T) {
	code := "sht_usdt"
	orders, err := zbg.QueryPendingOrders(code)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		zbg.CancelOrder(code, o.OrderID2)
	}
}

func TestZBG_QueryOrder(t *testing.T) {
	code := "sht_usdt"
	order, err := zbg.QueryOrder(code, "E6542270567591006208")
	assert.Nil(t, err)
	output(order)
}
