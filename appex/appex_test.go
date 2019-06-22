package appex

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"time"
)

var appex *Appex

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
	appex = NewAppex(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestAppex_GetSymbols(t *testing.T) {
	ret, err := appex.GetSymbols()
	chk(err)
	output(ret)
}

func TestAppex_getPairByName(t *testing.T) {
	fmt.Println(appex.getPairByName("btcusdt"))
}

func TestAppex_GetTicker(t *testing.T) {
	ret, err := appex.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestAppex_GetDepth(t *testing.T) {
	api := NewAppex("", "")
	ret, err := api.GetDepth("ETC_USDT")
	chk(err)
	output(ret)
}

func TestAppex_GetTrades(t *testing.T) {
	api := NewAppex("", "")
	ret, err := api.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestAppex_GetAccounts(t *testing.T) {
	ret, err := appex.GetAccounts()
	chk(err)
	output(ret)
}

func TestAppex_GetAccountBalance(t *testing.T) {
	ret, err := appex.GetAccountBalance(8497920)
	chk(err)
	output(ret)
}

func TestAppex_GetAccount(t *testing.T) {
	ret, err := appex.GetAccount()
	chk(err)
	output(ret)
}

func TestAppex_PlaceOrder(t *testing.T) {
	code := "SHT_USDT"
	orderId, err := appex.PlaceOrder(decimal.NewFromFloat32(10), SIDE_BUY, TYPE_LIMIT, code, decimal.NewFromFloat(0.05))
	assert.Nil(t, err)
	output(orderId)

	order, err := appex.QueryOrder(orderId)
	assert.Nil(t, err)
	output(order)
}

func TestAppex_CancelOrder(t *testing.T) {
	err := appex.CancelOrder("37800595691")
	assert.Nil(t, err)
}

func TestAppex_GetPendingOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := appex.QueryPendingOrders(code, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestAppex_Freq(t *testing.T) {
	code := "sht_usdt"
	for i := 0; i < 100; i++ {
		_, err := appex.QueryPendingOrders(code, 100)
		fmt.Println(err)
		if err != nil {
			time.Sleep(time.Second)
		}
	}
}

func TestAppex_GetOrder(t *testing.T) {
	order, err := appex.QueryOrder("37800595691")
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelAll(t *testing.T) {
	code := "sht_usdt"
	orders, err := appex.QueryPendingOrders(code, 100)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		err = appex.CancelOrder(o.OrderID2)
		fmt.Println(err)
	}
}
