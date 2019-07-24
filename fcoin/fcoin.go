package fcoin

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"net/http"
	"net/url"
	"strings"
	"time"
	"encoding/json"
	"sync"
	"io/ioutil"
	"github.com/shopspring/decimal"
	"strconv"
)

type FCoinTicker struct {
	Ticker
	SellAmount,
	BuyAmount float64
}

type FCoin struct {
	httpClient *http.Client
	baseUrl,
	accessKey,
	secretKey string
	timeoffset   int64
	tradeSymbols []TradeSymbol

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

type TradeSymbol struct {
	Name          string `json:"name"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	PriceDecimal  int    `json:"price_decimal"`
	AmountDecimal int    `json:"amount_decimal"`
	Tradable      bool   `json:"tradable"`
}

type Asset struct {
	Currency  Currency
	Avaliable float64
	Frozen    float64
	Finances  float64
	Lock      float64
	Total     float64
}

func NewFCoin(client *http.Client, apikey, secretkey string) *FCoin {
	fc := &FCoin{baseUrl: "https://api.fcoin.com/v2/", accessKey: apikey, secretKey: secretkey, httpClient: client}
	fc.setTimeOffset()
	var err error
	fc.tradeSymbols, err = fc.GetTradeSymbols()
	if len(fc.tradeSymbols) == 0 || err != nil {
		panic("trade symbol is empty, pls check connection...")
	}

	return fc
}

func (fc *FCoin) GetExchangeName() string {
	return FCOIN
}

func (fc *FCoin) setTimeOffset() error {
	respmap, err := HttpGet(fc.httpClient, fc.baseUrl+"public/server-time")
	if err != nil {
		return err
	}
	stime := int64(ToInt(respmap["data"]))
	st := time.Unix(stime/1000, 0)
	lt := time.Now()
	offset := st.Sub(lt).Seconds()
	fc.timeoffset = int64(offset)
	return nil
}

func (fc *FCoin) GetTicker(currencyPair CurrencyPair) (*TickerDecimal, error) {
	reqUrl := fc.baseUrl + fmt.Sprintf("market/ticker/%s", strings.ToLower(currencyPair.ToSymbol("")))

	var resp struct {
		Status int
		Msg string
		Data struct {
			 Ticker []decimal.Decimal
			 }
	}

	err := HttpGet4(fc.httpClient, reqUrl, nil, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Msg)
	}

	t := resp.Data.Ticker

	ticker := new(TickerDecimal)
	ticker.Pair = currencyPair
	ticker.Date = uint64(time.Now().UnixNano() / 1000000)
	ticker.Last = t[0]
	ticker.Vol = t[9]
	ticker.Low = t[8]
	ticker.High = t[7]
	ticker.Buy = t[2]
	ticker.Sell = t[4]
	return ticker, nil
}

func (fc *FCoin) GetDepth(size int, currency CurrencyPair) (*DepthDecimal, error) {
	var uri string
	if size <= 20 {
		uri = fmt.Sprintf("market/depth/L20/%s", strings.ToLower(currency.ToSymbol("")))
	} else {
		uri = fmt.Sprintf("market/depth/L150/%s", strings.ToLower(currency.ToSymbol("")))
	}

	var resp struct {
		Status int
		Msg string
		Data struct {
			Asks []decimal.Decimal
			Bids []decimal.Decimal
			 }
	}

	err := HttpGet4(fc.httpClient, fc.baseUrl+uri, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Msg)
	}

	bids := resp.Data.Bids
	asks := resp.Data.Asks

	depth := new(DepthDecimal)
	depth.Pair = currency

	n := 0
	for i := 0; i < len(bids); {
		depth.BidList = append(depth.BidList, DepthRecordDecimal{bids[i], bids[i+1]})
		i += 2
		n++
		if n == size {
			break
		}
	}

	n = 0
	for i := 0; i < len(asks); {
		depth.AskList = append(depth.AskList, DepthRecordDecimal{asks[i], asks[i+1]})
		i += 2
		n++
		if n == size {
			break
		}
	}

	return depth, nil
}

func (fc *FCoin) getAuthenticatedHeader(method, uri string, params url.Values) map[string]string {
	timestamp := time.Now().Unix()*1000 + fc.timeoffset*1000
	sign := fc.buildSigned(method, fc.baseUrl+uri, timestamp, params)

	return map[string]string{
		"FC-ACCESS-KEY":       fc.accessKey,
		"FC-ACCESS-SIGNATURE": sign,
		"FC-ACCESS-TIMESTAMP": fmt.Sprint(timestamp)}
}

func (fc *FCoin) doAuthenticatedRequest(method, uri string, params url.Values) (interface{}, error) {
	header := fc.getAuthenticatedHeader(method, uri, params)

	var (
		respmap map[string]interface{}
		err     error
	)

	switch method {
	case "GET":
		respmap, err = HttpGet2(fc.httpClient, fc.baseUrl+uri+"?"+params.Encode(), header)
		if err != nil {
			return nil, err
		}

	case "POST":
		var parammap = make(map[string]string, 1)
		for k, v := range params {
			parammap[k] = v[0]
		}

		respbody, err := HttpPostForm4(fc.httpClient, fc.baseUrl+uri, parammap, header)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(respbody, &respmap)
	}
	if ToInt(respmap["status"]) != 0 {
		return nil, errors.New(respmap["msg"].(string))
	}

	return respmap["data"], err
}

func (fc *FCoin) buildSigned(httpmethod string, apiurl string, timestamp int64, para url.Values) string {

	var (
		param = ""
		err   error
	)

	if para != nil {
		param = para.Encode()
	}

	if "GET" == httpmethod && param != "" {
		apiurl += "?" + param
	}

	signStr := httpmethod + apiurl + fmt.Sprint(timestamp)
	if "POST" == httpmethod && param != "" {
		signStr += param
	}

	signStr2, err := url.QueryUnescape(signStr) // 不需要编码
	if err != nil {
		signStr2 = signStr
	}

	sign := base64.StdEncoding.EncodeToString([]byte(signStr2))

	mac := hmac.New(sha1.New, []byte(fc.secretKey))

	mac.Write([]byte(sign))
	sum := mac.Sum(nil)

	s := base64.StdEncoding.EncodeToString(sum)
	return s
}

func (fc *FCoin) PlaceOrder(orderType, orderSide, amount, price string, pair CurrencyPair) (string, error) {
	params := url.Values{}

	params.Set("side", orderSide)
	params.Set("amount", amount)
	//params.Set("price", price)
	params.Set("symbol", strings.ToLower(pair.AdaptUsdToUsdt().ToSymbol("")))

	switch orderType {
	case "LIMIT", "limit":
		params.Set("price", price)
		params.Set("type", "limit")
	case "MARKET", "market":
		params.Set("type", "market")
	}

	r, err := fc.doAuthenticatedRequest("POST", "orders", params)
	if err != nil {
		return "", err
	}

	return r.(string), nil
}

func (fc *FCoin) LimitBuy(amount, price string, currency CurrencyPair) (string, error) {
	return fc.PlaceOrder("limit", "buy", amount, price, currency)
}

func (fc *FCoin) LimitSell(amount, price string, currency CurrencyPair) (string, error) {
	return fc.PlaceOrder("limit", "sell", amount, price, currency)
}

func (fc *FCoin) MarketBuy(amount, price string, currency CurrencyPair) (string, error) {
	return fc.PlaceOrder("market", "buy", amount, price, currency)
}

func (fc *FCoin) MarketSell(amount, price string, currency CurrencyPair) (string, error) {
	return fc.PlaceOrder("market", "sell", amount, price, currency)
}

func (fc *FCoin) CancelOrder(orderId string, currency CurrencyPair) (error) {
	uri := fmt.Sprintf("orders/%s/submit-cancel", orderId)
	_, err := fc.doAuthenticatedRequest("POST", uri, url.Values{})
	return err
}

type OrderInfo struct {
	Id string
	Symbol string
	Type string
	Side string
	Price decimal.Decimal
	Amount decimal.Decimal
	State string
	ExecutedValue decimal.Decimal 	`json:"executed_value"`
	FillFees decimal.Decimal 		`json:"fill_fees"`
	FilledAmount decimal.Decimal	`json:"filled_amount"`
	CreateAt int64 					`json:"created_at"`
	Source string 					`json:"source"`
}
func (this *OrderInfo) ToOrderDecimal(pair CurrencyPair) *OrderDecimal {
	side := SELL
	if this.Side == "buy" {
		side = BUY
	}

	orderStatus := ORDER_UNFINISH
	switch this.State {
	case "partial_filled":
		orderStatus = ORDER_PART_FINISH
	case "filled":
		orderStatus = ORDER_FINISH
	case "pending_cancel":
		orderStatus = ORDER_CANCEL_ING
	case "canceled", "partial_canceled":
		orderStatus = ORDER_CANCEL
	}
	return &OrderDecimal{
		Currency:   pair,
		Side:       TradeSide(side),
		OrderID2:   this.Id,
		Amount:     this.Amount,
		Price:      this.Price,
		DealAmount: this.FilledAmount,
		Status:     TradeStatus(orderStatus),
		Fee:        this.FillFees,
		Timestamp:  this.CreateAt,
	}
}

func (fc *FCoin) GetOneOrder(orderId string, currency CurrencyPair) (*OrderDecimal, error) {
	uri := fmt.Sprintf("orders/%s", orderId)
	header := fc.getAuthenticatedHeader("GET", uri, url.Values{})

	var resp struct {
		Status int
		Msg string
		Data *OrderInfo
	}

	err := HttpGet4(fc.httpClient, fc.baseUrl + uri, header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Msg)
	}

	return resp.Data.ToOrderDecimal(currency), nil
}

func (fc *FCoin) GetUnfinishedOrders(currency CurrencyPair, from, to int64, limit int) ([]OrderDecimal, error) {
	if limit == 0 {
		limit = 100
	}
	params := url.Values{}
	params.Set("symbol", strings.ToLower(currency.AdaptUsdToUsdt().ToSymbol("")))
	params.Set("states", "submitted,partial_filled")
	if from > 0 {
		params.Set("after", strconv.FormatInt(from, 10))
	}
	if to > 0 {
		params.Set("before", strconv.FormatInt(to, 10))
	}
	params.Set("limit", strconv.Itoa(limit))
	header := fc.getAuthenticatedHeader("GET", "orders", params)

	var resp struct {
		Status int
		Msg string
		Data []OrderInfo
	}

	err := HttpGet4(fc.httpClient, fc.baseUrl + "orders?" + params.Encode(), header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Msg)
	}

	var ords []OrderDecimal

	for _, ord := range resp.Data {
		ords = append(ords, *ord.ToOrderDecimal(currency))
	}

	return ords, nil
}

func (fc *FCoin) GetFinishedOrders(currency CurrencyPair, from, to int64, limit int) ([]OrderDecimal, error) {
	if limit == 0 {
		limit = 100
	}
	params := url.Values{}
	params.Set("symbol", strings.ToLower(currency.AdaptUsdToUsdt().ToSymbol("")))
	params.Set("states", "partial_canceled,filled,canceled")
	if from > 0 {
		params.Set("after", strconv.FormatInt(from, 10))
	}
	if to > 0 {
		params.Set("before", strconv.FormatInt(to, 10))
	}
	params.Set("limit", strconv.Itoa(limit))

	header := fc.getAuthenticatedHeader("GET", "orders", params)

	var resp struct {
		Status int
		Msg string
		Data []OrderInfo
	}

	err := HttpGet4(fc.httpClient, fc.baseUrl + "orders?" + params.Encode(), header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Msg)
	}

	var ords []OrderDecimal

	for _, ord := range resp.Data {
		ords = append(ords, *ord.ToOrderDecimal(currency))
	}

	return ords, nil
}

func (fc *FCoin) GetAccount() (*Account, error) {
	r, err := fc.doAuthenticatedRequest("GET", "accounts/balance", url.Values{})
	if err != nil {
		return nil, err
	}
	acc := new(Account)
	acc.SubAccounts = make(map[Currency]SubAccount)
	acc.Exchange = fc.GetExchangeName()

	balances := r.([]interface{})
	for _, v := range balances {
		vv := v.(map[string]interface{})
		currency := NewCurrency(vv["currency"].(string), "")
		acc.SubAccounts[currency] = SubAccount{
			Currency:     currency,
			Amount:       ToFloat64(vv["available"]),
			ForzenAmount: ToFloat64(vv["frozen"]),
		}
	}
	return acc, nil
}

func (fc *FCoin) GetAssets() ([]Asset, error) {
	r, err := fc.doAuthenticatedRequest("GET", "assets/accounts/balance", url.Values{})
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0)
	balances := r.([]interface{})
	for _, v := range balances {
		vv := v.(map[string]interface{})
		currency := NewCurrency(vv["currency"].(string), "")
		assets = append(assets, Asset{
			Currency:  currency,
			Avaliable: ToFloat64(vv["available"]),
			Frozen:    ToFloat64(vv["frozen"]),
			Finances:  ToFloat64(vv["demand_deposit"]),
			Lock:      ToFloat64(vv["lock_deposit"]),
			Total:     ToFloat64(vv["balance"]),
		})
	}
	return assets, nil
}

// from, to: assets, spot
func (fc *FCoin) AssetTransfer(currency Currency, amount, from, to string) (bool, error) {
	params := url.Values{}
	params.Set("currency", strings.ToLower(currency.String()))
	params.Set("amount", amount)
	_, err := fc.doAuthenticatedRequest("POST", fmt.Sprintf("assets/accounts/%s-to-%s", from, to), params)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (fc *FCoin) GetKlineRecords(currency CurrencyPair, period, size, since int) ([]Kline, error) {
	panic("not implement")
}

//非个人，整个交易所的交易记录
func (fc *FCoin) GetTrades(currencyPair CurrencyPair, since int64) ([]TradeDecimal, error) {
	url := fmt.Sprintf(fc.baseUrl + "market/trades/%s", strings.ToLower(currencyPair.ToSymbol("")))
	println(url)
	resp, err := fc.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Data []struct {
			Amount decimal.Decimal
			Price decimal.Decimal
			Id decimal.Decimal
			Ts int64
			Side string
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	var trades = make([]TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Tid = o.Id.IntPart()
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Side
		t.Date = o.Ts
	}

	return trades, nil
}

//交易符号
func (fc *FCoin) GetTradeSymbols() ([]TradeSymbol, error) {
	respmap, err := HttpGet(fc.httpClient, fc.baseUrl+"public/symbols")
	if err != nil {
		return nil, err
	}

	if respmap["status"].(float64) != 0 {
		return nil, errors.New(respmap["msg"].(string))
	}

	datamap := respmap["data"].([]interface{})

	tradeSymbols := make([]TradeSymbol, 0)
	for _, v := range datamap {
		vv := v.(map[string]interface{})
		var symbol TradeSymbol
		symbol.Name = vv["name"].(string)
		symbol.BaseCurrency = vv["base_currency"].(string)
		symbol.QuoteCurrency = vv["quote_currency"].(string)
		symbol.PriceDecimal = int(vv["price_decimal"].(float64))
		symbol.AmountDecimal = int(vv["amount_decimal"].(float64))
		symbol.Tradable = vv["tradable"].(bool)
		if symbol.Tradable {
			tradeSymbols = append(tradeSymbols, symbol)
		}
	}
	return tradeSymbols, nil
}

func (fc *FCoin) GetTradeSymbol(currencyPair CurrencyPair) (*TradeSymbol, error) {
	if len(fc.tradeSymbols) == 0 {
		var err error
		fc.tradeSymbols, err = fc.GetTradeSymbols()
		if err != nil {
			return nil, err
		}
	}
	for k, v := range fc.tradeSymbols {
		if v.Name == strings.ToLower(currencyPair.ToSymbol("")) {
			return &fc.tradeSymbols[k], nil
		}
	}
	return nil, errors.New("symbol not found")
}
