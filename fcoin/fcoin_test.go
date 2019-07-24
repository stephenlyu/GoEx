package fcoin

import (
	"github.com/stephenlyu/GoEx"
	"net/http"
	"testing"
	"io/ioutil"
	"encoding/json"
	"os"
	"fmt"
	"github.com/stretchr/testify/assert"
)

var ft *FCoin

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
	ft = NewFCoin(http.DefaultClient, key.ApiKey, key.SecretKey)
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestFCoin_GetTicker(t *testing.T) {
	t.Log(ft.GetTicker(goex.NewCurrencyPair2("BTC_USDT")))
}

func TestFCoin_GetDepth(t *testing.T) {
	dep, _ := ft.GetDepth(1, goex.BTC_USDT)
	t.Log(dep.AskList)
	t.Log(dep.BidList)
}

func TestFCoin_GetTrade(t *testing.T) {
	dep, _ := ft.GetTrades(goex.BTC_USDT, 0)
	t.Log(dep)
}

func TestFCoin_GetAccount(t *testing.T) {
	acc, _ := ft.GetAccount()
	output(acc)
}

func TestFCoin_LimitBuy(t *testing.T) {
	t.Log(ft.LimitBuy("40", "0.043", goex.NewCurrencyPair2("SHT_USDT")))
}

func TestFCoin_LimitSell(t *testing.T) {
	t.Log(ft.LimitSell("10", "0.06", goex.NewCurrencyPair2("SHT_USDT")))
}

func TestFCoin_GetOneOrder(t *testing.T) {
	ret, err := ft.GetOneOrder("uCtK8XvJmhIE7p46f2eKdITwMf7t-tYXNB4nhX3ruiiy8_qApf9rVPWNEI_oNXE0WOrpHgZ14Nz29mrjmOKuDA==", goex.NewCurrencyPair2("SHT_USDT"))
	chk(err)
	output(ret)
}

func TestFCoin_CancelOrder(t *testing.T) {
	err := ft.CancelOrder("Oi0QSBsm1k8liCTE1OA6NMwuTCM_QgECsd86Pl-Va-2KCZsrJShYjfPGr27KmjP9_sYDCxeTkdK9sqC-XusUiA==", goex.NewCurrencyPair2("SHT_USDT"))
	chk(err)
}

func TestFCoin_GetUnfinishOrders(t *testing.T) {
	ret, err := ft.GetUnfinishedOrders(goex.NewCurrencyPair2("SHT_USDT"), 0, 0, 0)
	chk(err)
	output(ret)
	fmt.Println(len(ret))
}

func TestFCoin_GetFinishedOrders(t *testing.T) {
	ret, err := ft.GetFinishedOrders(goex.NewCurrencyPair2("SHT_USDT"), 0, 0, 0)
	chk(err)
	output(ret)
}

func TestFCoin_AssetTransfer(t *testing.T) {
	ft.AssetTransfer(goex.NewCurrency("FT", ""), "0.000945618753747253", "assets", "spot")
}

func TestFCoin_GetAssets(t *testing.T) {
	acc, _ := ft.GetAssets()
	t.Log(acc)
}

func TestFCoin_CancelAll(t *testing.T) {
	pair := goex.NewCurrencyPair2("SHT_USDT")
	orders, err := ft.GetUnfinishedOrders(pair, 0, 0, 100)
	assert.Nil(t, err)
	output(orders)

	for _, o := range orders {
		ft.CancelOrder(o.OrderID2, pair)
	}
}
