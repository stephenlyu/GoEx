package huobifuture

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"time"
	"net/http"
)

var huobi *HuobiFuture

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
	huobi = NewHuobiFuture(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestHuobiFuture_GetContractInfo(t *testing.T) {
	ret, err := huobi.GetContractInfo()
	chk(err)
	output(ret)
}

func TestHuobiFuture_GetTicker(t *testing.T) {
	ret, err := huobi.GetTicker("BTC_CQ")
	chk(err)
	output(ret)
}

func TestHuobiFuture_GetDepth(t *testing.T) {
	ret, err := huobi.GetDepth("BTC_CQ")
	chk(err)
	output(ret)
}

func TestHuobiFuture_GetTrades(t *testing.T) {
	ret, err := huobi.GetTrades("BTC_CQ")
	chk(err)
	output(ret)
}

func TestHuobiFuture_GetAccounts(t *testing.T) {
	ret, err := huobi.GetAccounts()
	chk(err)
	output(ret)
}

func newId() int64 {
	return time.Now().UnixNano()
}

func TestHuobiFuture_PlaceOrder(t *testing.T) {
	req := OrderReq{
		ContractCode: "ETH190927",
		ClientOid: newId(),
		Price: decimal.NewFromFloat(220),
		Volume: 1,
		Direction: DirectionBuy,
		Offset: OffsetOpen,
		LeverRate: 10,
		OrderPriceType: PriceTypeLimit,
	}

	orderId, err := huobi.PlaceOrder(req)
	assert.Nil(t, err)
	output(orderId)
}

func TestHuobiFuture_PlaceOrders(t *testing.T) {
	var reqList []OrderReq = []OrderReq {
		{
			ContractCode: "ETH190927",
			ClientOid: newId(),
			Price: decimal.NewFromFloat(220),
			Volume: 1,
			Direction: DirectionBuy,
			Offset: OffsetOpen,
			LeverRate: 10,
			OrderPriceType: PriceTypeLimit,
		},
	}

	orderIds, errorList, err := huobi.PlaceOrders(reqList)
	assert.Nil(t, err)
	fmt.Println(errorList)
	output(orderIds)
}

func TestHuobiFuture_CancelOrder(t *testing.T) {
	code := "ETH"
	err, errList := huobi.BatchCancelOrders(code, []string{"6145283534", "6145283533"})
	assert.Nil(t, err)
	for _, err := range errList {
		assert.Nil(t, err)
	}
}

func TestHuobiFuture_GetPendingOrders(t *testing.T) {
	code := "ETH"
	orders, err := huobi.QueryPendingOrders(code, 1, 10)
	assert.Nil(t, err)
	output(orders)
}

func TestHuobiFuture_GetHisOrders(t *testing.T) {
	code := "ETH"
	orders, err := huobi.QueryHisOrders(code, 1, 10)
	assert.Nil(t, err)
	output(orders)
}

func TestHuobiFuture_GetOrderById(t *testing.T) {
	order, err := huobi.QueryOrder("ETH", "6145283530", "")
	assert.Nil(t, err)
	output(order)
}

func TestHuobiFuture_GetOrderByClientId(t *testing.T) {
	order, err := huobi.QueryOrder("ETH", "", "1565267630132314000")
	assert.Nil(t, err)
	output(order)
}
