package plo

import (
	"testing"
	"fmt"
	"github.com/stephenlyu/GoEx"
	"io/ioutil"
	"encoding/json"
	"github.com/stephenlyu/tds/util"
)

type Key struct {
	ApiKey string 	`json:"api-key"`
	SecretKey string `json:"secret-key"`
}

var (
	API_KEY = ""
	SECRET_KEY = ""
)

func init() {
	bytes, err := ioutil.ReadFile("key.json")
	chk(err)
	var key Key
	err = json.Unmarshal(bytes, &key)
	chk(err)
	API_KEY = key.ApiKey
	SECRET_KEY = key.SecretKey
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func Output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestPloRest_GetTrade(t *testing.T) {
	api := NewPloRest("", "")
	err, ret := api.GetTrade(goex.NewCurrencyPair(goex.EOS, goex.USD))
	chk(err)
	Output(ret)
}

func TestPloRest_GetOrderBook(t *testing.T) {
	api := NewPloRest("", "")
	err, ret := api.GetOrderBook(goex.NewCurrencyPair(goex.EOS, goex.USD))
	chk(err)
	Output(ret)
}

func TestPloRest_GetConfigList(t *testing.T) {
	api := NewPloRest("", "")
	err, ret := api.GetConfigList()
	chk(err)
	Output(ret)
}

func TestPloRest_GetBalances(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.GetBalances()
	chk(err)

	Output(ret)
}

func TestPloRest_PlaceOrders(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	reqOrders := []OrderReq {
		{
			ClientId: fmt.Sprintf("%d", util.NanoTick()),
			PosAction: 0,
			Side: "sell",
			Symbol: "EOSUSD",
			TotalQty: 1,
			Type: "limit",
			Price: 4,
			Leverage: 10,
		},
	}

	err, ret := api.PlaceOrders(reqOrders)
	chk(err)

	Output(ret)
}

func TestPloRest_SelfTrade(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	reqOrders := []OrderReq {
		{
			PosAction: 0,
			Side: "sell",
			Symbol: "EOSUSD",
			TotalQty: 1,
			Type: "limit",
			Price: 2.5172,
			Leverage: 10,
		},
		{
			PosAction: 0,
			Side: "buy",
			Symbol: "EOSUSD",
			TotalQty: 1,
			Type: "limit",
			Price: 2.5172,
			Leverage: 10,
		},
	}

	err := api.SelfTrade(reqOrders)
	chk(err)
}

func TestPloRest_BatchOrders(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.BatchOrders([]string{"58923A6E-A98F-3B6B-A153-CC988DC1A66D"})
	chk(err)

	Output(ret)
}

func TestPloRest_CancelOrders(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.CancelOrders([]string{"9E24C4AE-2D64-F479-55D3-8244295642F7"})
	chk(err)

	Output(ret)
}

func TestPloRest_QueryOrders(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.QueryOrders(goex.EOS_USD, 1)
	chk(err)

	Output(ret)
}

func TestPloRest_QueryPositions(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.QueryPositions(goex.EOS_USD, 2)
	chk(err)

	Output(ret)
}

func TestPloRest_QueryPosRanking(t *testing.T) {
	api := NewPloRest(API_KEY, SECRET_KEY)
	err, ret := api.QueryPosRanking(goex.EOS_USD, "long", 2)
	chk(err)

	Output(ret)
}
