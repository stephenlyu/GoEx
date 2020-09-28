package biki

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
	"github.com/stretchr/testify/assert"
)

var biki *Biki

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		APIKey    string `json:"api-key"`
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
	biki = NewBiki(key.APIKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestBiki_GetSymbols(t *testing.T) {
	ret, err := biki.GetSymbols()
	chk(err)
	output(ret)
}

func TestBiki_getPairByName(t *testing.T) {
	fmt.Println(biki.getPairByName("bikiusdt"))
}

func TestBiki_GetTicker(t *testing.T) {
	ret, err := biki.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestBiki_GetDepth(t *testing.T) {
	api := NewBiki("", "")
	ret, err := api.GetDepth("GUNG_ODIN")
	chk(err)
	output(ret)
}

func TestBiki_GetTrades(t *testing.T) {
	api := NewBiki("", "")
	ret, err := api.GetTrades("ETC_USDT")
	chk(err)
	output(ret)
}

func TestBiki_GetAccount(t *testing.T) {
	ret, err := biki.GetAccount()
	chk(err)
	output(ret)
}

func TestBiki_PlaceOrder(t *testing.T) {
	code := "SHT_USDT"
	orderID, err := biki.PlaceOrder(decimal.NewFromFloat32(500), OrerSell, OrderTypeLimit, code, decimal.NewFromFloat(0.04))
	assert.Nil(t, err)
	output(orderID)

	order, err := biki.QueryOrder(code, orderID)
	assert.Nil(t, err)
	output(order)
}

func TestBiki_MassPlaceFail(t *testing.T) {
	code := "GUNG_ODIN"
	reqList := []OrderReq{
		{
			Side:   OrderBuy,
			Type:   OrderTypeLimitStr,
			Volume: 0.0001,
			Price:  35000,
		},
		{
			Side:   OrderBuy,
			Type:   OrderTypeLimitStr,
			Volume: 0,
			Price:  35001,
		},
	}

	orderIDs, placeErrors, _, err := biki.MassReplace(code, nil, reqList)
	fmt.Println(err)
	fmt.Println(placeErrors)
	fmt.Println(orderIDs)
}

func TestBiki_MassPlaceSuccess(t *testing.T) {
	code := "GUNG_ODIN"
	reqList := []OrderReq{
		{
			Side:   OrderBuy,
			Type:   OrderTypeLimitStr,
			Volume: 0.0001,
			Price:  10000,
		},
		{
			Side:   OrderBuy,
			Type:   OrderTypeLimitStr,
			Volume: 0.0001,
			Price:  10001,
		},
	}

	orderIDs, placeErrors, _, err := biki.MassReplace(code, nil, reqList)
	fmt.Println(err)
	fmt.Println(placeErrors)
	fmt.Println(orderIDs)
}

func TestBiki_MassCancelSuccess(t *testing.T) {
	code := "GUNG_ODIN"

	_, _, cancelErrors, err := biki.MassReplace(code, []string{"1770", "1771"}, nil)
	fmt.Println(err)
	fmt.Println(cancelErrors)
}

func TestBiki_MassCancelFail(t *testing.T) {
	code := "GUNG_ODIN"

	_, _, cancelErrors, err := biki.MassReplace(code, []string{"10000", "1771"}, nil)
	fmt.Println(err)
	fmt.Println(cancelErrors)
}

func TestBiki_FutureCancelOrder(t *testing.T) {
	code := "sht_usdt"
	err := biki.CancelOrder(code, "10278430")
	assert.Nil(t, err)
}

func TestBiki_GetPendingOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := biki.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestBiki_GetOrder(t *testing.T) {
	code := "GUNG_ODIN"
	order, err := biki.QueryOrder(code, "1770")
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelAll(t *testing.T) {
	code := "sht_usdt"
	orders, err := biki.QueryPendingOrders(code, 1, 100)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		err = biki.CancelOrder(code, o.OrderID2)
		fmt.Println(err)
	}
}

func TestBiki_GetALLOrders(t *testing.T) {
	code := "sht_usdt"
	orders, err := biki.QueryAllOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)
}

func TestZBG_QueryAllDoneOrders(t *testing.T) {
	code := "sht_usdt"

	const pageSize = 100

	queryDoneOrders := func(page int) (orders []goex.OrderDecimal, err error) {
		for i := 0; i < 3; i++ {
			orders, err = biki.QueryAllOrders(code, page, pageSize)
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		return
	}

	var page = 1
	var allOrders []goex.OrderDecimal

	for {
		orders, err := queryDoneOrders(page)
		assert.Nil(t, err)
		if len(orders) == 0 {
			break
		}
		fmt.Printf("Get page %d... lastId: %s\n", page, orders[len(orders)-1].OrderID2)
		allOrders = append(allOrders, orders...)
		if len(allOrders) > 5000 {
			break
		}
		page++
	}
	bytes, err := json.MarshalIndent(allOrders, "", "  ")
	assert.Nil(t, err)
	ioutil.WriteFile(code+"-orders.json", bytes, 0666)
}
