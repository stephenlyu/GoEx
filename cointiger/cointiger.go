package cointiger

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
	"sort"
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
	Host = "api.cointiger.com"
	Trading_Macro = "https://" + Host + "/exchange/trading"
	Trading_Macro_v2 = "https://" + Host + "/exchange/trading/api/v2"
	Market_Macro = "https://" + Host + "/exchange/trading/api"
	COMMON_SYMBOLS = "/currencys"
	GET_TICKER = "/market/detail?symbol=%s"
	GET_MARKET_DEPH = "/market/depth?symbol=%s&type=step0"
	GET_TRADES = "/market/history/trade?symbol=%s&size=1"
	ACCOUNT = "/api/user/balance"
	CREATE_ORDER = "/order"
	CANCEL_ORDER = "/order/batch_cancel"
	NEW_ORDER = "/order/current"
	ORDER_INFO = "/order/details"
	ALL_ORDER = "/order/history"
)

type CoinTiger struct {
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

func NewCoinTiger(ApiKey string, SecretKey string) *CoinTiger {
	this := new(CoinTiger)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *CoinTiger) standardHeader() map[string]string {
	return map[string]string {
		"Language": "zh_CN",
		"User-Agent": "Mozilla/5.0(Macintosh;U;IntelMacOSX10_6_8;en-us)AppleWebKit/534.50(KHTML,likeGecko)Version/5.1Safari/534.50",
		"Referer": "https://" + Host,
	}
}

func (this *CoinTiger) getPairByName(name string) string {
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
		key := fmt.Sprintf("%s%s", o.BaseCurrency, o.QuoteCurrency)
		this.symbolNameMap[key] = o.Symbol
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (ok *CoinTiger) GetSymbols() ([]Symbol, error) {
	url := Trading_Macro_v2 + COMMON_SYMBOLS
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
		Data map[string][]Symbol
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
	for _, l := range data.Data {
		for _, s := range l {
			s.Symbol = strings.ToUpper(fmt.Sprintf("%s_%s", s.BaseCurrency, s.QuoteCurrency))
			ret = append(ret, s)
		}
	}

	return ret, nil
}

func (this *CoinTiger) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *CoinTiger) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := Market_Macro + GET_TICKER
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
		Msg string
		Code decimal.Decimal
		Data struct {
			Data struct {
				Tick struct {
					Amount decimal.Decimal
					Vol decimal.Decimal
					High decimal.Decimal
					Low decimal.Decimal
					Close decimal.Decimal
					Open decimal.Decimal

				}
				Ts int64
			} 	`json:"trade_ticker_data"`
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	r := data.Data.Data.Tick

	ticker := new(TickerDecimal)
	ticker.Date = uint64(data.Data.Data.Ts)
	ticker.Open = r.Open
	ticker.Last = r.Close
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol

	return ticker, nil
}

func (this *CoinTiger) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(Market_Macro + GET_MARKET_DEPH, symbol)
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
	    Data struct {
            DepthData struct {
				Tick struct {
					Asks [][]decimal.Decimal
					Buys [][]decimal.Decimal
				}
				Ts int64
			} `json:"depth_data"`
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	r := data.Data.DepthData.Tick

	depth := new(DepthDecimal)
	depth.Pair = NewCurrencyPair2(inputSymbol)

	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Buys), len(r.Buys))
	for i, o := range r.Buys {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

func (this *CoinTiger) GetTrades(symbol string) ([]TradeDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(Market_Macro + GET_TRADES, symbol)
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
		Data struct {
			TradeData []struct {
				Id int64
				Side string
				Price decimal.Decimal
				Vol decimal.Decimal
				Ts int64
			}	`json:"trade_data"`
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if !data.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", data.Code)
	}

	var trades = make([]TradeDecimal, len(data.Data.TradeData))

	for i, o := range data.Data.TradeData {
		t := &trades[i]
		t.Tid = o.Id
		t.Amount = o.Vol
		t.Price = o.Price
		t.Type = strings.ToLower(o.Side)
		t.Date = o.Ts
	}

	return trades, nil
}

func (this *CoinTiger) signData(data string) string {
	message := data + this.SecretKey
	sign, _ := GetParamHmacSHA512Sign(this.SecretKey, message)

	return sign
}

func (this *CoinTiger) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["time"] = timestamp

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

	sign := this.signData(data)
	param["api_key"] = this.ApiKey
	param["sign"] = sign
	return param
}

func (this *CoinTiger) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

