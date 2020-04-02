package zingex

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	. "github.com/stephenlyu/GoEx"
	"sort"
	"net/url"
	"sync"
	"errors"
	"crypto/tls"
)

const (
	OrderBuy = "0"
	OrderSell = "1"
)

const (
	OrderStatusNew = 1
	OrderStatusPartiallyFilled = 2
	OrderStatusFilled = 3
	OrderStatusCanceled = 4
)

const (
	API_BASE_URL = "https://tinance.pro"
	COMMON_SYMBOLS = "/appApi.json?action=tickers"
	GET_TICKER = "/appApi.json?action=market&symbol=%s"
	GET_MARKET_DEPTH = "/appApi.json?action=depth&symbol=%s&size=30"
	GET_TRADES = "/appApi.json?action=trades&symbol=%s"
	ACCOUNT = "/appApi.json?action=userinfo"
	CREATE_ORDER = "/appApi.json?action=trade"
	CANCEL_ORDER = "/appApi.json?action=cancel_entrust"
	NEW_ORDER = "/appApi.json?action=entrust"
	ORDER_INFO = "/appApi.json?action=order"
)

var ErrNotExist = errors.New("NOT EXISTS")

type ZingEx struct {
	ApiKey           string
	SecretKey        string
	client           *http.Client

	symbolNameMap    map[string]string

	ws               *WsConn
	createWsLock     sync.Mutex
	wsDepthHandleMap map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	errorHandle      func(error)
	wsSymbolMap      map[string]string
}

func NewZingEx(client *http.Client, ApiKey string, SecretKey string) *ZingEx {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = tr

	this := new(ZingEx)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = client

	this.symbolNameMap = map[string]string{
		"btc_usdt": "1",
		"eth_usdt": "2",
		"leee_usdt": "3",
		"leee_eth": "5",
		"odin_usdt": "6",
	}
	return this
}

func (ok *ZingEx) GetSymbols() ([]Symbol, error) {
	url := API_BASE_URL + COMMON_SYMBOLS
	resp, err := ok.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Ticker []Symbol
		Msg    string
		Code   decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	for i := range data.Ticker {
		s := &data.Ticker[i]
		s.Symbol = strings.ToUpper(fmt.Sprintf("%s_%s", s.BaseCoin, s.CountCoin))
	}

	return data.Ticker, nil
}

func (this *ZingEx) transSymbol(symbol string) string {
	symbol = strings.ToLower(symbol)
	return this.symbolNameMap[symbol]
}

func (this *ZingEx) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_TICKER, symbol)
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
		Msg  string
		Code decimal.Decimal

		Time int64
		Data struct {
				 High decimal.Decimal
				 Vol  decimal.Decimal
				 Last decimal.Decimal
				 Low  decimal.Decimal
				 Buy  decimal.Decimal
				 Sell decimal.Decimal
			 }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data.Data

	ticker := new(TickerDecimal)
	ticker.Date = uint64(data.Time)
	ticker.Buy = r.Buy
	ticker.Sell = r.Sell
	ticker.Last = r.Last
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol

	return ticker, nil
}

func (this *ZingEx) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_MARKET_DEPTH, symbol)
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
		Msg  string
		Code decimal.Decimal
		Data struct {
				 Asks [][]decimal.Decimal
				 Bids [][]decimal.Decimal
			 }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data.Data

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

func (this *ZingEx) GetTrades(symbol string) ([]TradeDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_TRADES, symbol)
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
		Msg  string
		Code decimal.Decimal
		Data []struct {
			Amount decimal.Decimal
			Price  decimal.Decimal
			ID     int64
			Type   string    `json:"en_type"`
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	var trades = make([]TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Tid = o.ID
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = strings.ToLower(o.Type)
	}

	return trades, nil
}

func (this *ZingEx) signData(data string) string {
	message := data
	sign, _ := GetParamMD5Sign(this.SecretKey, message)

	return strings.ToUpper(sign)
}

func (this *ZingEx) buildQueryString(param map[string]string) string {
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

func (this *ZingEx) sign(param map[string]string) map[string]string {
	param["api_key"] = this.ApiKey

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
		parts = append(parts, key + "=" + value)
	}
	parts = append(parts, "secret_key=" + this.SecretKey)
	data := strings.Join(parts, "&")

	sign := this.signData(data)
	param["sign"] = sign
	return param
}

func (this *ZingEx) getAuthHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *ZingEx) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string{}
	params = this.sign(params)

	url := API_BASE_URL + ACCOUNT + "&" + this.buildQueryString(params)

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
				 Frozen map[string]decimal.Decimal
				 Free   map[string]decimal.Decimal
			 }
	}

	header := this.getAuthHeader()
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var m = make(map[string]*SubAccountDecimal)
	for currency, amount := range resp.Data.Free {
		currency := strings.ToUpper(currency)
		m[currency] = &SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: amount,
		}
	}

	for currency, amount := range resp.Data.Frozen {
		o, ok := m[currency]
		if !ok {
			m[currency] = &SubAccountDecimal{
				Currency: Currency{Symbol: currency},
				FrozenAmount: amount,
			}
		} else {
			o.FrozenAmount = amount
		}
	}

	var ret []SubAccountDecimal
	for _, o := range m {
		o.Amount = o.AvailableAmount.Add(o.FrozenAmount)
		ret = append(ret, *o)
	}

	return ret, nil
}

func (this *ZingEx) PlaceOrder(volume decimal.Decimal, side string, symbol string, price decimal.Decimal) (string, error) {
	params := map[string]string{
		"symbol": this.transSymbol(symbol),
		"type": side,
		"amount": volume.String(),
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CREATE_ORDER + "&" + data
	body, err := HttpPostForm3(this.client, url, "", this.getAuthHeader())

	if err != nil {
		return "", err
	}

	var resp struct {
		Msg     string
		Code    decimal.Decimal
		OrderId decimal.Decimal
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 200 {
		return "", fmt.Errorf("error code: %s msg: %s", resp.Code.String(), resp.Msg)
	}

	return resp.OrderId.String(), nil
}

func (this *ZingEx) CancelOrder(orderId string) error {
	params := map[string]string{
		"id": orderId,
	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CANCEL_ORDER + "&" + data
	body, err := HttpPostForm3(this.client, url, "", this.getAuthHeader())
	if err != nil {
		return err
	}

	var resp struct {
		Msg  string
		Code decimal.Decimal
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Code.IntPart() != 200 && resp.Code.IntPart() != 0 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

func (this *ZingEx) QueryPendingOrders(symbol string) ([]OrderDecimal, error) {
	param := map[string]string{
		"symbol": this.transSymbol(symbol),
	}
	param = this.sign(param)

	url := fmt.Sprintf(API_BASE_URL + NEW_ORDER + "&" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data []OrderInfo
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]OrderDecimal, len(resp.Data))
	for i := range resp.Data {
		ret[i] = *resp.Data[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *ZingEx) QueryOrder(symbol, orderId string) (*OrderDecimal, error) {
	param := this.sign(map[string]string{
		"id": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + ORDER_INFO + "&" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data []OrderInfo
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	if len(resp.Data) == 0 {
		return nil, ErrNotExist
	}

	return resp.Data[0].ToOrderDecimal(symbol), nil
}
