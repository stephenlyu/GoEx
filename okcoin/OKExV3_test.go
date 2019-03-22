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
	"time"
	"strconv"
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
	ret, err := okexV3.GetInstrumentPosition("ETH-USD-190628")
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

func TestOKExV3_GetCurrencyAccount(t *testing.T) {
	ret, err := okexV3.GetCurrencyAccount(goex.Currency{Symbol: "EOS"})
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_PlaceFutureOrder(t *testing.T) {
	code := "EOS-USD-190329"
	clientOid := getId()
	println(clientOid)
	orderId, err := okexV3.PlaceFutureOrder(clientOid, code, "3.6", "1", 1, V3_ORDER_TYPE_POST_ONLY, 0, 10)
	assert.Nil(t, err)
	output(orderId)

	order, err := okexV3.GetInstrumentOrder(code, orderId)
	assert.Nil(t, err)
	output(order)
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
	err := okexV3.FutureCancelOrders("EOS-USD-190329", []string{"2465877328667648"})
	assert.Nil(t, err)
}

func TestOKExV3_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentOrders("EOS-USD-190329", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3.GetInstrumentOrder("EOS-USD-190628", "2517531327024128")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_GetLedger(t *testing.T) {
	var ledgers []FutureLedger
	page := 1
	for {
		resp, err := okexV3.GetLedger(goex.EOS, strconv.Itoa(page), "", "100")
		assert.Nil(t, err)
		if len(resp) == 0 {
			break
		} else {
			ledgers = append(ledgers, resp...)
		}
		page++
		time.Sleep(time.Millisecond * 500)
	}
	bytes, err := json.MarshalIndent(ledgers, "", "  ")
	assert.Nil(t, err)
	ioutil.WriteFile("eos-ledgers.json", bytes, 0666)
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
	currency := "ETH"
	ret, err := okexV3Swap.GetInstrumentPosition(currency + "-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
	ret, err = okexV3.GetInstrumentPosition(currency + "-USD-190329")
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
	ret, err := okexV3Swap.GetInstrumentAccount("EOS-USD-SWAP")
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3Swap_PlaceFutureOrder(t *testing.T) {
	code := "EOS-USD-SWAP"
	clientOid := getId()
	println(clientOid)
	orderId, err := okexV3Swap.PlaceFutureOrder(clientOid, code, "3.45", "1", 1, V3_SWAP_ORDER_TYPE_POST_ONLY, 0, 10)
	assert.Nil(t, err)
	output(orderId)

	for {
		order, err := okexV3Swap.GetInstrumentOrder(code, orderId)
		assert.Nil(t, err)
		if order == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		output(order)
		break
	}
}

func TestOKExV3Swap_FutureCancelOrder(t *testing.T) {
	err := okexV3Swap.FutureCancelOrder("EOS-USD-SWAP", "6a-4-51419f705-0")
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
	orders, err := okexV3Swap.GetInstrumentOrders("ETH-USD-SWAP", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3Swap_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3Swap.GetInstrumentOrder("EOS-USD-SWAP", "65a0951ebaa341bfa892062d44a5d113")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3Swap_GetLedger(t *testing.T) {
	var ledgers []V3FutureLedger
	page := 1
	for {
		fmt.Printf("page %d..\n", page)
		resp, err := okexV3Swap.GetLedger("EOS-USD-SWAP", strconv.Itoa(page), "", "100")
		assert.Nil(t, err)
		if len(resp) == 0 {
			break
		} else {
			ledgers = append(ledgers, resp...)
		}
		page++
		time.Sleep(time.Millisecond * 400)
	}
	bytes, err := json.MarshalIndent(ledgers, "", "  ")
	assert.Nil(t, err)
	ioutil.WriteFile("eos-swap-ledgers.json", bytes, 0666)
}

func TestQueryAccount(t *testing.T) {
	currency := "EOS"
	ret, err := okexV3.GetCurrencyAccount(goex.Currency{Symbol: currency})
	assert.Nil(t, err)
	output(ret)

	retSwap, err := okexV3Swap.GetInstrumentAccount(currency + "-USD-SWAP")
	assert.Nil(t, err)
	output(retSwap)

	fmt.Println(ret.AccountRights + retSwap.AccountRights)
}

func TestOKExV3_WalletTransfer(t *testing.T) {
	currency := "EOS"
	err, resp := okexV3.WalletTransfer(goex.Currency{Symbol: currency}, 200, WALLET_ACCOUNT_FUTURE, WALLET_ACCOUNT_WALLET, "", "")
	assert.Nil(t, err)
	output(resp)

	//err, resp = okexV3.WalletTransfer(goex.Currency{Symbol: currency}, 10, WALLET_ACCOUNT_WALLET, WALLET_ACCOUNT_SWAP, "", "")
	//assert.Nil(t, err)
	//output(resp)
}

func TestOKExV3_GetWallet(t *testing.T) {
	currency := "EOS"
	ret, err := okexV3.GetWallet(goex.Currency{Symbol: currency})
	assert.Nil(t, err)
	output(ret)
}


func TestOKExV3_GetWalletLedger(t *testing.T) {
	var ledgers []WalletLedger
	page := 1
	for {
		fmt.Printf("page %d..\n", page)
		resp, err := okexV3.GetWalletLedger(goex.ETH, strconv.Itoa(page), "", "100", "")
		assert.Nil(t, err)
		ledgers = append(ledgers, resp...)
		fmt.Println(len(resp))
		if len(resp) < 100 {
			break
		}
		page++
		time.Sleep(time.Millisecond * 400)
		break
	}
	bytes, err := json.MarshalIndent(ledgers, "", "  ")
	assert.Nil(t, err)
	ioutil.WriteFile("eos-wallet-ledgers.json", bytes, 0666)
}
