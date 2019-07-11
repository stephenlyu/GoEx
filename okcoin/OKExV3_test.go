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
	"os"
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

	var configFile = os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "key.json"
	}

	bytes, err := ioutil.ReadFile(configFile)
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

func TestOKExV3_GetTicker(t *testing.T) {
	ret, err := okexV3.GetTicker("EOS-USD-190927")
	chk(err)
	output(ret)
}

func TestOKExV3_GetDepth(t *testing.T) {
	ret, err := okexV3.GetDepth("EOS-USD-190927")
	chk(err)
	output(ret)
}

func TestOKExV3_GetTrades(t *testing.T) {
	ret, err := okexV3.GetTrades("EOS-USD-190927")
	chk(err)
	output(ret)
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
	ret, err := okexV3.GetCurrencyAccount(goex.Currency{Symbol: "BTC"})
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_PlaceFutureOrder(t *testing.T) {
	code := "EOS-USD-190628"
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
	err := okexV3.FutureCancelOrder("EOS-USD-190628", "2706131533665280")
	assert.Nil(t, err)
}

func getId() string {
	return strings.Replace(uuid.New(), "-", "", -1)
}

func TestOKExV3_PlaceFutureOrders(t *testing.T) {
	req := BatchPlaceOrderReq{
		InstrumentId: "EOS-USD-190628",
		Leverage: 10,
		OrdersData: []OrderItem{
			{
				ClientOid: getId(),
				Type: V3_TYPE_BUY_OPEN,
				OrderType: strconv.Itoa(V3_ORDER_TYPE_NORMAL),
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
	err := okexV3.FutureCancelOrders("EOS-USD-190628", nil, []string{"8a43cedd001c4b36843d4c802c176782", "3d99117119dd441db38b44034df1b99c"})
	assert.Nil(t, err)
}

func TestOKExV3_GetInstrumentOrders(t *testing.T) {
	orders, err := okexV3.GetInstrumentOrders("BTC-USD-190628", "7", "", "", "")
	assert.Nil(t, err)
	output(orders)
}

func TestOKExV3_GetInstrumentOrder(t *testing.T) {
	order, err := okexV3.GetInstrumentOrder("EOS-USD-190628", "25117531327024128")
	assert.Nil(t, err)
	output(order)
}

func TestOKExV3_GetLedger(t *testing.T) {
	var ledgers []FutureLedger
	from := ""
	for {
		resp, err := okexV3.GetLedger(goex.EOS, from, "", "100")
		assert.Nil(t, err)
		if len(resp) == 0 {
			break
		} else {
			ledgers = append(ledgers, resp...)
		}
		from = resp[len(resp) - 1].LedgerId
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
	orderId, err := okexV3Swap.PlaceFutureOrder(clientOid, code, "5", "1", 4, V3_SWAP_ORDER_TYPE_NORMAL, 1, 10)
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
	orders, err := okexV3Swap.GetInstrumentOrders("BTC-USD-SWAP", "7", "", "", "")
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
	from := ""
	for {
		println("from:" + from)
		resp, err := okexV3Swap.GetLedger("EOS-USD-SWAP", from, "", "100")
		assert.Nil(t, err)
		if len(resp) == 0 {
			break
		} else {
			ledgers = append(ledgers, resp...)
		}
		from = resp[len(resp) - 1].LedgerId
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
	currency := "USDT"
	//err, resp := okexV3.WalletTransfer(goex.Currency{Symbol: currency}, 0.3464083564, WALLET_ACCOUNT_FUTURE, WALLET_ACCOUNT_WALLET, "", "")
	//assert.Nil(t, err)
	//output(resp)

	err, resp := okexV3.WalletTransfer(goex.Currency{Symbol: currency}, 1700, WALLET_ACCOUNT_WALLET, WALLET_ACCOUNT_SPOT, "", "")
	assert.Nil(t, err)
	output(resp)
}

func TestOKExV3_GetWallet(t *testing.T) {
	currency := "USDT"
	ret, err := okexV3.GetWallet(goex.Currency{Symbol: currency})
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetWithdrawFee(t *testing.T) {
	currency := "BTC"
	ret, err := okexV3.GetWithdrawFee(currency)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_Withdraw(t *testing.T) {
	currency := goex.EOS
	ret, err := okexV3.GetWithdrawFee(currency.Symbol)
	assert.Nil(t, err)
	output(ret)
	fee, _ := ret[0].MinFee.Float64()
	//{"amount":3193.9991152142,"currency":"EOS","destination":4,"fee":0.1,"to_address":"tokenpanda11","trade_pwd":"tokenpanda2018"}

	err, resp := okexV3.Withdraw(currency, 3193.99911521, WithdrawDestinationOuter, "tokenpanda11", "tokenpanda2018", fee)
	assert.Nil(t, err)
	output(resp)
}

func TestOKExV3_GetWalletLedger(t *testing.T) {
	var ledgers []WalletLedger
	from := ""
	currency := goex.EOS
	for {
		fmt.Printf("from: %s\n", from)
		resp, err := okexV3.GetWalletLedger(goex.EOS, from, "", "100", "")
		assert.Nil(t, err)
		ledgers = append(ledgers, resp...)
		if len(resp) < 100 {
			break
		}
		from = resp[len(resp) - 1].LedgerId.String()
		time.Sleep(time.Millisecond * 400)
	}
	bytes, err := json.MarshalIndent(ledgers, "", "  ")
	assert.Nil(t, err)
	ioutil.WriteFile(currency.Symbol + "-wallet-ledgers.json", bytes, 0666)
}

func TestOKExV3_GetDepositHistory(t *testing.T) {
	currency := "EOS"
	ret, err := okexV3.GetDepositHistory(currency)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_GetWithdrawHistory(t *testing.T) {
	currency := "BTC"
	ret, err := okexV3.GetWithdrawHistory(currency)
	assert.Nil(t, err)
	output(ret)
}

func TestOKExV3_SWAP_GetFundingRateHistory(t *testing.T) {
	ret, err := okexV3Swap.GetFundingRateHistory("BTC-USD-SWAP", "", "", "")
	assert.Nil(t, err)
	output(ret)
}
