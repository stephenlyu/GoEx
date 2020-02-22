package bitribe

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	. "github.com/stephenlyu/GoEx"
	"strconv"
	"sort"
	"net/url"
	"sync"
	"errors"
)

const (
	OrderSell = "SELL"
	OrderBuy = "BUY"
)

const (
	OrderTypeLimit = "LIMIT"
	OrderTypeMarket = "MARKET"
)

const (
	OrderStatusNew = "NEW"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled = "FILLED"
	OrderStatusCanceled = "CANCELED"
	OrderStatusPendingCancel = "PENDING_CANCEL"
	OrderStatusRejected = "REJECTED"
)

const (
	API_BASE_URL    = "https://api.bitribe.com"
	COMMON_SYMBOLS = "/openapi/v1/brokerInfo"
	GET_TICKER = "/openapi/quote/v1/ticker/24hr?symbol=%s"
	GET_MARKET_DEPH = "/openapi/quote/v1/depth?symbol=%s&limit=5"
	GET_TRADES = "/openapi/quote/v1/trades?symbol=%s&limit=1"
	ACCOUNT = "/openapi/v1/account"
	CREATE_ORDER = "/openapi/v1/order"
	CANCEL_ORDER = "/openapi/v1/order"
	NEW_ORDER = "/openapi/v1/openOrders"
	ORDER_INFO = "/openapi/v1/order"
)

var ErrNotExist = errors.New("NOT EXISTS")

type Bitribe struct {
	ApiKey    string
	SecretKey string
	client    *http.Client

	symbolNameMap map[string]string

	ws                *WsConn
	createWsLock      sync.Mutex
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	errorHandle      func(error)
	wsSymbolMap map[string]string
}

func NewBitribe(client *http.Client, ApiKey string, SecretKey string) *Bitribe {
	this := new(Bitribe)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = client

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *Bitribe) getPairByName(name string) string {
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
		oSymbol := fmt.Sprintf("%s%s", o.BaseAsset, o.QuoteAsset)
		this.symbolNameMap[strings.ToUpper(oSymbol)] = fmt.Sprintf("%s_%s", o.BaseAsset, o.QuoteAsset)
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (ok *Bitribe) GetSymbols() ([]Symbol, error) {
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
		Symbols []Symbol
		Msg     string
		Code    decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	for i := range data.Symbols {
		s := &data.Symbols[i]
		s.Symbol = strings.ToUpper(fmt.Sprintf("%s_%s", s.BaseAsset, s.QuoteAsset))
	}

	return data.Symbols, nil
}

func (this *Bitribe) transSymbol(symbol string) string {
	return strings.ToUpper(strings.Replace(symbol, "_", "", -1))
}

func (this *Bitribe) GetTicker(symbol string) (*TickerDecimal, error) {
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
		Msg          string
		Code         decimal.Decimal

		HighPrice    decimal.Decimal
		Volume       decimal.Decimal
		LastPrice    decimal.Decimal
		LowPrice     decimal.Decimal
		OpenPrice    decimal.Decimal
		BestBidPrice decimal.Decimal
		BestAskPrice decimal.Decimal
		Time         int64
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data

	ticker := new(TickerDecimal)
	ticker.Date = uint64(r.Time)
	ticker.Buy = r.BestBidPrice
	ticker.Sell = r.BestAskPrice
	ticker.Open = r.OpenPrice
	ticker.Last = r.LastPrice
	ticker.High = r.HighPrice
	ticker.Low = r.LowPrice
	ticker.Vol = r.Volume

	return ticker, nil
}

func (this *Bitribe) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_MARKET_DEPH, symbol)
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
		Msg string
		Code decimal.Decimal
		Asks [][]decimal.Decimal
		Bids [][]decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data

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

