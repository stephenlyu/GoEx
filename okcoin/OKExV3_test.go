package okcoin

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/stephenlyu/GoEx"
)

var (
	okexV3 *OKExV3
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

	bytes, err := ioutil.ReadFile("key.json")
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)

	okexV3 = NewOKExV3(http.DefaultClient, key.ApiKey, key.SecretKey, key.Passphrase)
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

func TestOKExV3_GetPosition(t *testing.T) {
	ret, err := okexV3.GetPosition()
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetInstrumentPosition(t *testing.T) {
	ret, err := okexV3.GetInstrumentPosition("ETH-USD-181228")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetInstrumentTicker(t *testing.T) {
	ret, err := okexV3.GetInstrumentTicker("ETH-USD-181228")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetInstrumentIndex(t *testing.T) {
	ret, err := okexV3.GetInstrumentIndex("ETH-USD-181228")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetAccount(t *testing.T) {
	ret, err := okexV3.GetAccount()
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_PlaceFutureOrder(t *testing.T) {
	ret, err := okexV3.PlaceFutureOrder("", "EOS-USD-181228", "2", "1", 1, 0, 10)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	err := okexV3.FutureCancelOrder("EOS-USD-181228", "1922187005039616")
	assert.Nil(t, err)
}

func TestOKExV3_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentOrders("EOS-USD-181228", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3.GetInstrumentOrder("EOS-USD-181228", "1922187005039616")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_GetLedger(t *testing.T) {
	resp, err := okexV3.GetLedger(goex.EOS, "", "", "")
	assert.Nil(t, err)
	output(resp)
}
