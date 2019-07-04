package ceohk

import (
	"testing"
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)


var ceo *CEOHK

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
	ceo = NewCEOHK(key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestCEOHK_GetAllTickers(t *testing.T) {
	tickers, err := ceo.GetAllTickers()
	chk(err)
	output(tickers)
}

func TestCEOHK_GetTicker(t *testing.T) {
	ticker, err := ceo.GetTicker("sht_qc")
	chk(err)
	output(ticker)
}

func TestCEOHK_GetAccount(t *testing.T) {
	ret, err := ceo.GetAccount()
	chk(err)
	output(ret)
}

func TestCEOHK_PlaceOrder(t *testing.T) {
	code := "SHT_QC"
	orderId, err := ceo.PlaceOrder(decimal.NewFromFloat32(20), TRADE_TYPE_BUY, code, decimal.NewFromFloat(0.20))
	assert.Nil(t, err)
	output(orderId)

	//order, err := ceo.QueryOrder(code, orderId)
	//assert.Nil(t, err)
	//output(order)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	code := "SHT_QC"
	err := ceo.CancelOrder(code, "41219633")
	assert.Nil(t, err)
}

func TestOKExV3_QueryOrders(t *testing.T) {
	code := "SHT_QC"
	orders, err := ceo.QueryOrders(code, 0, 0, 0, "0")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetOrder(t *testing.T) {
	code := "SHT_QC"
	order, err := ceo.QueryOrder(code, "41219442")
	assert.Nil(t, err)
	output(order)
}

func TestZBG_CancelAll(t *testing.T) {
	code := "sht_qc"
	orders, err := ceo.QueryOrders(code, 0, 100, 0, "0")
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		err = ceo.CancelOrder(code, o.OrderID2)
		fmt.Println(err)
	}
}