func (this *Bitribe) GetTrades(symbol string) ([]TradeDecimal, error) {
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

	if strings.HasPrefix(string(body), "{") {
		var data struct {
			Msg string
			Code decimal.Decimal
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}

		if data.Code.IntPart() != 0 {
			return nil, fmt.Errorf("error code: %s", data.Code.String())
		}
		panic("Unreachable code")
	}

	var data []struct {
		Price decimal.Decimal
		Qty decimal.Decimal
		Time int64
		IsBuyerMaker bool
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var trades = make([]TradeDecimal, len(data))

	for i, o := range data {
		t := &trades[i]
		t.Tid = o.Time
		t.Amount = o.Qty
		t.Price = o.Price
		if o.IsBuyerMaker {
			t.Type = "sell"
		} else {
			t.Type = "buy"
		}
		t.Date = o.Time
	}

	return trades, nil
}

func (this *Bitribe) signData(data string) string {
	message := data
	sign, _ := GetParamHmacSHA256Sign(this.SecretKey, message)

	return sign
}

func (this *Bitribe) buildQueryString(param map[string]string) string {
	var parts []string

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		v := param[k]
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

func (this *Bitribe) buildQueryStringUnescape(param map[string]string) string {
	var parts []string
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		v := param[k]
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, "&")
}

func (this *Bitribe) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["timestamp"] = timestamp

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	data := this.buildQueryStringUnescape(param)

	sign := this.signData(data)
	param["signature"] = sign
	return param
}

func (this *Bitribe) getAuthHeader() map[string]string {
	return map[string]string {
		"X-BH-APIKEY": this.ApiKey,
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *Bitribe) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	params = this.sign(params)

	url := API_BASE_URL + ACCOUNT + "?" + this.buildQueryString(params)

	var resp struct {
		Msg string
		Code decimal.Decimal
		Balances []struct {
			Asset string
			Free decimal.Decimal
			Locked decimal.Decimal
		}
	}

	header := this.getAuthHeader()
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret []SubAccountDecimal
	for _, o := range resp.Balances {
		currency := strings.ToUpper(o.Asset)
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Free,
			FrozenAmount: o.Locked,
			Amount: o.Free.Add(o.Locked),
		})
	}

	return ret, nil
}

func (this *Bitribe) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	params := map[string]string {
		"symbol": symbol,
		"side": side,
		"type": _type,
		"quantity": volume.String(),
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CREATE_ORDER
	body, err := HttpPostForm3(this.client, url, data, this.getAuthHeader())

	if err != nil {
		return "", err
	}

	var resp struct {
		Msg string
		Code decimal.Decimal
		OrderId decimal.Decimal
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 0 {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.OrderId.String(), nil
}

func (this *Bitribe) CancelOrder(orderId, clientOrderId string) error {
	params := map[string]string {
	}
	if orderId != "" {
		params["orderId"] = orderId
	}
	if clientOrderId != "" {
		params["clientOrderId"] = clientOrderId
	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CANCEL_ORDER + "?" + data
	body, err := HttpDeleteForm3(this.client, url, "", this.getAuthHeader())

	if err != nil {
		if strings.Contains(err.Error(), "-2013") {
			return ErrNotExist
		}
		return err
	}

	var resp struct {
		Msg string
		Code decimal.Decimal
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Code.IntPart() != 0 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

func (this *Bitribe) QueryPendingOrders(symbol string, orderId string, limit int) ([]OrderDecimal, error) {
	param := map[string]string {
		"symbol": this.transSymbol(symbol),
	}
	if orderId != "" {
		param["orderId"] = orderId
	}
	if limit > 0 {
		param["limit"] = strconv.Itoa(limit)
	}
	param = this.sign(param)

	url := fmt.Sprintf(API_BASE_URL + NEW_ORDER + "?" + this.buildQueryString(param))

	bytes, err := HttpGet6(this.client, url, this.getAuthHeader())
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(strings.TrimSpace(string(bytes)), "{") {
		var resp struct {
			Msg string
			Code decimal.Decimal
		}
		err = json.Unmarshal(bytes, &resp)
		if err != nil {
			return nil, err
		}
		if resp.Code.IntPart() != 0 {
			return nil, fmt.Errorf("error code: %s", resp.Code.String())
		}
		panic("unreachable code")
	}

	var data []OrderInfo
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	var ret = make([]OrderDecimal, len(data))
	for i := range data {
		symbol := this.getPairByName(data[i].Symbol)
		ret[i] = *data[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *Bitribe) QueryOrder(orderId string) (*OrderDecimal, error) {
	param := this.sign(map[string]string {
		"orderId": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + ORDER_INFO + "?" + this.buildQueryString(param))

	var resp OrderInfo

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		if strings.Contains(err.Error(), "-2013") {
			return nil, ErrNotExist
		}
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	symbol := this.getPairByName(resp.Symbol)

	return resp.ToOrderDecimal(symbol), nil
}
