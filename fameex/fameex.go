package fameex

import (
	"net/http"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
	"sort"
	"net/url"
	"strconv"
	"sync"
	"log"
)

const (
	SIDE_BUY = 0
	SIDE_SELL = 1
)

const (
	HOST = "preapi.fameex.com"
	API_BASE_URL = "https://" + HOST
	SYMBOL = "/v1/common/symbols"
	TICKER = "/market/history/kline24h"
	DEPTH = "/market/depth"
	TRADE = "/market/history/trade"
	ACCOUNTS = "/api/account/v3/wallet"
	PLACE_ORDER = "/api/spot/v3/orders"
	BATCH_PLACE_ORDERS = "/api/spot/v3/orders_list"
	CANCEL_ORDER = "/api/spot/v3/cancel_orders"
	BATCH_CANCEL = "/api/spot/v3/cancel_orders_all_orders"
	OPEN_ORDERS = "/api/spot/v3/orderlist"
	QUERY_ORDER = "/api/spot/v3/orderdetail"
)

type Fameex struct {
	ApiKey string
	SecretKey string
	UserId string
	client *http.Client

	symbols map[string]*Symbol

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	wsOrderHandle  	func([]OrderDecimal)
	errorHandle      func(error)

	lock sync.Mutex
}

func NewFameex(client *http.Client, ApiKey string, SecretKey, userId string) *Fameex {
	this := new(Fameex)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.UserId = userId
	this.client = client

	return this
}

func (this *Fameex) ensureSymbols() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.symbols != nil {
		return nil
	}

	var symbols []Symbol
	var err error
	for i := 0; i < 5; i++ {
		symbols, err = this.GetSymbols()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return err
	}

	this.symbols = make(map[string]*Symbol)
	for i := range symbols {
		s := &symbols[i]
		this.symbols[s.Symbol] = s
	}
	return nil
}

func (this *Fameex) getPrecision(symbol string) (error, int) {
	err := this.ensureSymbols()
	if err != nil {
		return err, 0
	}
	s := this.symbols[symbol]
	if s == nil {
		return fmt.Errorf("unknow symbol %s", symbol), 0
	}
	return nil, s.PricePrecision
}

func (this *Fameex) signData(data string) string {
	sign, _ := GetParamHmacSHA256Sign(this.SecretKey, data)

	return sign
}

func (this *Fameex) sign(method, reqUrl string, param map[string]string) string {
	param["AccessKeyId"] = this.ApiKey
	param["SignatureMethod"] = "HmacSHA256"
	param["SignatureVersion"] = "v0.6"
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
	return data + "&Signature=" + url.QueryEscape(sign) + "&Timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
}

