package appex

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
)

const (
	SIDE_BUY = "buy"
	SIDE_SELL = "sell"

	TYPE_LIMIT = "limit"
	TYPE_MARKET = "market"
)

const (
	HOST = "www.appex.pro"
	API_BASE_URL = "https://www.appex.pro/api"
	SYMBOL = "/v1/common/symbols"
	TICKER = "/market/detail/merged?symbol=%s"
	DEPTH = "/market/depth?symbol=%s&type=step0&depth=20"
	TRADE = "/market/trade?symbol=%s"
	ACCOUNTS = "/v1/account/accounts"
	ACCOUNT_BALANCE = "/v1/account/accounts/%d/balance"
	PLACE_ORDER = "/v1/order/orders/place"
	CANCEL_ORDER = "/v1/order/orders/%s/submitcancel"
	BATCH_CANCEL = "/v1/order/orders/batchcancel"
	OPEN_ORDERS = "/v1/order/openOrders"
	QUERY_ORDER = "/v1/order/orders/%s"
)

var (
	ErrOrderStateError = errors.New("order-orderstate-error")
	ErrNotExist = errors.New("not exist")
)

type Appex struct {
	ApiKey string
	SecretKey string
	client *http.Client

	accountId int64
	symbolNameMap map[string]string
}

func NewAppex(ApiKey string, SecretKey string) *Appex {
	this := new(Appex)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *Appex) getPairByName(name string) string {
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
		this.symbolNameMap[strings.ToUpper(o.Symbol)] = fmt.Sprintf("%s_%s", o.BaseCurrency, o.QuoteCurrency)
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (this *Appex) GetSymbols() ([]Symbol, error) {
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
		Status string
		Ch     string
		Ts     int64
		Data   []Symbol
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", data.Status)
	}

	return data.Data, nil
}

func (this *Appex) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Appex) GetTicker(symbol string) (*TickerDecimal, error) {
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
		Status string
		Ch     string
		Ts     int64
		Tick   struct {
			Open decimal.Decimal
			High decimal.Decimal
			Vol decimal.Decimal
			Close decimal.Decimal
			Low decimal.Decimal
			Bid []decimal.Decimal
			Ask []decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", data.Status)
	}
	r := data.Tick

	ticker := new(TickerDecimal)
	ticker.Date = uint64(data.Ts)
	ticker.Buy = r.Bid[0]
	ticker.Sell = r.Ask[0]
	ticker.Last = r.Close
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Open = r.Open
	ticker.Vol = r.Vol

	return ticker, nil
}

func (this *Appex) GetDepth(symbol string) (*DepthDecimal, error) {
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
		Status string
		Ch     string
		Ts     int64
		Tick struct {
				 Asks [][]decimal.Decimal
				 Bids [][]decimal.Decimal
			 }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", data.Status)
	}

	r := data.Tick

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

func (this *Appex) GetTrades(symbol string) ([]TradeDecimal, error) {
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
		Status string
		Ch     string
		Ts     int64
		Tick struct {
			Data [] struct {
				Amount decimal.Decimal
				Price decimal.Decimal
				Id decimal.Decimal
				Ts int64
				Direction string
			}
			Ts int64
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", data.Status)
	}

	var trades = make([]TradeDecimal, len(data.Tick.Data))

	for i, o := range data.Tick.Data {
		t := &trades[i]
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Direction
		t.Date = o.Ts
	}

	return trades, nil
}

func (this *Appex) signData(data string) string {
	sign, _ := GetParamHmacSHA256Base64Sign(this.SecretKey, data)

	return sign
}

func (this *Appex) sign(method, reqUrl string, param map[string]string) string {
	now := time.Now().In(time.UTC)
	param["AccessKeyId"] = this.ApiKey
	param["SignatureMethod"] = "HmacSHA256"
	param["SignatureVersion"] = "2"
	param["Timestamp"] = now.Format("2006-01-02T15:04:05")
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + "=" + url.QueryEscape(param[k]))
	}
	data := strings.Join(parts, "&")

	lines := []string {
		method,
		HOST,
		reqUrl,
		data,
	}

	message := strings.Join(lines, "\n")
	sign := this.signData(message)
	return data + "&Signature=" + url.QueryEscape(sign)
}

