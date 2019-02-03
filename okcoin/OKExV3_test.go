package okcoin

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/stephenlyu/GoEx"
	"github.com/pborman/uuid"
	"strings"
)

var (
	okexV3 *OKExV3
	okexV3Swap *OKExV3_SWAP
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

	okexV3Swap = NewOKExV3_SWAP(http.DefaultClient, key.ApiKey, key.SecretKey, key.Passphrase)
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
	ret, err := okexV3.PlaceFutureOrder(getId(), "EOS-USD-190329", "2", "1", 1, 0, 10)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_FutureCancelOrder(t *testing.T) {
	err := okexV3.FutureCancelOrder("EOS-USD-190329", "2229360331400192")
	assert.Nil(t, err)
}

func getId() string {
	return strings.Replace(uuid.New(), "-", "", -1)
}

func TestOKExV3_PlaceFutureOrders(t *testing.T) {
	req := BatchPlaceOrderReq{
		InstrumentId: "EOS-USD-190329",
		Leverage: 10,
		OrdersData: []OrderItem{
			{
				ClientOid: getId(),
				Type: "1",
				Price: "2",
				Size: "1",
				MatchPrice: "0",
			},
		},
	}

	ret, err := okexV3.PlaceFutureOrders(req)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_FutureCancelOrders(t *testing.T) {
	err := okexV3.FutureCancelOrders("EOS-USD-190329", []string{"2228533294214144"})
	assert.Nil(t, err)
}

func TestOKExV3_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentOrders("EOS-USD-181228", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3.GetInstrumentOrder("EOS-USD-190329", "2228965275147265")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_GetLedger(t *testing.T) {
	resp, err := okexV3.GetLedger(goex.EOS, "", "", "")
	assert.Nil(t, err)
	output(resp)
}



func TestOKExV3Swap_GetInstruments(t *testing.T) {
	instruments, err := okexV3Swap.GetInstruments()
	assert.Nil(t, err)
	output(instruments)
}

func TestOKExV3Swap_GetPosition(t *testing.T) {
	ret, err := okexV3Swap.GetPosition()
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_GetInstrumentPosition(t *testing.T) {
	ret, err := okexV3Swap.GetInstrumentPosition("EOS-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_GetInstrumentTicker(t *testing.T) {
	ret, err := okexV3Swap.GetInstrumentTicker("ETH-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_GetInstrumentIndex(t *testing.T) {
	ret, err := okexV3Swap.GetInstrumentIndex("ETH-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_GetAccount(t *testing.T) {
	ret, err := okexV3Swap.GetAccount()
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_SWAP_GetInstrumentAccount(t *testing.T) {
	ret, err := okexV3Swap.GetInstrumentAccount("ETH-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_PlaceFutureOrder(t *testing.T) {
	ret, err := okexV3Swap.PlaceFutureOrder(getId(), "EOS-USD-SWAP", "1.9", "1", 1, 0, 10)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_FutureCancelOrder(t *testing.T) {
	err := okexV3Swap.FutureCancelOrder("EOS-USD-SWAP", "6a-4-43264dfdb-0")
	assert.Nil(t, err)
}

func TestOKExV3Swap_PlaceFutureOrders(t *testing.T) {
	req := V3SwapBatchPlaceOrderReq{
		InstrumentId: "EOS-USD-SWAP",
		OrdersData: []V3SwapOrderItem{
			{
				ClientOid: getId(),
				Type: "1",
				Price: "1.9",
				Size: "1",
				MatchPrice: "0",
			},
		},
	}

	ret, err := okexV3Swap.PlaceFutureOrders(req)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_FutureCancelOrders(t *testing.T) {
	err := okexV3Swap.FutureCancelOrders("EOS-USD-SWAP", []string{"6a-9-432f23bf5-0"})
	assert.Nil(t, err)
}

func TestOKExV3Swap_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3Swap.GetInstrumentOrders("EOS-USD-SWAP", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3Swap_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3Swap.GetInstrumentOrder("EOS-USD-SWAP", "6a-4-432634603-0")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3Swap_GetLedger(t *testing.T) {
	resp, err := okexV3Swap.GetLedger("EOS-USD-SWAP", "", "", "")
	assert.Nil(t, err)
	output(resp)
}
