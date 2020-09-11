package deerdex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
)

const (
	ORDER_SELL = "SELL"
	ORDER_BUY  = "BUY"

	ORDER_TYPE_LIMIT  = "LIMIT"
	ORDER_TYPE_MARKET = "MARKET"
)

const (
	Host            = "api.deerdex.com"
	API_BASE        = "https://" + Host
	COMMON_SYMBOLS  = "/openapi/v1/brokerInfo"
	GET_TICKER      = "/openapi/quote/v1/ticker/24hr?symbol=%s"
	GET_MARKET_DEPH = "/openapi/quote/v1/depth?symbol=%s&limit=20"
	GET_TRADES      = "/openapi/quote/v1/trades?symbol=%s&limit=1"
	ACCOUNT         = "/openapi/v1/account"
	CREATE_ORDER    = "/openapi/v1/order"
	CANCEL_ORDER    = "/openapi/v1/order"
	NEW_ORDER       = "/openapi/v1/openOrders"
	ORDER_INFO      = "/openapi/v1/order"
	His_ORDER       = "/openapi/v1/historyOrders"
	MY_TRADES       = "/openapi/v1/myTrades"

	USER_DATA_STREAM = "/openapi/v1/userDataStream"
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
		Msg     string
		Code    decimal.Decimal
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
	url := fmt.Sprintf(API_BASE+GET_TICKER, symbol)
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
	url := fmt.Sprintf(API_BASE+GET_MARKET_DEPH, symbol)
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
	url := fmt.Sprintf(API_BASE+GET_TRADES, symbol)
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
			Msg  string
			Code decimal.Decimal
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	var data []struct {
		Price        decimal.Decimal
		Qty          decimal.Decimal
		Time         int64
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

func (this *DeerDex) sign(param map[string]string) string {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	param["timestamp"] = timestamp

	var parts []string
	for k, v := range param {
		parts = append(parts, k+"="+v)
	}
	data := strings.Join(parts, "&")
	sign := this.signData(data)
	return data + "&signature=" + sign
}

func (this *DeerDex) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

func (this *DeerDex) authHeader() map[string]string {
	return map[string]string{
		"X-BH-APIKEY":  this.ApiKey,
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *DeerDex) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string{}
	queryString := this.sign(params)

	url := API_BASE + ACCOUNT + "?" + queryString
	var resp struct {
		Msg      string
		Code     decimal.Decimal
		Balances []struct {
			Asset  string
			Free   decimal.Decimal
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
			Currency:        Currency{Symbol: currency},
			AvailableAmount: o.Free,
			FrozenAmount:    o.Locked,
			Amount:          o.Free.Add(o.Locked),
		})
	}

	return ret, nil
}

func (this *DeerDex) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	signParams := map[string]string{
		"side":        side,
		"quantity":    volume.String(),
		"type":        _type,
		"symbol":      symbol,
		"price":       price.String(),
		"timeInForce": "GTC",
	}
	postData := this.sign(signParams)
	println(postData)
	url := API_BASE + CREATE_ORDER

	body, err := HttpPostForm3(this.client, url, postData, this.authHeader())

	if err != nil {
		return "", err
	}
	println(string(body))
	var resp struct {
		Msg     string
		Code    decimal.Decimal
		OrderId decimal.Decimal `json:"orderId"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if !resp.Code.IsZero() {
		return "", fmt.Errorf("error code: %s", resp.Code)
	}

	return resp.OrderId.String(), nil
}

func (this *DeerDex) CancelOrder(orderId string) error {
	signParams := map[string]string{
		"orderId": orderId,
	}
	postData := this.sign(signParams)
	url := API_BASE + CANCEL_ORDER + "?" + postData
	body, err := HttpDeleteForm3(this.client, url, "", this.authHeader())

	if err != nil {
		if strings.Contains(err.Error(), "-1142") {
			return nil
		}
		if strings.Contains(err.Error(), "-1139") {
			return nil
		}
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

	if !resp.Code.IsZero() {
		return fmt.Errorf("error code: %s", resp.Code)
	}

	return nil
}

func (this *DeerDex) QueryPendingOrders(symbol string, from string, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 50
	}

	param := map[string]string{
		"symbol": this.transSymbol(symbol),
	}
	if from != "" {
		param["orderId"] = from
	}
	if pageSize > 0 {
		param["limit"] = strconv.Itoa(pageSize)
	}
	queryStr := this.sign(param)
	url := API_BASE + NEW_ORDER + "?" + queryStr

	bytes, err := HttpGet6(this.client, url, this.authHeader())
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(bytes), "{") {
		var resp struct {
			Msg  string
			Code decimal.Decimal
		}

		err = json.Unmarshal(bytes, &resp)
		if err != nil {
			return nil, err
		}

		if !resp.Code.IsZero() {
			return nil, fmt.Errorf("error code: %s", resp.Code)
		}
	}

	var l []OrderInfo
	err = json.Unmarshal(bytes, &l)
	if err != nil {
		return nil, err
	}

	var ret = make([]OrderDecimal, len(l))
	for i := range l {
		ret[i] = *l[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *DeerDex) QueryHisOrders(symbol string, from string, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 50
	}

	param := map[string]string{
		"symbol": this.transSymbol(symbol),
	}
	if from != "" {
		param["orderId"] = from
	}
	if pageSize > 0 {
		param["limit"] = strconv.Itoa(pageSize)
	}
	queryStr := this.sign(param)
	url := API_BASE + His_ORDER + "?" + queryStr

	bytes, err := HttpGet6(this.client, url, this.authHeader())
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(bytes), "{") {
		var resp struct {
			Msg  string
			Code decimal.Decimal
		}

		err = json.Unmarshal(bytes, &resp)
		if err != nil {
			return nil, err
		}

		if !resp.Code.IsZero() {
			return nil, fmt.Errorf("error code: %s", resp.Code)
		}
	}

	var l []OrderInfo
	err = json.Unmarshal(bytes, &l)
	if err != nil {
		return nil, err
	}

	var ret = make([]OrderDecimal, len(l))
	for i := range l {
		ret[i] = *l[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *DeerDex) QueryOrder(symbol string, orderId string) (*OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	queryStr := this.sign(map[string]string{
		"orderId": orderId,
	})

	url := API_BASE + ORDER_INFO + "?" + queryStr

	var resp *OrderInfo

	err := HttpGet4(this.client, url, this.authHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code)
	}

	if resp == nil || resp.OrderId.IsZero() {
		return nil, nil
	}

	return resp.ToOrderDecimal(symbol), nil
}

func (this *DeerDex) QueryFills(startTime, endTime int64, fromId int64, pageSize int) ([]Fill, error) {
	if pageSize == 0 {
		pageSize = 50
	}

	param := map[string]string{}
	if startTime > 0 {
		param["startTime"] = strconv.FormatInt(startTime, 10)
	}
	if endTime > 0 {
		param["endTime"] = strconv.FormatInt(endTime, 10)
	}
	if fromId > 0 {
		param["fromId"] = strconv.FormatInt(fromId, 10)
	}
	if pageSize > 0 {
		param["limit"] = strconv.Itoa(pageSize)
	}
	queryStr := this.sign(param)
	url := API_BASE + MY_TRADES + "?" + queryStr

	bytes, err := HttpGet6(this.client, url, this.authHeader())
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(bytes), "{") {
		var resp struct {
			Msg  string
			Code decimal.Decimal
		}

		err = json.Unmarshal(bytes, &resp)
		if err != nil {
			return nil, err
		}

		if !resp.Code.IsZero() {
			return nil, fmt.Errorf("error code: %s", resp.Code)
		}
	}

	var l []Fill
	err = json.Unmarshal(bytes, &l)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// User Stream Helpers

func (this *DeerDex) CreateListenKey() (string, error) {
	signParams := map[string]string{}
	postData := this.sign(signParams)

	url := API_BASE + USER_DATA_STREAM

	body, err := HttpPostForm3(this.client, url, postData, this.authHeader())

	if err != nil {
		return "", err
	}

	var resp struct {
		Msg       string
		Code      decimal.Decimal
		ListenKey string
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if !resp.Code.IsZero() {
		return "", fmt.Errorf("error code: %s", resp.Code)
	}

	return resp.ListenKey, nil
}

func (this *DeerDex) ListenKeyKeepAlive(listenKey string) error {
	signParams := map[string]string{
		"listenKey": listenKey,
	}
	postData := this.sign(signParams)

	url := API_BASE + USER_DATA_STREAM
	body, err := HttpPutForm3(this.client, url, postData, this.authHeader())

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

	if !resp.Code.IsZero() {
		return fmt.Errorf("error code: %s", resp.Code)
	}

	return nil
}