func (this *Fameex) buildQueryString(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k + "=" + url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

func (this *Fameex) GetSymbols() ([]Symbol, error) {
	params := map[string]string {}
	queryString := this.sign("GET", SYMBOL, params)

	url := API_BASE_URL + SYMBOL + "?" + queryString
	var resp struct {
		Code int
		Msg string
		Data []struct {
			Base string
			Quote string
			PricePercision decimal.Decimal
			AmountPercision decimal.Decimal
			PermitAmountPercision decimal.Decimal
		}
	}

	err := HttpGet4(this.client, url, nil, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		log.Printf("Fameex.GetSymbols error code: %d\n", resp.Code)
		return nil, fmt.Errorf("error_code: %d", resp.Code)
	}

	var ret []Symbol
	for _, r := range resp.Data {
		ret = append(ret, Symbol{
			BaseCurrency: r.Base,
			QuoteCurrency: r.Quote,
			PricePrecision: int(r.PricePercision.IntPart()),
			AmountPrecision: int(r.AmountPercision.IntPart()),
			MinAmount: r.PermitAmountPercision,
			Symbol: fmt.Sprintf("%s_%s", r.Base, r.Quote),
		})
	}

	return ret, nil
}

func (this *Fameex) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Fameex) GetTicker(symbol string) (*TickerDecimal, error) {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", TICKER, params)

	reqUrl := API_BASE_URL + TICKER + "?" + queryString
	postData := map[string]interface{} {
		"symbol": pair.ToSymbol("-"),
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)

	if err != nil {
		return nil, err
	}

	var resp struct {
		Code int
		Msg string
		CoinBase struct {
				 TransactionPrice decimal.Decimal
				 CoinHour24LowPrice decimal.Decimal
				 CoinHour24HighPrice decimal.Decimal
				 Hour24Volume decimal.Decimal
			 }
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil{
		return nil, err
	}

	if resp.Code != 200 {
		log.Printf("Fameex.GetTicker error code: %d\n", resp.Code)
		return nil, fmt.Errorf("error_code: %d", resp.Code)
	}

	r := &resp.CoinBase

	ticker := new(TickerDecimal)
	ticker.Date = uint64(time.Now().UnixNano()/1000000)
	ticker.Last = r.TransactionPrice
	ticker.High = r.CoinHour24HighPrice
	ticker.Low = r.CoinHour24LowPrice
	ticker.Vol = r.Hour24Volume

	return ticker, nil
}

func (this *Fameex) GetDepth(symbol string) (*DepthDecimal, error) {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", DEPTH, params)

	reqUrl := API_BASE_URL + DEPTH + "?" + queryString
	postData := map[string]interface{} {
		"base": pair.CurrencyA.Symbol,
		"quote": pair.CurrencyB.Symbol,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	type Item struct {
		Price decimal.Decimal
		Count decimal.Decimal
	}
	var data struct {
		Code int
		Data struct {
				 SellList []Item
				 BuyList  []Item
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Code != 200 {
		log.Printf("Fameex.GetDepth error code: %d\n", data.Code)
		return nil, fmt.Errorf("error_code: %d", data.Code)
	}

	r := data.Data

	depth := new(DepthDecimal)
	depth.Pair = pair

	depth.AskList = make([]DepthRecordDecimal, len(r.SellList), len(r.SellList))
	for i, o := range r.SellList {
		depth.AskList[i] = DepthRecordDecimal{Price: o.Price, Amount: o.Count}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.BuyList), len(r.BuyList))
	for i, o := range r.BuyList {
		depth.BidList[i] = DepthRecordDecimal{Price: o.Price, Amount: o.Count}
	}

	return depth, nil
}

func (this *Fameex) GetTrades(symbol string) ([]TradeDecimal, error) {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", TRADE, params)

	reqUrl := API_BASE_URL + TRADE + "?" + queryString
	postData := map[string]interface{} {
		"base": pair.CurrencyA.Symbol,
		"quote": pair.CurrencyB.Symbol,
		"num": "1",
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var data struct {
		Code int
		Data [] struct {
			Count decimal.Decimal
			Price decimal.Decimal
			Time int64
			BuyType int
		}
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Code != 200 {
		log.Printf("Fameex.GetTrades error code: %d\n", data.Code)
		return nil, fmt.Errorf("error_code: %d", data.Code)
	}

	var trades = make([]TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Amount = o.Count
		t.Price = o.Price
		if o.BuyType == 0 {
			t.Type = "buy"
		} else {
			t.Type = "sell"
		}
		t.Date = o.Time / 1000000
	}

	return trades, nil
}

func (this *Fameex) GetAccounts() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	queryString := this.sign("GET", ACCOUNTS, params)

	url := API_BASE_URL + ACCOUNTS + "?" + queryString
	var resp struct {
		Code int
		Data []struct {
			List []struct {
					 Available decimal.Decimal
					 Total decimal.Decimal
					 Hold decimal.Decimal
					 Currency string
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

	if resp.Code != 200 {
		return nil, fmt.Errorf("error_code: %d\n", resp.Code)
	}

	if len(resp.Data) == 0 {
		return nil, nil
	}

	var ret []SubAccountDecimal
	for _, r := range resp.Data[0].List {
		ret = append(ret, SubAccountDecimal{
			Currency: NewCurrency(r.Currency, ""),
			AvailableAmount: r.Available,
			Amount: r.Total,
			FrozenAmount: r.Hold,
		})
	}

	return ret, nil
}

func (this *Fameex) PlaceOrder(symbol string, side int, price, volume decimal.Decimal) (string, error) {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", PLACE_ORDER, params)

	reqUrl := API_BASE_URL + PLACE_ORDER + "?" + queryString
	postData := map[string]interface{} {
		"coin1": pair.CurrencyA.Symbol,
		"coin2": pair.CurrencyB.Symbol,
		"buytype": side,
		"price": price.String(),
		"count": volume.String(),
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}

	var data struct {
		Code int
		TaskId string
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return "", err
	}

	if data.Code != 200 {
		log.Printf("Fameex.PlaceOrder error code: %d\n", data.Code)
		return "", fmt.Errorf("error_code: %d", data.Code)
	}

	return data.TaskId, nil
}

func (this *Fameex) PlaceOrders(symbol string, reqList []OrderReq) ([]string, []error, error) {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", BATCH_PLACE_ORDERS, params)

	reqUrl := API_BASE_URL + BATCH_PLACE_ORDERS + "?" + queryString
	postData := map[string]interface{} {
		"coin1": pair.CurrencyA.Symbol,
		"coin2": pair.CurrencyB.Symbol,
		"orders": reqList,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, nil, err
	}

	if err != nil {
		return nil, nil, err
	}

	var data struct {
		Code int
		Data []struct {
			OrderId   string
			OrderCode int
		}
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, nil, err
	}

	if data.Code != 200 {
		log.Printf("Fameex.PlaceOrders error code: %d\n", data.Code)
		return nil, nil, fmt.Errorf("error_code: %d", data.Code)
	}

	var orderIds = make([]string, len(reqList))
	var errorList = make([]error, len(reqList))
	for i, r := range data.Data {
		if r.OrderCode == 200 {
			orderIds[i] = r.OrderId
		} else {
			log.Printf("Fameex.PlaceOrders error code: %d\n", r.OrderCode)
			errorList[i] = fmt.Errorf("error_code: %d", r.OrderCode)
		}
	}

	return orderIds, errorList, nil
}

func (this *Fameex) CancelOrder(symbol string, orderId string) error {
	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", CANCEL_ORDER, params)

	reqUrl := API_BASE_URL + CANCEL_ORDER + "?" + queryString
	postData := map[string]interface{} {
		"coin1": pair.CurrencyA.Symbol,
		"coin2": pair.CurrencyB.Symbol,
		"orderid": orderId,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	var data struct {
		Code int
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}

	switch data.Code {
	case 200, 201:
		return nil
	case 2404:
		return nil			// 找不到订单，当成成功
	default:
		log.Printf("Fameex.CancelOrder error code: %d\n", data.Code)
		return fmt.Errorf("error_code: %d", data.Code)
	}
}

func (this *Fameex) BatchCancelOrders(symbol string, orderIds []string) (error, []error) {
	var errorList =  make([]error, len(orderIds))

	pair := NewCurrencyPair2(symbol)
	params := map[string]string {}
	queryString := this.sign("POST", BATCH_CANCEL, params)

	reqUrl := API_BASE_URL + BATCH_CANCEL + "?" + queryString
	postData := map[string]interface{} {
		"coin1": pair.CurrencyA.Symbol,
		"coin2": pair.CurrencyB.Symbol,
		"orderIds": orderIds,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return err, errorList
	}

	if err != nil {
		return err, errorList
	}

	var data struct {
		Code int
		Data []struct {
			OrderId   string
			OrderCode int
		}
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err, errorList
	}

	for i, r := range data.Data {
		if r.OrderCode != 200 && r.OrderCode != 21010 {
			log.Printf("Fameex.BatchCancelOrders error code: %d\n", r.OrderCode)
			errorList[i] = fmt.Errorf("error_code: %d", r.OrderCode)
		}
	}

	return nil, errorList
}

func (this *Fameex) QueryPendingOrders(symbol string, page, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 100
	}

	params := map[string]string {}
	queryString := this.sign("POST", OPEN_ORDERS, params)

	parts := strings.Split(symbol, "_")

	reqUrl := API_BASE_URL + OPEN_ORDERS+ "?" + queryString
	postData := map[string]interface{} {
		"type": "2",
		"buyClass": "-1",
		"direction": "-1",
		"coin1": parts[0],
		"coin2": parts[1],
		"pageNum": page,
		"pageSize": pageSize,
		"startTime": "0",
		"endTime": "0",
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var data struct {
		Code int
		Data struct {
			List []OrderInfo
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Code != 200 {
		log.Printf("Fameex.QueryPendingOrders error code: %d", data.Code)
		return nil, fmt.Errorf("error_code: %d", data.Code)
	}

	var ret = make([]OrderDecimal, len(data.Data.List))
	for i := range data.Data.List {
		ret[i] = *data.Data.List[i].ToOrderDecimal()
	}

	return ret, nil
}

func (this *Fameex) QueryOrder(orderId string) (*OrderDecimal, error) {
	params := map[string]string {}
	queryString := this.sign("POST", QUERY_ORDER, params)

	reqUrl := API_BASE_URL + QUERY_ORDER + "?" + queryString
	postData := map[string]interface{} {
		"orderid": orderId,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		Code int
		Data *OrderInfo
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Code != 200 {
		log.Printf("Fameex.QueryOrder error code: %d\n", data.Code)
		return nil, fmt.Errorf("error_code: %d", data.Code)
	}

	if data.Data.OrderId == "" {
		return nil, nil
	}

	return data.Data.ToOrderDecimal(), nil
}
