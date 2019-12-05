package deerdex

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	. "github.com/stephenlyu/GoEx"
	"sync"
	"strconv"
	"net/url"
)

const (
	ORDER_SELL = "SELL"
	ORDER_BUY = "BUY"

	ORDER_TYPE_LIMIT = 1
	ORDER_TYPE_MARKET = 2

	OrderTypeBuyMarket = "buy-market"
	OrderTypeBuyLimit = "buy-limit"
	OrderTypeSellMarket = "sell-market"
	OrderTypeSellLimit = "sell-limit"
)

const (
	Host = "api.deerdex.com"
	API_BASE = "https://" + Host
	COMMON_SYMBOLS = "/openapi/v1/brokerInfo"
	GET_TICKER = "/openapi/quote/v1/ticker/24hr?symbol=%s"
	GET_MARKET_DEPH = "/openapi/quote/v1/depth?symbol=%s&limit=20"
	GET_TRADES = "/openapi/quote/v1/trades?symbol=%s&limit=1"
	ACCOUNT = "/openapi/v1/account"
	CREATE_ORDER = "/openapi/v1/order"
	CANCEL_ORDER = "/openapi/v1/order"
	NEW_ORDER = "/openapi/v1/openOrders"
	ORDER_INFO = "/openapi/v1/order"
	His_ORDER = "/openapi/v1/historyOrders"
)

type DeerDex struct {
	ApiKey    string
	SecretKey string
	client    *http.Client

	symbolNameMap map[string]string

	publicWs           *WsConn
	createPublicWsLock sync.Mutex
	wsDepthHandleMap   map[string]func(*DepthDecimal)
	wsTradeHandleMap   map[string]func(string, []TradeDecimal)
	errorHandle        func(error)
}

func NewDeerDex(ApiKey string, SecretKey string) *DeerDex {
	this := new(DeerDex)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *DeerDex) getPairByName(name string) string {
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
		key := fmt.Sprintf("%s%s", o.BaseAsset, o.QuoteAsset)
		this.symbolNameMap[key] = o.Symbol
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (ok *DeerDex) GetSymbols() ([]Symbol, error) {
	url := API_BASE + COMMON_SYMBOLS
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
		Msg string
		Code decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	var ret []Symbol
	for _, s := range data.Symbols {
		s.Symbol = strings.ToUpper(fmt.Sprintf("%s_%s", s.BaseAsset, s.QuoteAsset))
		for _, f := range s.Filters {
			if f.FilterType == "LOT_SIZE" {
				s.AmountMin = f.MinQty
				break
			}
		}
		ret = append(ret, s)
	}

	return ret, nil
}

func (this *DeerDex) transSymbol(symbol string) string {
	return strings.ToUpper(strings.Replace(symbol, "_", "", -1))
}

func (this *DeerDex) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE + GET_TICKER, symbol)
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
		Msg          string
		Code         decimal.Decimal

		Time         int64
		BestBidPrice decimal.Decimal
		BestAskPrice decimal.Decimal
		LastPrice    decimal.Decimal
		OpenPrice    decimal.Decimal
		HighPrice    decimal.Decimal
		LowPrice     decimal.Decimal
		Volume       decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	r := data

	ticker := new(TickerDecimal)
	ticker.Date = uint64(r.Time)
	ticker.Open = r.OpenPrice
	ticker.Last = r.LastPrice
	ticker.High = r.HighPrice
	ticker.Low = r.LowPrice
	ticker.Vol = r.Volume
	ticker.Buy = r.BestBidPrice
	ticker.Sell = r.BestAskPrice

	return ticker, nil
}

func (this *DeerDex) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE + GET_MARKET_DEPH, symbol)
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

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
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

