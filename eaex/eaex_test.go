package eaex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var api *EAEX

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
	api = NewEAEX(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestEAEX_GetSymbols(t *testing.T) {
	ret, err := api.GetSymbols()
	chk(err)
	output(ret)
}

func TestEAEX_getPairByName(t *testing.T) {
	fmt.Println(api.getPairByName("eostrx"))
}

func TestEAEX_GetTicker(t *testing.T) {
	ret, err := api.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestEAEX_GetDepth(t *testing.T) {
	api := NewEAEX("", "")
	ret, err := api.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestEAEX_GetTrades(t *testing.T) {
	api := NewEAEX("", "")
	ret, err := api.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestEAEX_GetAccount(t *testing.T) {
	ret, err := api.GetAccount()
	chk(err)
	output(ret)
}

func TestEAEX_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := api.PlaceOrder(decimal.NewFromFloat32(10), ORDER_BUY, ORDER_TYPE_LIMIT, code, decimal.NewFromFloat(10))
	assert.Nil(t, err)
	output(orderId)
}

func TestEAEX_CancelOrder(t *testing.T) {
	err := api.CancelOrder("50")
	assert.Nil(t, err)
}

func TestEAEX_GetPendingOrders(t *testing.T) {
	code := "T1_USDT"
	orders, err := api.QueryPendingOrders(code, "", 100)
	assert.Nil(t, err)
	output(orders)
	println(len(orders))
}

func TestEAEX_GetOrder(t *testing.T) {
	code := "T1_USDT"
	order, err := api.QueryOrder(code, "712763827420185088")
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

// func Test_QueryAllDoneOrders(t *testing.T) {
// 	code := "T1_USDT"

// 	const pageSize = 100

// 	queryHisOrders := func(fromOrderId string) (orders []goex.OrderDecimal, err error) {
// 		for i := 0; i < 3; i++ {
// 			orders, err = api.QueryHisOrders(code, fromOrderId, pageSize)
// 			if err == nil {
// 				break
// 			}
// 			time.Sleep(time.Second)
// 		}
// 		return
// 	}

// 	var fromOrderId string
// 	var allOrders []goex.OrderDecimal

// 	for {
// 		orders, err := queryHisOrders(fromOrderId)
// 		assert.Nil(t, err)
// 		if len(orders) == 0 {
// 			break
// 		}
// 		fmt.Printf("Get page %s... lastId: %s\n", fromOrderId, orders[len(orders)-1].OrderID2)
// 		allOrders = append(allOrders, orders...)
// 		if len(allOrders) > 10000 {
// 			break
// 		}
// 		fromOrderId = orders[len(orders)-1].OrderID2
// 	}
// 	bytes, err := json.MarshalIndent(allOrders, "", "  ")
// 	assert.Nil(t, err)
// 	ioutil.WriteFile(code+"-orders.json", bytes, 0666)
// }