func (this *CoinTiger) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	params = this.sign(params)

	url := Trading_Macro + ACCOUNT + "?" + this.buildQueryString(params)
	var resp struct {
		Msg string
		Code decimal.Decimal
		Data []struct {
			Normal decimal.Decimal
			Lock decimal.Decimal
			Coin string
		}
	}

	err := HttpGet4(this.client, url, map[string]string{}, &resp)

	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code)
	}

	var ret []SubAccountDecimal
	for _, o := range resp.Data {
		currency := strings.ToUpper(o.Coin)
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Normal,
			FrozenAmount: o.Lock,
			Amount: o.Normal.Add(o.Lock),
		})
	}

	return ret, nil
}

func (this *CoinTiger) PlaceOrder(volume decimal.Decimal, side string, _type int, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	signParams := map[string]string {
		"side": side,
		"volume": volume.String(),
		"type": strconv.Itoa(_type),
		"symbol": symbol,
		"price": price.String(),
	}
	signParams = this.sign(signParams)

	queryParams := map[string]string {
		"api_key": this.ApiKey,
		"time": signParams["time"],
		"sign": signParams["sign"],
	}

	delete(signParams, "api_key")
	delete(signParams, "sign")

	queryString := this.buildQueryString(queryParams)
	url := Trading_Macro_v2 + CREATE_ORDER + "?" + queryString

	postData := this.buildQueryString(signParams)

	body, err := HttpPostForm3(this.client, url, postData, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return "", err
	}

	var resp struct {
		Msg string
		Code decimal.Decimal
		Data struct {
		   OrderId decimal.Decimal		`json:"order_id"`
	   }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if !resp.Code.IsZero() {
		return "", fmt.Errorf("error code: %s", resp.Code)
	}

	return resp.Data.OrderId.String(), nil
}

func (this *CoinTiger) CancelOrder(symbol string, orderIds []string) (error, []error) {
	var errors = make([]error, len(orderIds))

	symbol = this.transSymbol(symbol)
	bytes, _ := json.Marshal(map[string][]string{symbol: orderIds})
	signParams := map[string]string {
		"orderIdList": string(bytes),

	}
	signParams = this.sign(signParams)

	queryParams := map[string]string {
		"api_key": this.ApiKey,
		"time": signParams["time"],
		"sign": signParams["sign"],
	}

	delete(signParams, "api_key")
	delete(signParams, "sign")

	queryString := this.buildQueryString(queryParams)

	data := this.buildQueryString(signParams)
	url := Trading_Macro_v2 + CANCEL_ORDER + "?" + queryString

	body, err := HttpPostForm3(this.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return err, errors
	}

	var resp struct {
		Msg string
		Code decimal.Decimal
		Data struct {
			Success []decimal.Decimal
			Failed []struct {
				OrderId decimal.Decimal `json:"order-id"`
				ErrCode decimal.Decimal	`json:"err-code"`
			}
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err, errors
	}

	if !resp.Code.IsZero() {
		return fmt.Errorf("error code: %s", resp.Code), errors
	}

	var m = make(map[string]int)
	for i, orderId := range orderIds {
		m[orderId] = i
	}

	for _, r := range resp.Data.Failed {
		index := m[r.OrderId.String()]
		errors[index] = fmt.Errorf("error_code: %s", r.ErrCode)
	}

	return nil, errors
}

func (this *CoinTiger) QueryPendingOrders(symbol string, from string, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 || pageSize > 50 {
		pageSize = 50
	}

	param := map[string]string {
		"symbol": this.transSymbol(symbol),
		"states": "new,part_filled",
		"direct": "prev",
	}
	if from != "" {
		param["from"] = from
	}
	if pageSize > 0 {
		param["size"] = strconv.Itoa(pageSize)
	}
	param = this.sign(param)
	url := Trading_Macro_v2 + NEW_ORDER + "?" + this.buildQueryString(param)
	var resp struct {
	    Msg string
	    Code decimal.Decimal
		Data []OrderInfo
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code)
	}

	var ret = make([]OrderDecimal, len(resp.Data))
	for i := range resp.Data {
		ret[i] = *resp.Data[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *CoinTiger) QueryOrder(symbol string, orderId string) (*OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	param := this.sign(map[string]string {
		"symbol": this.transSymbol(symbol),
		"order_id": orderId,
	})

	url := fmt.Sprintf(Trading_Macro_v2 + ORDER_INFO + "?" + this.buildQueryString(param))

	var resp struct {
	    Msg string
	    Code decimal.Decimal
		Data *OrderInfo
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Code.IsZero() {
		return nil, fmt.Errorf("error code: %s", resp.Code)
	}

	if resp.Data == nil || resp.Data.Id.IsZero() {
		return nil, nil
	}

	return resp.Data.ToOrderDecimal(symbol), nil
}