func (this *DeerDex) GetTrades(symbol string) ([]TradeDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE + GET_TRADES, symbol)
	resp, err := this.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	if bodyStr[0] == '{' {
		var data struct {
			Msg string
			Code decimal.Decimal
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("error code: %s", data.Code)
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

func (this *DeerDex) signData(data string) string {
	sign, _ := GetParamHmacSHA256Sign(this.SecretKey, data)

	return sign
}

func (this *DeerDex) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["timestamp"] = timestamp

	var parts []string
	for k, v := range param {
		parts = append(parts, k + "=" + v)
	}
	data := strings.Join(parts, "&")
	println(data)
	sign := this.signData(data)
	param["signature"] = sign
	return param
}

func (this *DeerDex) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

func (this *DeerDex) authHeader() map[string]string {
	return map[string]string {
		"X-BH-APIKEY": this.ApiKey,
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *DeerDex) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	params = this.sign(params)

	url := API_BASE + ACCOUNT + "?" + this.buildQueryString(params)
	var resp struct {
		Msg string
		Code decimal.Decimal
		Balances []struct {
			Asset string
			Free decimal.Decimal
			Locked decimal.Decimal
		}
	}

	err := HttpGet4(this.client, url, this.authHeader(), &resp)

	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code)
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

//func (this *DeerDex) PlaceOrder(volume decimal.Decimal, side string, _type int, symbol string, price decimal.Decimal) (string, error) {
//	symbol = this.transSymbol(symbol)
//	signParams := map[string]string {
//		"side": side,
//		"volume": volume.String(),
//		"type": strconv.Itoa(_type),
//		"symbol": symbol,
//		"price": price.String(),
//	}
//	signParams = this.sign(signParams)
//
//	queryParams := map[string]string {
//		"api_key": this.ApiKey,
//		"time": signParams["time"],
//		"sign": signParams["sign"],
//	}
//
//	delete(signParams, "api_key")
//	delete(signParams, "sign")
//
//	queryString := this.buildQueryString(queryParams)
//	url := Trading_Macro_v2 + CREATE_ORDER + "?" + queryString
//
//	postData := this.buildQueryString(signParams)
//
//	body, err := HttpPostForm3(this.client, url, postData, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
//
//	if err != nil {
//		return "", err
//	}
//
//	var resp struct {
//		Msg string
//		Code decimal.Decimal
//		Data struct {
//		   OrderId decimal.Decimal		`json:"order_id"`
//	   }
//	}
//
//	err = json.Unmarshal(body, &resp)
//	if err != nil {
//		return "", err
//	}
//
//	if !resp.Code.IsZero() {
//		return "", fmt.Errorf("error code: %s", resp.Code)
//	}
//
//	return resp.Data.OrderId.String(), nil
//}
//
//func (this *DeerDex) CancelOrder(symbol string, orderIds []string) (error, []error) {
//	var errors = make([]error, len(orderIds))
//
//	symbol = this.transSymbol(symbol)
//	bytes, _ := json.Marshal(map[string][]string{symbol: orderIds})
//	signParams := map[string]string {
//		"orderIdList": string(bytes),
//
//	}
//	signParams = this.sign(signParams)
//
//	queryParams := map[string]string {
//		"api_key": this.ApiKey,
//		"time": signParams["time"],
//		"sign": signParams["sign"],
//	}
//
//	delete(signParams, "api_key")
//	delete(signParams, "sign")
//
//	queryString := this.buildQueryString(queryParams)
//
//	data := this.buildQueryString(signParams)
//	url := Trading_Macro_v2 + CANCEL_ORDER + "?" + queryString
//
//	body, err := HttpPostForm3(this.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
//
//	if err != nil {
//		return err, errors
//	}
//
//	var resp struct {
//		Msg string
//		Code decimal.Decimal
//		Data struct {
//			Success []decimal.Decimal
//			Failed []struct {
//				OrderId decimal.Decimal `json:"order-id"`
//				ErrCode decimal.Decimal	`json:"err-code"`
//			}
//		}
//	}
//
//	err = json.Unmarshal(body, &resp)
//	if err != nil {
//		return err, errors
//	}
//
//	if !resp.Code.IsZero() {
//		return fmt.Errorf("error code: %s", resp.Code), errors
//	}
//
//	var m = make(map[string]int)
//	for i, orderId := range orderIds {
//		m[orderId] = i
//	}
//
//	for _, r := range resp.Data.Failed {
//		index := m[r.OrderId.String()]
//		errors[index] = fmt.Errorf("error_code: %s", r.ErrCode)
//	}
//
//	return nil, errors
//}
//
//func (this *DeerDex) QueryPendingOrders(symbol string, from string, pageSize int) ([]OrderDecimal, error) {
//	if pageSize == 0 || pageSize > 50 {
//		pageSize = 50
//	}
//
//	param := map[string]string {
//		"symbol": this.transSymbol(symbol),
//		"states": "new,part_filled",
//		"direct": "prev",
//	}
//	if from != "" {
//		param["from"] = from
//	}
//	if pageSize > 0 {
//		param["size"] = strconv.Itoa(pageSize)
//	}
//	param = this.sign(param)
//	url := Trading_Macro_v2 + NEW_ORDER + "?" + this.buildQueryString(param)
//	var resp struct {
//	    Msg string
//	    Code decimal.Decimal
//		Data []OrderInfo
//	}
//
//	err := HttpGet4(this.client, url, nil, &resp)
//	if err != nil {
//		return nil, err
//	}
//
//	if !resp.Code.IsZero() {
//		return nil, fmt.Errorf("error code: %s", resp.Code)
//	}
//
//	var ret = make([]OrderDecimal, len(resp.Data))
//	for i := range resp.Data {
//		ret[i] = *resp.Data[i].ToOrderDecimal(symbol)
//	}
//
//	return ret, nil
//}
//
//func (this *DeerDex) QueryOrder(symbol string, orderId string) (*OrderDecimal, error) {
//	symbol = strings.ToUpper(symbol)
//	param := this.sign(map[string]string {
//		"symbol": this.transSymbol(symbol),
//		"order_id": orderId,
//	})
//
//	url := fmt.Sprintf(Trading_Macro_v2 + ORDER_INFO + "?" + this.buildQueryString(param))
//
//	var resp struct {
//	    Msg string
//	    Code decimal.Decimal
//		Data *OrderInfo
//	}
//
//	err := HttpGet4(this.client, url, nil, &resp)
//	if err != nil {
//		return nil, err
//	}
//
//	if !resp.Code.IsZero() {
//		return nil, fmt.Errorf("error code: %s", resp.Code)
//	}
//
//	if resp.Data == nil || resp.Data.Id.IsZero() {
//		return nil, nil
//	}
//
//	return resp.Data.ToOrderDecimal(symbol), nil
//}
