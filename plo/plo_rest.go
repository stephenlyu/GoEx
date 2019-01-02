package plo

import (
	"net/http"
	"sort"
	"strings"
	"fmt"
	"github.com/stephenlyu/GoEx"
	"github.com/stephenlyu/tds/util"
	"encoding/json"
	"strconv"
	"encoding/base64"
)

const (
	BASE_URL = "https://api.plo.one/"
	TRADE_URL = "/m_api/trade"
	ORDER_BOOK_URL = "/m_api/orderbookL2"
	CONFIG_LIST_URL = "/hapi/Config/ConfList"
	BALANCES_URL = "/hapi/BatchOperation/balances"
	PLACE_ORDER_URL = "/hapi/BatchOperation/batchPosExec"
	CANCEL_ORDER_URL = "/hapi/BatchOperation/batchOrderCancel"
	BATCH_ORDER_URL = "/hapi/BatchOperation/batchOrderCondquery"
	ORDERS_URL = "/hapi/BatchOperation/orders"
	POSITIONS_URL = "/hapi/BatchOperation/positions"
)

type PloRest struct {
	apiKey string
	apiSecretKey string
	client *http.Client
}

func NewPloRest(apiKey string, apiSecretKey string) *PloRest {
	return &PloRest{
		apiKey: apiKey,
		apiSecretKey: apiSecretKey,

		client: http.DefaultClient,
	}
}

func (bitmex *PloRest) map2Query(params map[string]string) string {
	keys := make([]string, len(params))
	var i int
	for k := range params {
		keys[i] = k
		i++
	}
	sort.SliceStable(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	parts := make([]string, len(params))
	for i, k := range keys {
		v := params[k]
		parts[i] = k + "=" + v
		i++
	}
	return strings.Join(parts, "&")
}

func (this *PloRest) GetTrade(pair goex.CurrencyPair) (error, interface{}) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data interface{}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+TRADE_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
}

func (this *PloRest) GetOrderBook(pair goex.CurrencyPair) (error, interface{}) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data interface{}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+ORDER_BOOK_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
}

func (this *PloRest) GetConfigList() (error, interface{}) {
	var data interface{}
	err := goex.HttpGet4(this.client, BASE_URL+CONFIG_LIST_URL, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
}

func (this *PloRest) GetBalances() (error, *goex.FutureAccount) {
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, "")

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+BALANCES_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []struct {
			AccountId string 		`json:"accountId"`
			Address string 			`json:"address"`
			Balance string 			`json:"balance"`
			Currency string 		`json:"currency"`
			OrderMargin string 		`json:"orderMargin"`
			PositionMargin string 	`json:"positionMargin"`
		}	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error > 0 {
		return fmt.Errorf("error: %d", resp.Error), nil
	}

	ret := new(goex.FutureAccount)
	ret.FutureSubAccounts = make(map[goex.Currency]goex.FutureSubAccount)

	for _, r := range resp.Data {
		currency := goex.Currency{Symbol: r.Currency}
		balance, _ := strconv.ParseFloat(r.Balance, 64)
		ret.FutureSubAccounts[currency] = goex.FutureSubAccount{
			Currency: currency,
			AccountRights: balance,
		}
	}
	return nil, ret
}

type OrderReq struct {
	PosAction int 			`json:"posAction"`
	Side string 			`json:"side"`
	Symbol string 			`json:"symbol"`
	TotalQty int64 			`json:"totalQty"`
	Price float64 			`json:"price"`
	Type string 			`json:"type"`
	Leverage int 			`json:"leverage"`
	PosId string 			`json:"posId"`
}

type OrderResp struct {
	Error int 				`json:"err"`
	Msg string 				`json:"msg"`
	Data *json.RawMessage	`json:"data"`
	Order *struct {
		AccountId string 		`json:"accountId"`
		Symbol string 			`json:"symbol"`
		Type string 			`json:"type"`
		Side string 			`json:"side"`
		ClientId string 		`json:"clientId"`
		Price string 			`json:"price"`
		PosAction int 			`json:"posAction"`
		CurrentQty int64 		`json:"currentQty"`
		TotalQty int64 			`json:"totalQty"`
		Status int 				`json:"status"`
		Timestamp int64 		`json:"timestamp"`
		PosMargin string 		`json:"posMargin"`
		OpenFee string 			`json:"openFee"`
		CloseFee string 		`json:"closeFee"`
		PosId string 			`json:"posId"`
		OrderId string 			`json:"orderId"`
	}
}

func (this *PloRest) PlaceOrders(reqOrders []OrderReq) (error, []OrderResp) {
	ts := util.Tick()
	bytes, _ := json.Marshal(reqOrders)
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+PLACE_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	fmt.Println(string(bytes))

	var resp struct {
		Data []OrderResp	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	for i := range resp.Data {
		item := &resp.Data[i]
		if item.Error == 0 {
			json.Unmarshal([]byte(*item.Data), &item.Order)
			item.Data = nil
		}
	}

	return nil, resp.Data
}

func (this *PloRest) CancelOrders(orderIds []string) (error, []error) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(map[string]interface{}{"data": data})
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+CANCEL_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []struct {
			Error int 		`json:"err"`
			Msg string 		`json:"msg"`
		}	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	errors := make([]error, len(resp.Data))
	for i := range resp.Data {
		item := &resp.Data[i]
		if item.Error > 0 {
			errors[i] = fmt.Errorf("error: %d msg: %s", item.Error, item.Msg)
		}
	}

	return nil, errors
}

type Order struct {
	AccountId string 		`json:"accountId"`
	OwnerType int 			`json:"ownerType"`
	Symbol string 			`json:"symbol"`
	Type string 			`json:"type"`
	Side string 			`json:"side"`
	ClientId string 		`json:"clientId"`
	Price string 			`json:"price"`
	PosAction string 		`json:"posAction"`
	CurrentQty string 		`json:"currentQty"`
	TotalQty string 		`json:"totalQty"`
	Status int 				`json:"status"`
	Timestamp int64 		`json:"timestamp"`
	PosMargin string 		`json:"posMargin"`
	OpenFee string 			`json:"openFee"`
	CloseFee string 		`json:"closeFee"`
	PosId string 			`json:"posId"`
	OrderId string 			`json:"orderId"`
}

func (this *PloRest) BatchOrders(orderIds []string) (error, []Order) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(map[string]interface{}{"data": data})
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+BATCH_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []Order 	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}

func (this *PloRest) QueryOrders(pair goex.CurrencyPair, status int) (error, []Order) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"status": status,
	}

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	print(message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+ORDERS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []Order 	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}

func (this *PloRest) QueryPositions(pair goex.CurrencyPair, status int) (error, interface{}) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"status": status,
	}

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+POSITIONS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	fmt.Println(string(bytes))

	var resp struct {
		Data []interface{} 	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}
