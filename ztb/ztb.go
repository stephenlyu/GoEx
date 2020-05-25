package ztb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

// Order Side
const (
	OrderSell = "1"
	OrderBuy  = "2"
)

// Order type
const (
	OrderTypeLimit  = "1"
	OrderTypeMarket = "2"
)

// Order status
const (
	OrderStatusInit            = 1
	OrderStatusQueue           = 2
	OrderStatusCanceled        = 3
	OrderStatusPartiallyFilled = 4
	OrderStatusFilled          = 5
)

const (
	apiBaseURL           = "https://www.ztb.com"
	commonSymbolsURL     = "/api/v1/exchangeInfo"
	getTickerURL         = "/api/v1/tickers?symbol=%s"
	getMarketDepthURL    = "/api/v1/depth?symbol=%s&size=5"
	getTradesURL         = "/api/v1/trades?symbol=%s&size=1"
	accountURL           = "/api/v1/private/user"
	createOrderURL       = "/api/v1/private/trade/limit"
	batchPlaceURL        = "/trade/api/v1/batchOrder"
	cancelOrderURL       = "/api/v1/private/trade/cancel"
	batchCancelURL       = "/api/v1/private/trade/cancel_batch"
	newOrderURL          = "/api/v1/private/order/pending"
	finishedOrderURL     = "/api/v1/private/order/finished"
	orderInfoURL         = "/api/v1/private/order/pending/detail"
	finishedOrderInfoURL = "/api/v1/private/order/finished/detail"
)

// ErrNotExist is Error not exist
var ErrNotExist = errors.New("NOT EXISTS")

// Ztb is for adapt ztb restful & websocket APIs
type Ztb struct {
	APIKey    string
	SecretKey string
	client    *http.Client

	ws               *goex.WsConn
	createWsLock     sync.Mutex
	wsDepthHandleMap map[string]func(*goex.DepthDecimal)
	wsTradeHandleMap map[string]func(string, []goex.TradeDecimal)
	errorHandle      func(error)
	wsSymbolMap      map[string]string
	depthManagers    map[string]*depthManager
}

// NewZtb is constructor for Ztb object
func NewZtb(client *http.Client, APIKey string, SecretKey string) *Ztb {
	ztb := new(Ztb)
	ztb.APIKey = APIKey
	ztb.SecretKey = SecretKey
	ztb.client = client
	return ztb
}

