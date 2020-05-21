package atop

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
	"encoding/base64"
	"log"
	"math"
)

const (
	OrderSell = 0
	OrderBuy = 1
)

const (
	OrderTypeLimit = 0
	OrderTypeMarket = 1
)

const (
	OrderStatusInit = 0
	OrderStatusPartiallyFilled = 1
	OrderStatusFilled = 2
	OrderStatusCanceled = 3
	OrderStatusSettle = 4
)

const (
	API_BASE_URL = "https://api.a.top"
	COMMON_SYMBOLS = "/data/api/v1/getMarketConfig"
	GET_TICKER = "/data/api/v1/getTicker?market=%s"
	GET_MARKET_DEPH = "/data/api/v1/getDepth?market=%s"
	GET_TRADES = "/data/api/v1/getTrades?market=%s"
	ACCOUNT = "/trade/api/v1/getBalance"
	CREATE_ORDER = "/trade/api/v1/order"
	BATCH_PLACE = "/trade/api/v1/batchOrder"
	CANCEL_ORDER = "/trade/api/v1/cancel"
	BATCH_CANCEL = "/trade/api/v1/batchCancel"
	NEW_ORDER = "/trade/api/v1/getOpenOrders"
	ORDER_INFO = "/trade/api/v1/getOrder"
)

var ErrNotExist = errors.New("NOT EXISTS")

type Atop struct {
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

func NewAtop(client *http.Client, ApiKey string, SecretKey string) *Atop {
	this := new(Atop)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = client

	this.symbolNameMap = make(map[string]string)
	return this
}

func (ok *Atop) GetSymbols() ([]Symbol, error) {
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

	var data map[string]Symbol

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var ret []Symbol
	for k, v := range data {
		v.Symbol = strings.ToUpper(k)
		ret = append(ret, v)
	}

	return ret, nil
}

func (this *Atop) transSymbol(symbol string) string {
	return strings.ToLower(symbol)
}

func (this *Atop) GetTicker(symbol string) (*TickerDecimal, error) {
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
		Info    string
		Code    decimal.Decimal

		High    decimal.Decimal
		CoinVol decimal.Decimal
		Price   decimal.Decimal
		Low     decimal.Decimal
		Ask     decimal.Decimal
		Bid     decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := &data

	ticker := new(TickerDecimal)
	ticker.Buy = r.Bid
	ticker.Sell = r.Ask
	ticker.Last = r.Price
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.CoinVol

	return ticker, nil
}

func (this *Atop) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_MARKET_DEPH, symbol)
	println(url)
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
		Info string
		Code decimal.Decimal

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

func (this *Atop) GetTrades(symbol string) ([]TradeDecimal, error) {
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

	if strings.Contains(string(body), "code") {
		var r struct {
			Code decimal.Decimal
			Info string
		}
		err = json.Unmarshal(body, &r)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error code: %s", r.Code.String())
	}

	var data [][]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var trades = make([]TradeDecimal, len(data))

	for i, o := range data {
		t := &trades[i]

		t.Tid = int64(o[4].(float64))
		t.Amount = decimal.NewFromFloat(o[2].(float64))
		t.Price = decimal.NewFromFloat(o[1].(float64))
		t.Type = strings.ToLower(o[3].(string))
	}

	return trades, nil
}

func (this *Atop) signData(data string) string {
	message := data
	sign, _ := GetParamHmacSHA256Sign(this.SecretKey, message)

	return sign
}

func (this *Atop) buildQueryString(param map[string]string) string {
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

func (this *Atop) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["nonce"] = timestamp
	param["accesskey"] = this.ApiKey

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
	data := strings.Join(parts, "&")

	sign := this.signData(data)
	param["signature"] = sign
	return param
}

func (this *Atop) getAuthHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *Atop) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string{}
	params = this.sign(params)

	url := API_BASE_URL + ACCOUNT + "?" + this.buildQueryString(params)

	var resp struct {
		Info string
		Code decimal.Decimal
		Data map[string]struct {
			Freeze    decimal.Decimal
			Available decimal.Decimal
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

	var ret []SubAccountDecimal
	for coin, o := range resp.Data {
		currency := strings.ToUpper(coin)
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Available,
			FrozenAmount: o.Freeze,
			Amount: o.Available.Add(o.Freeze),
		})
	}

	return ret, nil
}