func (this *Appex) GetAccounts() ([]int64, error) {
	params := map[string]string {}
	queryString := this.sign("GET", ACCOUNTS, params)

	url := API_BASE_URL + ACCOUNTS + "?" + queryString
	var resp struct {
		Status string
		Data []struct {
				 Id int64
				 Type string
				 State string
			 }
	}

	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var ret []int64
	for _, r := range resp.Data {
		ret = append(ret, r.Id)
	}

	return ret, nil
}

func (this *Appex) GetAccountBalance(accountId int64) ([]SubAccountDecimal, error) {
	params := map[string]string {}
	path := fmt.Sprintf(ACCOUNT_BALANCE, accountId)
	queryString := this.sign("GET", path, params)

	url := API_BASE_URL + path + "?" + queryString
	var resp struct {
		Data struct {
				 Id int64
				 Type string
				 State string
				 UserId decimal.Decimal 		`json:"user-id"`
				 List []struct {
					 Currency string
					 Type string
					 Balance decimal.Decimal
				 }
			 }
	}

	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	var m = make(map[string]*SubAccountDecimal)
	for _, o := range resp.Data.List {
		currency := strings.ToUpper(o.Currency)
		if currency == "" {
			continue
		}
		if _, ok := m[currency]; !ok {
			m[currency] = &SubAccountDecimal{
				Currency: Currency{Symbol: currency},
			}
		}

		if o.Type == "trade" {
			m[currency].AvailableAmount = o.Balance
		} else {
			m[currency].FrozenAmount = o.Balance
		}
	}

	var ret []SubAccountDecimal
	for _, o := range m {
		o.Amount = o.AvailableAmount.Add(o.FrozenAmount)
		ret = append(ret, *o)
	}

	return ret, nil
}

func (this *Appex) ensureAccountId() error {
	if this.accountId == 0 {
		accountIds, err := this.GetAccounts()
		if err != nil {
			return err
		}
		if len(accountIds) == 0 {
			return errors.New("No account id")
		}
		this.accountId = accountIds[0]
	}
	return nil
}

func (this *Appex) GetAccount() ([]SubAccountDecimal, error) {
	err := this.ensureAccountId()
	if err != nil {
		return nil, err
	}
	return this.GetAccountBalance(this.accountId)
}

func (this *Appex) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	err := this.ensureAccountId()
	if err != nil {
		return "", err
	}

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

	queryString := this.sign("POST", PLACE_ORDER, map[string]string{})

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

func (this *Appex) CancelOrder(orderId string) error {
	path := fmt.Sprintf(CANCEL_ORDER, orderId)
	queryString := this.sign("POST", path, map[string]string{})

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

func (this *Appex) CancelOrders(orderIds []string) (error, []error) {
	params := map[string]interface{} {
		"order-ids": orderIds,
	}

	queryString := this.sign("POST", BATCH_CANCEL, map[string]string{})

	data, _ := json.Marshal(params)

	url := API_BASE_URL + BATCH_CANCEL + "?" + queryString
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

func (this *Appex) QueryPendingOrders(symbol string, size int) ([]OrderDecimal, error) {
	err := this.ensureAccountId()
	if err != nil {
		return nil, err
	}
	if size == 0 {
		size = 10
	}
	param := map[string]string {
		"account-id": strconv.FormatInt(this.accountId, 10),
		"symbol": this.transSymbol(symbol),
		"size": strconv.Itoa(size),
	}
	queryString := this.sign("GET", OPEN_ORDERS, param)

	url := API_BASE_URL + OPEN_ORDERS + "?" + queryString

	var resp struct {
		Status string
		ErrCode string
		Data []OrderInfo
	}

	err = HttpGet4(this.client, url, nil, &resp)
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

func (this *Appex) QueryOrder(orderId string) (*OrderDecimal, error) {
	path := fmt.Sprintf(QUERY_ORDER, orderId)
	queryString := this.sign("GET", path, map[string]string {})

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
