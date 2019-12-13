package deerdex

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var api *DeerDex

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
	api = NewDeerDex(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestDeerDex_GetSymbols(t *testing.T) {
	ret, err := api.GetSymbols()
	chk(err)
	output(ret)
}

func TestDeerDex_getPairByName(t *testing.T) {
	fmt.Println(api.getPairByName("eostrx"))
}

func TestDeerDex_GetTicker(t *testing.T) {
	ret, err := api.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestDeerDex_GetDepth(t *testing.T) {
	api := NewDeerDex("", "")
	ret, err := api.GetDepth("ETC_USDT")
	chk(err)
	output(ret)
}

func TestDeerDex_GetTrades(t *testing.T) {
	api := NewDeerDex("", "")
	ret, err := api.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestDeerDex_GetAccount(t *testing.T) {
	ret, err := api.GetAccount()
	chk(err)
	output(ret)
}

func TestDeerDex_PlaceOrder(t *testing.T) {
	code := "LEEE_USDT"
	orderId, err := api.PlaceOrder(decimal.NewFromFloat32(100), ORDER_SELL, ORDER_TYPE_LIMIT, code, decimal.NewFromFloat(0.005))
	assert.Nil(t, err)
	output(orderId)
}

func TestDeerDex_CancelOrder(t *testing.T) {
	err := api.CancelOrder("515778354371116288")
	assert.Nil(t, err)
}

func TestDeerDex_GetPendingOrders(t *testing.T) {
	code := "LEEE_USDT"
	orders, err := api.QueryPendingOrders(code, "", 100)
	assert.Nil(t, err)
	output(orders)
}

func TestDeerDex_GetOrder(t *testing.T) {
	code := "LEEE_USDT"
	order, err := api.QueryOrder(code, "515104109123199488")
	assert.Nil(t, err)
	output(order)
}


//func TestZBG_CancelAll(t *testing.T) {
//	code := "sht_usdt"
//	orders, err := api.QueryPendingOrders(code, 1, 100)
//	assert.Nil(t, err)
//	output(orders)
//
//	for _, o := range orders {
//		err = api.CancelOrder(code, o.OrderID2)
//		fmt.Println(err)
//	}
//}
//
//func TestOKExV3_GetALLOrders(t *testing.T) {
//	code := "sht_usdt"
//	orders, err := api.QueryAllOrders(code, 0, 100)
//	assert.Nil(t, err)
//	output(orders)
//}
//
//func TestZBG_QueryAllDoneOrders(t *testing.T) {
//	code := "sht_usdt"
//
//	const pageSize = 100
//
//	queryDoneOrders := func(page int) (orders []goex.OrderDecimal, err error) {
//		for i := 0; i < 3; i++ {
//			orders, err = api.QueryAllOrders(code, page, pageSize)
//			if err == nil {
//				break
//			}
//			time.Sleep(time.Second)
//		}
//		return
//	}
//
//	var page = 1
//	var allOrders []goex.OrderDecimal
//
//	for {
//		orders, err := queryDoneOrders(page)
//		assert.Nil(t, err)
//		if len(orders) == 0 {
//			break
//		}
//		fmt.Printf("Get page %d... lastId: %s\n", page, orders[len(orders) - 1].OrderID2)
//		allOrders = append(allOrders, orders...)
//		if len(allOrders) > 5000 {
//			break
//		}
//		page++
//	}
//	bytes, err := json.MarshalIndent(allOrders, "", "  ")
//	assert.Nil(t, err)
//	ioutil.WriteFile(code + "-orders.json", bytes, 0666)
//}

func TestDeerDex_CreateListenKey(t *testing.T) {
	ret, err := api.CreateListenKey()
	chk(err)
	output(ret)

	err = api.ListenKeyKeepAlive(ret)
	chk(err)
}