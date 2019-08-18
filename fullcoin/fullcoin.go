package fullcoin

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
	"sort"
	"net/url"
	"errors"
	"strconv"
	"sync"
)

const (
	SIDE_BUY = "buy"
	SIDE_SELL = "sell"

	TYPE_LIMIT = "limit"
	TYPE_MARKET = "market"
)

const (
	HOST = "openapi.fullcoin.com"
	API_BASE_URL = "https://" + HOST
	SYMBOL = "/open/api/common/symbols"
	TICKER = "/open/api/get_ticker?symbol=%s"
	DEPTH = "/open/api/market_dept?symbol=%s&type=step0"
	TRADE = "/open/api/get_trades?symbol=%s"
	ACCOUNTS = "/open/api/user/account"
	PLACE_ORDER = "/open/api/create_order"
	CANCEL_ORDER = "/open/api/cancel_order"
	CANCEL_ALL = "/open/api/cancel_order_all"
	OPEN_ORDERS = "/open/api/v2/new_order"
	QUERY_ORDER = "/open/api/order_info"
)

var (
	ErrOrderStateError = errors.New("order-orderstate-error")
	ErrNotExist = errors.New("not exist")
)

type FullCoin struct {
	ApiKey string
	SecretKey string
	client *http.Client

	accountId int64
	symbolNameMap map[string]string

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	wsAccountHandleMap  map[string]func(*SubAccountDecimal)
	wsOrderHandleMap  map[string]func([]OrderDecimal)
	wsSymbolMap map[string]string
	errorHandle      func(error)
}

func NewFullCoin(ApiKey string, SecretKey string) *FullCoin {
	this := new(FullCoin)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *FullCoin) getPairByName(name string) string {
	name = strings.ToUpper(name)
	c, ok := this.symbolNameMap[name]
	if ok {
		return c
	}

	var err error
	var l []Symbol
	for i := 0; i < 5; i++ {
		l, err = this.GetSymbols()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}

	for _, o := range l {
		this.symbolNameMap[strings.ToUpper(this.transSymbol(o.Symbol))] = fmt.Sprintf("%s_%s", o.BaseCurrency, o.QuoteCurrency)
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (this *FullCoin) GetSymbols() ([]Symbol, error) {
	url := API_BASE_URL + SYMBOL
	resp, err := this.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Code decimal.Decimal
		Data []Symbol
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error_code: %s", data.Code)
	}

	for i := range data.Data {
		s := &data.Data[i]
		s.BaseCurrency = strings.ToUpper(s.BaseCurrency)
		s.QuoteCurrency = strings.ToUpper(s.QuoteCurrency)
		s.Symbol = fmt.Sprintf("%s_%s", s.BaseCurrency, s.QuoteCurrency)
	}

	return data.Data, nil
}

func (this *FullCoin) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *FullCoin) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := API_BASE_URL + TICKER
	resp, err := this.client.Get(fmt.Sprintf(url, symbol))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data struct {
		Code decimal.Decimal
		Data struct {
				 High decimal.Decimal
				 Vol  decimal.Decimal
				 Last decimal.Decimal
				 Low  decimal.Decimal
				 Buy  decimal.Decimal
				 Sell decimal.Decimal
				 Time int64
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error_code: %s", data.Code)
	}
	r := data.Data

	ticker := new(TickerDecimal)
	ticker.Date = uint64(r.Time)
	ticker.Buy = r.Buy
	ticker.Sell = r.Sell
	ticker.Last = r.Last
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol

	return ticker, nil
}

func (this *FullCoin) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)

	url := fmt.Sprintf(API_BASE_URL + DEPTH, symbol)
	resp, err := this.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Code decimal.Decimal
		Data struct {
				 Tick struct {
						  Asks [][]decimal.Decimal
						  Bids [][]decimal.Decimal
					  }
			 }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error_code: %s", data.Code)
	}

	r := data.Data.Tick

	depth := new(DepthDecimal)
	depth.Pair = NewCurrencyPair2(inputSymbol)

	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

func (this *FullCoin) GetTrades(symbol string) ([]TradeDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + TRADE, symbol)
	resp, err := this.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Code decimal.Decimal
		Data []struct {
			Amount decimal.Decimal
			Price  decimal.Decimal
			Id     decimal.Decimal
			Type   string
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error_code: %s", data.Code)
	}

	var trades = make([]TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Type
		t.Tid = o.Id.IntPart()
	}

	return trades, nil
}

func (this *FullCoin) signData(data string) string {
	message := data + this.SecretKey
	println(message)
	sign, _ := GetParamMD5SignBase64(this.SecretKey, message)

	return sign
}

func (this *FullCoin) buildQueryString(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k + "=" + url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

func (this *FullCoin) sign(param map[string]string) string {
	now := time.Now()
	param["aki_key"] = this.ApiKey
	param["time"] = strconv.FormatInt(now.UnixNano()/1000000, 10)
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + param[k])
	}
	data := strings.Join(parts, "")

	println(data)
	sign := this.signData(data)
	param["sign"] = sign

	return this.buildQueryString(param)
}

