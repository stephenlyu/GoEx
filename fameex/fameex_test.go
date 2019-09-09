package fameex

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
	"crypto/tls"
	"github.com/nntaoli-project/GoEx"
)

var fameex *Fameex

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		ApiKey string 	`json:"api-key"`
		SecretKey string `json:"secret-key"`
		UserId string 	`json:"user-id"`
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
	fameex = NewFameex(&http.Client{Transport:&http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}},
		key.ApiKey, key.SecretKey, key.UserId)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestFameex_GetSymbols(t *testing.T) {
	ret, err := fameex.GetSymbols()
	chk(err)
	output(ret)
}

func TestFameex_GetTicker(t *testing.T) {
	ret, err := fameex.GetTicker("BTC_USDT")
	chk(err)
	output(ret)
}

func TestFameex_GetDepth(t *testing.T) {
	ret, err := fameex.GetDepth("BTC_USDT")
	chk(err)
	output(ret)
}

func TestFameex_GetTrades(t *testing.T) {
	ret, err := fameex.GetTrades("BTC_USDT")
	chk(err)
	output(ret)
}

func TestFameex_GetAccounts(t *testing.T) {
	ret, err := fameex.GetAccounts()
	chk(err)
	output(ret)
}

func TestFameex_PlaceOrder(t *testing.T) {
	code := "BTC_USDT"
	orderId, err := fameex.PlaceOrder(code, SIDE_BUY, decimal.NewFromFloat(98), decimal.NewFromFloat(0.01))
	assert.Nil(t, err)
	output(orderId)
}

func TestFameex_PlaceOrders(t *testing.T) {
	code := "BTC_USDT"

	var reqList []OrderReq = []OrderReq {
		{
			Side: SIDE_BUY,
			Price: decimal.NewFromFloat(98.5),
			Amount: decimal.NewFromFloat(0.01),
		},
		{
			Side: SIDE_SELL,
			Price: decimal.NewFromFloat(98.5),
			Amount: decimal.NewFromFloat(0.01),
		},
	}

	orderIds, errorList, err := fameex.PlaceOrders(code, reqList)
	assert.Nil(t, err)
	fmt.Println(errorList)
	output(orderIds)
}

func TestFameex_CancelOrder(t *testing.T) {
	code := "OMG_USDT"
	err := fameex.CancelOrder(code, "11390673392530882560")
	assert.Nil(t, err)
}

func TestFameex_BatchCancelOrders(t *testing.T) {
	code := "BTC_USDT"
	err, errorList := fameex.BatchCancelOrders(code, []string{"11390673392577019904", "10390673055099125760"})
	assert.Nil(t, err)
	fmt.Println(errorList)
}

func TestFameex_GetPendingOrders(t *testing.T) {
	code := "BTC_USDT"
	orders, err := fameex.QueryPendingOrders(code, 1, 10)
	assert.Nil(t, err)
	output(orders)
}

func TestFameex_Freq(t *testing.T) {
	code := "sht_usdt"
	for i := 0; i < 100; i++ {
		_, err := fameex.QueryPendingOrders(code, 0, 100)
		fmt.Println(err)
		if err != nil {
			time.Sleep(time.Second)
		}
	}
}

func TestFameex_GetOrder(t *testing.T) {
	code := "BTC_USDT"
	order, err := fameex.QueryOrder(code, "11390873839006908416")
	assert.Nil(t, err)
	output(order)
}

func Test_CancelAll(t *testing.T) {
	code := "BTC_USDT"
	orders, err := fameex.QueryPendingOrders(code, 0, 100)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		err = fameex.CancelOrder(code, o.OrderID2)
		fmt.Println(err)
	}
}

func TestSign(t *testing.T) {
	s := `GET
testapi.fameex.com
/v1/common/symbols
AccessKeyId=ef20232e-858f-8668-2f2f-b680a4d00c83&SignatureMethod=HmacSHA256&SignatureVersion=v0.6`
	sign, _ := goex.GetParamHmacSHA256Sign(fameex.SecretKey, s)
	println(sign)

}