// GetSymbols is for getting Ztb exchangable symbols
func (ztb *Ztb) GetSymbols() ([]Symbol, error) {
	url := apiBaseURL + commonSymbolsURL
	resp, err := ztb.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data []Symbol

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (ztb *Ztb) transSymbol(symbol string) string {
	return symbol
}

// GetTicker is for getting ticker data of a coin pair
func (ztb *Ztb) GetTicker(symbol string) (*goex.TickerDecimal, error) {
	symbol = ztb.transSymbol(symbol)
	url := fmt.Sprintf(apiBaseURL+getTickerURL, symbol)
	resp, err := ztb.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	// println(string(body))
	var data struct {
		Ticker []struct {
			High   decimal.Decimal
			Vol    decimal.Decimal
			Last   decimal.Decimal
			Low    decimal.Decimal
			Sell   decimal.Decimal
			Buy    decimal.Decimal
			Symbol string
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	ticker := new(goex.TickerDecimal)
	for _, r := range data.Ticker {
		if r.Symbol == symbol {
			ticker.Buy = r.Buy
			ticker.Sell = r.Sell
			ticker.Last = r.Last
			ticker.High = r.High
			ticker.Low = r.Low
			ticker.Vol = r.Vol
			return ticker, nil
		}
	}

	return nil, ErrNotExist
}

// GetDepth is for getting market depth of a coin pair
func (ztb *Ztb) GetDepth(symbol string) (*goex.DepthDecimal, error) {
	inputSymbol := symbol
	symbol = ztb.transSymbol(symbol)
	url := fmt.Sprintf(apiBaseURL+getMarketDepthURL, symbol)
	resp, err := ztb.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Message string
		Code    decimal.Decimal

		Asks [][]decimal.Decimal
		Bids [][]decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := &data

	depth := new(goex.DepthDecimal)
	depth.Pair = goex.NewCurrencyPair2(inputSymbol)

	depth.AskList = make([]goex.DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]goex.DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

// GetTrades is for getting latest trades of a coin pair
func (ztb *Ztb) GetTrades(symbol string) ([]goex.TradeDecimal, error) {
	symbol = ztb.transSymbol(symbol)
	url := fmt.Sprintf(apiBaseURL+getTradesURL, symbol)
	resp, err := ztb.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "code") {
		var r struct {
			Code    decimal.Decimal
			Message string
		}
		err = json.Unmarshal(body, &r)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error code: %s", r.Code.String())
	}

	var data []struct {
		Amount    decimal.Decimal
		Price     decimal.Decimal
		Side      string
		Timestamp decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var trades = make([]goex.TradeDecimal, len(data))

	for i, o := range data {
		t := &trades[i]

		t.Tid = o.Timestamp.IntPart()
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Side
	}

	return trades, nil
}

func (ztb *Ztb) signData(data string) string {
	message := data + "&secret_key=" + ztb.SecretKey
	sign, _ := goex.GetParamMD5Sign(ztb.SecretKey, message)

	return sign
}

func (ztb *Ztb) buildQueryString(param map[string]string) string {
	var parts []string

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var sign string
	for _, k := range keys {
		v := param[k]
		if k == "sign" {
			sign = v
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	parts = append(parts, fmt.Sprintf("sign=%s", url.QueryEscape(sign)))
	return strings.Join(parts, "&")
}

func (ztb *Ztb) sign(param map[string]string) map[string]string {
	param["api_key"] = ztb.APIKey

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, key := range keys {
		value := param[key]
		if value == "" {
			continue
		}
		parts = append(parts, key+"="+value)
	}
	data := strings.Join(parts, "&")

	sign := ztb.signData(data)
	param["sign"] = strings.ToUpper(sign)
	return param
}

func (ztb *Ztb) getAuthHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-SITE-ID":    "1",
	}
}

// GetAccount is for get account balance information
func (ztb *Ztb) GetAccount() ([]goex.SubAccountDecimal, error) {
	params := map[string]string{}
	params = ztb.sign(params)

	url := apiBaseURL + accountURL

	header := ztb.getAuthHeader()
	data := ztb.buildQueryString(params)
	bytes, err := goex.HttpPostForm3(ztb.client, url, data, header)

	if err != nil {
		return nil, err
	}
	var resp struct {
		Message string
		Code    decimal.Decimal
		Result  map[string]interface{}
	}

	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret []goex.SubAccountDecimal
	for coin, o := range resp.Result {
		if reflect.TypeOf(o).Kind() != reflect.Map {
			continue
		}
		currency := strings.ToUpper(coin)
		if currency == "" {
			continue
		}
		m := o.(map[string]interface{})
		available, _ := decimal.NewFromString(m["available"].(string))
		freeze, _ := decimal.NewFromString(m["freeze"].(string))

		ret = append(ret, goex.SubAccountDecimal{
			Currency:        goex.Currency{Symbol: currency},
			AvailableAmount: available,
			FrozenAmount:    freeze,
			Amount:          available.Add(freeze),
		})
	}

	return ret, nil
}

// PlaceOrder is for placing order
func (ztb *Ztb) PlaceOrder(volume decimal.Decimal, side string, symbol string, price decimal.Decimal) (string, error) {
	symbol = ztb.transSymbol(symbol)
	params := map[string]string{
		"market": symbol,
		"side":   side,
		"amount": volume.String(),
		"price":  price.String(),
	}

	params = ztb.sign(params)

	data := ztb.buildQueryString(params)
	// println(data)
	url := apiBaseURL + createOrderURL
	body, err := goex.HttpPostForm3(ztb.client, url, data, ztb.getAuthHeader())

	if err != nil {
		return "", err
	}
	// println(string(body))

	var resp struct {
		Message string
		Code    decimal.Decimal
		Result  struct {
			ID decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if !resp.Code.IsZero() {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Result.ID.String(), nil
}

// CancelOrder is for cancel a single order
func (ztb *Ztb) CancelOrder(symbol, orderID string) (*goex.OrderDecimal, error) {
	symbol = ztb.transSymbol(symbol)
	params := map[string]string{
		"market":   symbol,
		"order_id": orderID,
	}
	params = ztb.sign(params)

	data := ztb.buildQueryString(params)
	url := apiBaseURL + cancelOrderURL
	body, err := goex.HttpPostForm3(ztb.client, url, data, ztb.getAuthHeader())
	if err != nil {
		return nil, err
	}
	// println(string(body))
	var resp struct {
		Messag string
		Code   decimal.Decimal
		Result *OrderInfo
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() == 10 {
		return nil, ErrNotExist
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Result.ToOrderDecimal(symbol), nil
}

// QueryPendingOrders is for query pending orders
func (ztb *Ztb) QueryPendingOrders(symbol string, offset, limit int) ([]goex.OrderDecimal, error) {
	if limit == 0 {
		limit = 20
	}
	param := map[string]string{
		"market": ztb.transSymbol(symbol),
	}
	param["offset"] = strconv.Itoa(offset)
	param["limit"] = strconv.Itoa(limit)
	param = ztb.sign(param)

	url := fmt.Sprintf(apiBaseURL + newOrderURL)

	bytes, err := goex.HttpPostForm3(ztb.client, url, ztb.buildQueryString(param), ztb.getAuthHeader())
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code   decimal.Decimal
		Msg    string
		Result struct {
			Limit   int
			Offset  int
			Records []OrderInfo
		}
	}

	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]goex.OrderDecimal, len(resp.Result.Records))
	for i := range resp.Result.Records {
		ret[i] = *resp.Result.Records[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

// QueryFinishedOrders is for query pending orders
func (ztb *Ztb) QueryFinishedOrders(symbol string, offset, limit int) ([]goex.OrderDecimal, error) {
	if limit == 0 {
		limit = 20
	}
	param := map[string]string{
		"market": ztb.transSymbol(symbol),
	}
	param["offset"] = strconv.Itoa(offset)
	param["limit"] = strconv.Itoa(limit)
	param = ztb.sign(param)

	url := fmt.Sprintf(apiBaseURL + finishedOrderURL)

	bytes, err := goex.HttpPostForm3(ztb.client, url, ztb.buildQueryString(param), ztb.getAuthHeader())
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code   decimal.Decimal
		Msg    string
		Result struct {
			Limit   int
			Offset  int
			Records []OrderInfo
		}
	}

	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]goex.OrderDecimal, len(resp.Result.Records))
	for i := range resp.Result.Records {
		ret[i] = *resp.Result.Records[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

// QueryOrder is for query a incomplete order
func (ztb *Ztb) QueryOrder(symbol, orderID string) (*goex.OrderDecimal, error) {
	param := ztb.sign(map[string]string{
		"market":   ztb.transSymbol(symbol),
		"order_id": orderID,
	})

	url := fmt.Sprintf(apiBaseURL + orderInfoURL)

	var resp struct {
		Code   decimal.Decimal
		Info   string
		Result *OrderInfo
	}

	bytes, err := goex.HttpPostForm3(ztb.client, url, ztb.buildQueryString(param), ztb.getAuthHeader())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Result == nil {
		return nil, ErrNotExist
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Result.ToOrderDecimal(symbol), nil
}

// QueryFinishedOrder is for query a incomplete order
func (ztb *Ztb) QueryFinishedOrder(symbol, orderID string) (*goex.OrderDecimal, error) {
	param := ztb.sign(map[string]string{
		"market":   ztb.transSymbol(symbol),
		"order_id": orderID,
	})

	url := fmt.Sprintf(apiBaseURL + finishedOrderInfoURL)

	var resp struct {
		Code   decimal.Decimal
		Info   string
		Result *OrderInfo
	}

	bytes, err := goex.HttpPostForm3(ztb.client, url, ztb.buildQueryString(param), ztb.getAuthHeader())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Result == nil {
		return nil, ErrNotExist
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Result.ToOrderDecimal(symbol), nil
}