func (this *FullCoin) GetAccounts() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	queryString := this.sign(params)

	url := API_BASE_URL + ACCOUNTS + "?" + queryString
	println(url)
	var resp struct {
		Code decimal.Decimal
		Data struct {
				 TotalAsset decimal.Decimal 		`json:"total_assets"`
				 CoinList []struct {
					 Coin string
					 Normal decimal.Decimal
					 Locked decimal.Decimal
				 }
			 }
	}

	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded;charset=utf-8",
	}
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	var m = make(map[string]*SubAccountDecimal)
	for _, o := range resp.Data.CoinList {
		currency := strings.ToUpper(o.Coin)
		if currency == "" {
			continue
		}
		m[currency] = &SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Normal,
			FrozenAmount: o.Locked,
			Amount: o.Normal.Add(o.Locked),
		}
	}

	var ret []SubAccountDecimal
	for _, o := range m {
		ret = append(ret, *o)
	}

	return ret, nil
}

func (this *FullCoin) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)

	var orderType string
	if side == SIDE_BUY {
		if _type == TYPE_LIMIT {
			orderType = "buy-limit"
		} else {
			orderType = "buy-market"
		}
	} else {
		if _type == TYPE_LIMIT {
			orderType = "sell-limit"
		} else {
			orderType = "sell-market"
		}
	}

	params := map[string]string {
		"account-id": strconv.FormatInt(this.accountId, 10),
		"symbol": symbol,
		"type": orderType,
		"amount": volume.String(),
		"price": price.String(),
		"source": "api",
	}

	queryString := this.sign(map[string]string{})

	data, _ := json.Marshal(params)

	url := API_BASE_URL + PLACE_ORDER + "?" + queryString
	body, err := HttpPostJson(this.client, url, string(data), map[string]string{})

	if err != nil {
		return "", err
	}
	var resp struct {
		Status string
		ErrCode string	`json:"err-code"`
		Data string
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Status != "ok" {
		return "", fmt.Errorf("bad status: %s", resp.ErrCode)
	}

	return resp.Data, nil
}

func (this *FullCoin) CancelOrder(orderId string) error {
	var path string
	//path := fmt.Sprintf(CANCEL_ORDER, orderId)
	queryString := this.sign(map[string]string{})

	url := API_BASE_URL + path + "?" + queryString
	body, err := HttpPostJson(this.client, url, "", map[string]string{})

	if err != nil {
		return err
	}

	var resp struct {
		Status string
		ErrCode string		`json:"err-code"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "ok" {
		if resp.ErrCode == "order-orderstate-error" {
			return ErrOrderStateError
		}

		return fmt.Errorf("bad status: %s", resp.ErrCode)
	}

	return nil
}

func (this *FullCoin) CancelOrders(orderIds []string) (error, []error) {
	params := map[string]interface{} {
		"order-ids": orderIds,
	}

	queryString := this.sign(map[string]string{})

	data, _ := json.Marshal(params)

	url := API_BASE_URL + CANCEL_ALL + "?" + queryString
	body, err := HttpPostJson(this.client, url, string(data), map[string]string{})

	var errorList = make([]error, len(orderIds))

	if err != nil {
		return err, errorList
	}

	var resp struct {
		Status string
		ErrCode string		`json:"err-code"`
		Data struct {
			Success []string
			Failed []struct {
				ErrMsg string 		`json:"err-msg"`
				OrderId string 		`json:"order-id"`
				ErrorCode string 	`json:"err-code"`
			}
			 }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err, errorList
	}

	if resp.Status != "ok" {
		return fmt.Errorf("bad status: %s", resp.ErrCode), errorList
	}

	m := make(map[string]error)
	for _, orderId := range resp.Data.Success {
		m[orderId] = err
	}
	for _, r := range resp.Data.Failed {
		m[r.OrderId] = errors.New(r.ErrorCode)
	}

	for i, orderId := range orderIds {
		errorList[i] = m[orderId]
	}

	return nil, errorList
}

func (this *FullCoin) QueryPendingOrders(symbol string, size int) ([]OrderDecimal, error) {
	if size == 0 {
		size = 10
	}
	param := map[string]string {
		"account-id": strconv.FormatInt(this.accountId, 10),
		"symbol": this.transSymbol(symbol),
		"size": strconv.Itoa(size),
	}
	queryString := this.sign(param)

	url := API_BASE_URL + OPEN_ORDERS + "?" + queryString

	var resp struct {
		Status string
		ErrCode string
		Data []OrderInfo
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", resp.ErrCode)
	}

	var ret = make([]OrderDecimal, len(resp.Data))
	for i := range resp.Data {
		ret[i] = *resp.Data[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *FullCoin) QueryOrder(orderId string) (*OrderDecimal, error) {
	var path string
	//path := fmt.Sprintf(QUERY_ORDER, orderId)
	queryString := this.sign(map[string]string {})

	url := API_BASE_URL + path + "?" + queryString
	var resp struct {
		Status string
		ErrCode string		`json:"err-code"`
		Data  *OrderInfo
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		if resp.ErrCode == "base-record-invalid" {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("bad status: %s", resp.ErrCode)
	}

	if resp.Data == nil {
		return nil, nil
	}

	symbol := this.getPairByName(resp.Data.Symbol)

	return resp.Data.ToOrderDecimal(symbol), nil
}