func (this *Atop) PlaceOrder(volume decimal.Decimal, side int, _type int, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"market": symbol,
		"type": strconv.Itoa(side),
		"entrustType": strconv.Itoa(_type),
		"number": volume.String(),
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	println(data)
	url := API_BASE_URL + CREATE_ORDER
	body, err := HttpPostForm3(this.client, url, data, this.getAuthHeader())

	if err != nil {
		return "", err
	}
	println(string(body))

	var resp struct {
		Info string
		Code decimal.Decimal
		Data struct {
				 Id decimal.Decimal
			 }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 200 {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Data.Id.String(), nil
}

func (this *Atop) CancelOrder(symbol, orderId string) error {
	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"market": symbol,
		"id": orderId,
	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CANCEL_ORDER
	body, err := HttpPostForm3(this.client, url, data, this.getAuthHeader())
	if err != nil {
		return err
	}

	var resp struct {
		Info string
		Code decimal.Decimal
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Code.IntPart() == 404 {
		return ErrNotExist
	}

	if resp.Code.IntPart() != 200 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

func (this *Atop) QueryPendingOrders(symbol string, page, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 20
	}
	param := map[string]string{
		"market": this.transSymbol(symbol),
	}
	param["page"] = strconv.Itoa(page)
	param["pageSize"] = strconv.Itoa(pageSize)
	param = this.sign(param)

	url := fmt.Sprintf(API_BASE_URL + NEW_ORDER + "?" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data []OrderInfo
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() == 404 {
		return nil, nil
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

func (this *Atop) QueryOrder(symbol, orderId string) (*OrderDecimal, error) {
	param := this.sign(map[string]string{
		"market": this.transSymbol(symbol),
		"id": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + ORDER_INFO + "?" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Info string
		Data *OrderInfo
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() == 404 {
		return nil, ErrNotExist
	}

	if resp.Code.IntPart() != 200 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Data.ToOrderDecimal(symbol), nil
}

func (this *Atop) BatchPlace(symbol string, placeReqList []OrderReq) (orderIds []string, err error) {
	if len(placeReqList) == 0 {
		return nil, nil
	}

	orderIds = make([]string, len(placeReqList))

	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"market": symbol,
	}

	bytes, _ := json.Marshal(placeReqList)
	params["data"] = base64.StdEncoding.EncodeToString(bytes)

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + BATCH_PLACE
	var body []byte
	body, err = HttpPostForm3(this.client, url, data, this.getAuthHeader())

	if err != nil {
		return
	}

	var resp struct {
		Info string
		Code decimal.Decimal
		Data []struct {
			Amount float64
			Price  float64
			Id     decimal.Decimal
			Type   int
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}

	if resp.Code.IntPart() != 200 {
		err = fmt.Errorf("error code: %s", resp.Code.String())
		return
	}

	floatEquals := func(a, b float64) bool {
		diff := math.Abs(a - b)
		return diff < 0.00000000001
	}

	find := func(amount, price float64, _type int) int {
		for i, o := range placeReqList {
			if floatEquals(o.Amount, amount) && floatEquals(o.Price, price) && _type == o.Type {
				return i
			}
		}
		panic("not found")
		return -1
	}

	for i, o := range resp.Data {
		j := find(o.Amount, o.Price, o.Type)
		if i != j {
			log.Println("order req sequence not matched")
		}
		orderIds[j] = o.Id.String()
	}

	return
}

func (this *Atop) BatchCancel(symbol string, inOrderIds []string) (errors []error, err error) {
	if len(inOrderIds) == 0 {
		return nil, nil
	}

	var orderIds = make([]int64, len(inOrderIds))
	for i, id := range inOrderIds {
		orderIds[i], _ = strconv.ParseInt(id, 10, 64)
	}

	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"market": symbol,
	}

	bytes, _ := json.Marshal(orderIds)
	params["data"] = base64.StdEncoding.EncodeToString(bytes)

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + BATCH_CANCEL
	var body []byte
	body, err = HttpPostForm3(this.client, url, data, this.getAuthHeader())

	if err != nil {
		return
	}

	var resp struct {
		Info string
		Code decimal.Decimal
		Data []struct {
			Msg  string
			Code decimal.Decimal
			Id   decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}

	if resp.Code.IntPart() != 200 {
		err = fmt.Errorf("error code: %s", resp.Code.String())
		return
	}

	find := func(id string) int {
		for i, o := range inOrderIds {
			if o == id {
				return i
			}
		}
		panic("not found")
		return -1
	}

	errors = make([]error, len(inOrderIds))
	for i, o := range resp.Data {
		j := find(o.Id.String())
		if i != j {
			log.Println("order req sequence not matched")
		}
		if o.Code.IntPart() != 120 && o.Code.IntPart() != 121 {
			errors[j] = fmt.Errorf("error code: %s", o.Code.String())
		}
	}

	return
}
