package okexv3spot

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/stephenlyu/GoEx"
	"strings"
	"github.com/pborman/uuid"
	"github.com/shopspring/decimal"
)

var (
	okexV3 *OKExV3Spot
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	type Key struct {
		ApiKey string 	`json:"api-key"`
		SecretKey string `json:"secret-key"`
		Passphrase string `json:"passphrase"`
	}

	bytes, err := ioutil.ReadFile("../key.json")
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)
	okexV3 = NewOKExV3Spot(http.DefaultClient, key.ApiKey, key.SecretKey, key.Passphrase)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestOKExV3_GetInstruments(t *testing.T) {
	instruments, err := okexV3.GetInstruments()
	assert.Nil(t, err)
	output(instruments)
}

func TestOKExV3_GetInstrumentTicker(t *testing.T) {
	ret, err := okexV3.GetInstrumentTicker("ETH-USDT")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetTrades(t *testing.T) {
	ret, err := okexV3.GetTrades("ETH-USDT")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetAccount(t *testing.T) {
	ret, err := okexV3.GetAccount()
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetCurrencyAccount(t *testing.T) {
	ret, err := okexV3.GetCurrencyAccount(goex.USDT)
	assert.Nil(t, err)
	output(ret)
}

func getId() string {
	return strings.Replace(uuid.New(), "-", "", -1)
}

func TestOKExV3Spot_PlaceOrder(t *testing.T) {
	code := "USDT-USDK"
	clientOid := getId()
	println(clientOid)
	orderId, err := okexV3.PlaceOrder(OrderReq{
		ClientOid: clientOid,
		Type: "limit",
		Side: "sell",
		InstrumentId: code,
		OrderType: "0",
		Price: decimal.NewFromFloat(1.0008),
		Size: decimal.NewFromFloat(42),
		MarginTrading: 1,
	})
	assert.Nil(t, err)
	output(orderId)

	order, err := okexV3.GetInstrumentOrder(code, orderId)
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	err := okexV3.CancelOrder("EOS-USDT", "", "cba31249763b45459bbf684bb47f9fb5")
	assert.Nil(t, err)
}

func TestOKExV3_PlaceOrders(t *testing.T) {
	clientOid := getId()
	println(clientOid)
	reqs := []OrderReq {
		{
			ClientOid: clientOid,
			Type: "limit",
			Side: "sell",
			InstrumentId: "btc-usdt",
			OrderType: "0",
			Price: decimal.NewFromFloat(10000),
			Size: decimal.NewFromFloat(1),
			MarginTrading: 1,
		},
	}

	ret, err := okexV3.PlaceOrders(reqs)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_FutureCancelOrders(t *testing.T) {
	err := okexV3.CancelOrders("EOS-USDT", []string {"2500567789935616"}, nil)
	assert.Nil(t, err)
}

func TestOKExV3_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentOrders("EOS-USDT", "6", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentPendingOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentPendingOrders("USDT-USDK", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3.GetInstrumentOrder("USDT-USDK", "3034317952855040")
	assert.Nil(t, err)
	output(order)
}
