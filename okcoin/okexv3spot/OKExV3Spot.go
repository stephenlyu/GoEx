package okexv3spot

import (
	. "github.com/stephenlyu/GoEx"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"fmt"
	"strings"
	"errors"
	"sync"
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/TdxProtocol/util"
)

const (
	SPOT_V3_API_BASE_URL    = "https://www.okex.com"
	SPOT_V3_INSTRUMENTS 	  = "/api/spot/v3/instruments"
	SPOT_V3_TRADES 			= "/api/spot/v3/instruments/%s/trades"
	SPOT_V3_ACCOUNTS 		  = "/api/spot/v3/accounts"
	SPOT_V3_CURRENCY_ACCOUNTS = "/api/spot/v3/accounts/%s"
	SPOT_V3_INSTRUMENT_TICKER = "/api/spot/v3/instruments/%s/ticker"
	SPOT_V3_ORDERS 		   = "/api/spot/v3/orders"
	SPOT_V3_BATCH_ORDERS 		  = "/api/spot/v3/batch_orders"
	SPOT_V3_CANCEL_ORDERS    = "/api/spot/v3/cancel_batch_orders"
	SPOT_V3_CANCEL_ORDER		= "/api/spot/v3/cancel_orders/%s"
	SPOT_V3_INSTRUMENT_ORDERS = "/api/spot/v3/orders?instrument_id=%s"
	SPOT_V3_INSTRUMENT_ORDERS_PENDING = "/api/spot/v3/orders_pending?instrument_id=%s"
	SPOT_V3_ORDER_INFO 		= "/api/spot/v3/orders/%s?instrument_id=%s"
)

const (
	V3_DATE_FORMAT = "2006-01-02T15:04:05.000Z"
)

const (
	V3_ORDER_TYPE_NORMAL = 0
	V3_ORDER_TYPE_POST_ONLY = 1
	V3_ORDER_TYPE_FOK = 2
	V3_ORDER_TYPE_IOC = 3
)

type V3Instrument struct {
	InstrumentId string 			`json:"instrument_id"`
	BaseCurrency string 			`json:"base_currency"`
	QuoteCurrency string			`json:"quote_currency"`
	MinSize decimal.Decimal			`json:"min_size"`
	SizeIncrement decimal.Decimal 	`json:"size_increment"`
	TickSize decimal.Decimal 		`json:"tick_size"`
}

func V3ParseDate(s string) int64 {
	t, _ := time.ParseInLocation(V3_DATE_FORMAT, s, time.UTC)
	return t.UnixNano() / int64(time.Millisecond)
}

func InstrumentId2CurrencyPair(instrumentId string) CurrencyPair {
	parts := strings.Split(instrumentId, "-")
	return CurrencyPair{
		Currency{Symbol: parts[0]},
		Currency{Symbol: parts[1]},
	}
}

func CurrencyPair2InstrumentId(pair CurrencyPair) string {
	return fmt.Sprintf("%s-%s", pair.CurrencyA.Symbol, pair.CurrencyB.Symbol)
}

type OKExV3Spot struct {
	apiKey,
	apiSecretKey string
	passphrase string
	client            *http.Client

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	wsAccountHandleMap  map[string]func(*SubAccountDecimal)
	wsOrderHandleMap  map[string]func([]OrderDecimal)
	depthManagers	 map[string]*DepthManager
	errorHandle      func(error)
}

func NewOKExV3Spot(client *http.Client, api_key, secret_key, passphrase string) *OKExV3Spot {
	ok := new(OKExV3Spot)
	ok.apiKey = api_key
	ok.apiSecretKey = secret_key
	ok.passphrase = passphrase
	ok.client = client
	return ok
}

func (ok *OKExV3Spot) buildHeader(method, requestPath, body string) map[string]string {
	now := time.Now().In(time.UTC)
	timestamp := now.Format(V3_DATE_FORMAT)
	message := timestamp + method + requestPath + body
	signature, _ := GetParamHmacSHA256Base64Sign(ok.apiSecretKey, message)
	return map[string]string {
		"OK-ACCESS-KEY": ok.apiKey,
		"OK-ACCESS-SIGN": signature,
		"OK-ACCESS-TIMESTAMP": timestamp,
		"OK-ACCESS-PASSPHRASE": ok.passphrase,
		"Content-Type": "application/json",
	}
}

func (ok *OKExV3Spot) GetInstruments() ([]V3Instrument, error) {
	resp, err := ok.client.Get(SPOT_V3_API_BASE_URL + SPOT_V3_INSTRUMENTS)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var instruments []V3Instrument
	err = json.Unmarshal(body, &instruments)
	return instruments, err
}

func (ok *OKExV3Spot) GetTrades(instrumentId string) ([]TradeDecimal, error) {
	resp, err := ok.client.Get(SPOT_V3_API_BASE_URL + fmt.Sprintf(SPOT_V3_TRADES, instrumentId))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data []struct{
		Timestamp string
		TradeId decimal.Decimal 		`json:"trade_id"`
		Price decimal.Decimal
		Size decimal.Decimal
		Side string
	}
	err = json.Unmarshal(body, &data)

	var ret []TradeDecimal
	for _, o := range data {
		ret = append(ret, TradeDecimal{
			Tid: o.TradeId.IntPart(),
			Type: o.Side,
			Amount: o.Size,
			Price: o.Price,
			Date: V3ParseDate(o.Timestamp),
		})
	}

	return ret, err
}

func (ok *OKExV3Spot) GetInstrumentTicker(instrumentId string) (*TickerDecimal, error) {
	url := SPOT_V3_API_BASE_URL + SPOT_V3_INSTRUMENT_TICKER
	resp, err := ok.client.Get(fmt.Sprintf(url, instrumentId))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	tickerMap := make(map[string]interface{})

	err = json.Unmarshal(body, &tickerMap)
	if err != nil {
		return nil, err
	}

	ticker := new(TickerDecimal)
	ticker.Date = uint64(V3ParseDate(tickerMap["timestamp"].(string)))
	ticker.Buy, _ = decimal.NewFromString(tickerMap["best_bid"].(string))
	ticker.Sell, _ = decimal.NewFromString(tickerMap["best_ask"].(string))
	ticker.Last, _ = decimal.NewFromString(tickerMap["last"].(string))
	ticker.High, _ = decimal.NewFromString(tickerMap["high_24h"].(string))
	ticker.Low, _ = decimal.NewFromString(tickerMap["low_24h"].(string))
	ticker.Vol, _ = decimal.NewFromString(tickerMap["base_volume_24h"].(string))

	return ticker, nil
}

type V3CurrencyInfo struct {
	Currency string
	Balance decimal.Decimal		`json:"balance"`
	Hold decimal.Decimal		`json:"hold"`
	Available decimal.Decimal	`json:"available"`
	Id string 					`json:"id"`
}

func (this *V3CurrencyInfo) ToSubAccount() *SubAccountDecimal {
	a := new(SubAccountDecimal)

	a.Currency = Currency{Symbol: this.Currency}
	a.Amount = this.Balance
	a.FrozenAmount = this.Hold
	a.AvailableAmount = this.Available
	return a
}

func (ok *OKExV3Spot) GetAccount() (*AccountDecimal, error) {
	var resp []V3CurrencyInfo
	header := ok.buildHeader("GET", SPOT_V3_ACCOUNTS, "")
	err := HttpGet4(ok.client, SPOT_V3_API_BASE_URL + SPOT_V3_ACCOUNTS, header, &resp)
	if err != nil {
		return nil, err
	}

	ret := new(AccountDecimal)
	ret.Exchange = OKEX
	ret.SubAccounts = make(map[Currency]SubAccountDecimal)

	for _, a := range resp {
		currency := Currency{Symbol: a.Currency}
		ret.SubAccounts[currency] = *a.ToSubAccount()
	}

	return ret, nil
}

func (ok *OKExV3Spot) GetCurrencyAccount(currency Currency) (*SubAccountDecimal, error) {
	var resp *V3CurrencyInfo
	reqUrl := fmt.Sprintf(SPOT_V3_CURRENCY_ACCOUNTS, currency)
	header := ok.buildHeader("GET", reqUrl, "")
	err := HttpGet4(ok.client, SPOT_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	return resp.ToSubAccount(), nil
}

type OrderReq struct {
	ClientOid string 		`json:"client_oid"`
	Type string 			`json:"type"`
	Side string 			`json:"side"`
	InstrumentId string 	`json:"instrument_id"`
	OrderType string 		`json:"order_type"`
	MarginTrading int 		`json:"margin_trading"`
	Price decimal.Decimal	`json:"price"`
	Size decimal.Decimal	`json:"size"`
	Notional decimal.Decimal`json:"notional"`
}

func (this OrderReq) ToParam() map[string]interface{} {
	ret := make(map[string]interface{})
	if this.ClientOid != "" {
		ret["client_oid"] = this.ClientOid
	}
	if this.Type != "" {
		ret["type"] = this.Type
	}
	ret["side"] = this.Side
	ret["instrument_id"] = this.InstrumentId
	ret["order_type"] = this.OrderType
	if this.MarginTrading == 0 {
		ret["margin_trading"] = 1
	} else {
		ret["margin_trading"] = this.MarginTrading
	}
	if this.Type == "limit" {
		ret["price"] = this.Price
		ret["size"] = this.Size
	} else {
		if this.Side == "buy" {
			ret["size"] = this.Size
		} else {
			ret["notional"] = this.Notional
		}
	}
	return ret
}

func (ok *OKExV3Spot) PlaceOrder(req OrderReq) (string, error) {
	bytes, _ := json.Marshal(req.ToParam())
	data := string(bytes)
	println(data)

	header := ok.buildHeader("POST", SPOT_V3_ORDERS, data)

	placeOrderUrl := SPOT_V3_API_BASE_URL + SPOT_V3_ORDERS
	println(placeOrderUrl)
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return "", err
	}

	var ret *struct {
		OrderId string `json:"order_id"`
		ClientOid string `json:"client_oid"`
		ErrorCode int 	`json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Result bool `json:"result"`
	}
	println(string(body))
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return "", err
	}

	if ret.ErrorCode != 0 {
		return "", fmt.Errorf("error code: %d", ret.ErrorCode)
	}

	return ret.OrderId, nil
}

func (ok *OKExV3Spot) CancelOrder(instrumentId, orderId, clientOid string) error {
	var param = make(map[string]interface{})
	param["instrument_id"] = instrumentId
	var reqUrl string
	if orderId != "" {
		param["order_id"] = orderId
		reqUrl = fmt.Sprintf(SPOT_V3_CANCEL_ORDER, orderId)
	}
	if clientOid != "" {
		param["client_oid"] = clientOid
		reqUrl = fmt.Sprintf(SPOT_V3_CANCEL_ORDER, clientOid)
	}
	bytes, _ := json.Marshal(param)
	data := string(bytes)
	println(data)


	header := ok.buildHeader("POST", reqUrl, data)

	reqPath := SPOT_V3_API_BASE_URL + reqUrl
	println(reqPath)
	body, err := HttpPostJson(ok.client, reqPath, data, header)
	if err != nil {
		if strings.Contains(err.Error(), "33027") {
			return nil
		}
		return err
	}
	respMap := make(map[string]interface{})
	err = json.Unmarshal(body, &respMap)

	if respMap["result"] != nil && !respMap["result"].(bool) {
		if respMap["error_code"] != nil {
			return fmt.Errorf("error code: %s", respMap["error_code"].(string))
		}
		return errors.New(string(body))
	}

	return nil
}

type BatchPlaceOrderRespItem struct {
	ClientOid string 			`json:"client_oid"`
	OrderId string 				`json:"order_id"`
	Result bool 				`json:"result"`
	ErrorCode decimal.Decimal	`json:"error_code"`
	ErrorMessage string 		`json:"error_message"`
}

func (ok *OKExV3Spot) PlaceOrders(req []OrderReq) ([]BatchPlaceOrderRespItem, error) {
	util.Assert(len(req) > 0, "")
	instrumentId := req[0].InstrumentId
	for i := 1; i < len(req); i++ {
		if req[i].InstrumentId != instrumentId {
			return nil, errors.New("Bad instrumentId")
		}
	}

	param := make([]map[string]interface{}, len(req))
	for i := range req {
		param[i] = req[i].ToParam()
	}

	bytes, _ := json.Marshal(param)
	data := string(bytes)
	println(data)
	header := ok.buildHeader("POST", SPOT_V3_BATCH_ORDERS, data)

	placeOrderUrl := SPOT_V3_API_BASE_URL + SPOT_V3_BATCH_ORDERS
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return nil, err
	}

	var ret map[string][]BatchPlaceOrderRespItem

	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	for _, l := range ret {
		return l, nil
	}

	return nil, nil
}

func (ok *OKExV3Spot) CancelOrders(instrumentId string, orderIds []string, clientOid string) error {
	param := make(map[string]interface{})
	param["instrument_id"] = instrumentId
	if len(orderIds) > 0 {
		param["order_ids"] = orderIds
	} else if clientOid != "" {
		param["client_oid"] = clientOid
	} else {
		return errors.New("Bad param")
	}

	bytes, _ := json.Marshal([]interface{}{param})

	reqUrl := SPOT_V3_CANCEL_ORDERS

	header := ok.buildHeader("POST", reqUrl, string(bytes))

	reqPath := SPOT_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, string(bytes), header)
	if err != nil {
		if strings.Contains(err.Error(), "33027") || strings.Contains(err.Error(), "33014") {
			return nil
		}
		return err
	}

	var resp map[string]struct {
		Result bool 		`json:"result"`
		OrderIds []string 	`json:"order_ids"`
		ClientOid string 	`json:"client_oid"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	return nil
}

type V3OrderInfo struct {
	OrderId string 			`json:"order_id"`
	ClientOid string 		`json:"client_oid"`
	Price string			`json:"price"`
	Size string				`json:"size"`
	Notional string			`json:"notional"`
	InstrumentId string 	`json:"instrument_id"`
	Type string
	Side string
	Timestamp string
	FilledSize string 		`json:"filled_size"`
	FilledNotional string 	`json:"filled_notional"`
	Status string
	OrderType string 		`json:"order_type"`
}

func (this *V3OrderInfo) ToOrder() *OrderDecimal {
	if this.OrderId == "" {
		return nil
	}
	o := new(OrderDecimal)
	o.OrderID2 = this.OrderId
	o.ClientOid = this.ClientOid
	if this.Price != "" {
		o.Price,_ = decimal.NewFromString(this.Price)
	}
	if this.Side != "" {
		o.Amount, _ = decimal.NewFromString(this.Size)
	}
	if this.Notional != "" {
		o.Notinal, _ = decimal.NewFromString(this.Notional)
	}
	if this.Side == "buy" {
		if this.Type == "limit" {
			o.Side = BUY
		} else {
			o.Side = BUY_MARKET
		}
	} else {
		if this.Type == "limit" {
			o.Side = SELL
		} else {
			o.Side = SELL_MARKET
		}
	}
	o.Timestamp = V3ParseDate(this.Timestamp)
	if this.FilledSize != "" {
		o.DealAmount, _ = decimal.NewFromString(this.FilledSize)
	}
	if this.FilledNotional != "" {
		o.DealNotional, _ = decimal.NewFromString(this.FilledNotional)
	}
	switch this.Status {
	case ORDER_STATUS_ORDERING, ORDER_STATUS_OPEN:
		o.Status = ORDER_UNFINISH
	case ORDER_STATUS_PART_FILLED:
		o.Status = ORDER_PART_FINISH
	case ORDER_STATUS_FILLED:
		o.Status = ORDER_FINISH
	case ORDER_STATUS_CANCELING:
		o.Status = ORDER_CANCEL_ING
	case ORDER_STATUS_CANCELLED:
		o.Status = ORDER_CANCEL
	case ORDER_STATUS_FAILURE:
		o.Status = ORDER_REJECT
	}
	o.Currency = InstrumentId2CurrencyPair(this.InstrumentId)
	return o
}

const (
	ORDER_STATUS_ALL = "all"
	ORDER_STATUS_OPEN = "open"
	ORDER_STATUS_PART_FILLED = "part_filled"
	ORDER_STATUS_CANCELING = "canceling"
	ORDER_STATUS_FILLED = "filled"
	ORDER_STATUS_CANCELLED = "cancelled"
	ORDER_STATUS_ORDERING = "ordering"
	ORDER_STATUS_FAILURE = "failure"
)

func (ok *OKExV3Spot) GetInstrumentOrders(instrumentId string, status, from, to, limit string) ([]OrderDecimal, error) {
	reqUrl := fmt.Sprintf(SPOT_V3_INSTRUMENT_ORDERS, instrumentId)
	var params []string
	if status != "" {
		params = append(params, "status=" + status)
	}
	if from != "" {
		params = append(params, "from=" + from)
	}
	if to != "" {
		params = append(params, "to=" + to)
	}
	if limit != "" {
		params = append(params, "limit=" + limit)
	}
	if len(params) > 0 {
		reqUrl += "&" + strings.Join(params, "&")
	}

	header := ok.buildHeader("GET", reqUrl, "")

	var resp []V3OrderInfo

	err := HttpGet4(ok.client, SPOT_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	ret := make([]OrderDecimal, len(resp))
	for i, o := range resp {
		ret[i] = *o.ToOrder()
	}

	return ret, nil
}

func (ok *OKExV3Spot) GetInstrumentPendingOrders(instrumentId string, from, to, limit string) ([]OrderDecimal, error) {
	reqUrl := fmt.Sprintf(SPOT_V3_INSTRUMENT_ORDERS_PENDING, instrumentId)
	var params []string
	if from != "" {
		params = append(params, "from=" + from)
	}
	if to != "" {
		params = append(params, "to=" + to)
	}
	if limit != "" {
		params = append(params, "limit=" + limit)
	}
	if len(params) > 0 {
		reqUrl += "&" + strings.Join(params, "&")
	}

	header := ok.buildHeader("GET", reqUrl, "")

	var resp []V3OrderInfo

	err := HttpGet4(ok.client, SPOT_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	ret := make([]OrderDecimal, len(resp))
	for i, o := range resp {
		ret[i] = *o.ToOrder()
	}

	return ret, nil
}

func (ok *OKExV3Spot) GetInstrumentOrder(instrumentId string, orderId string) (*OrderDecimal, error) {
	reqUrl := fmt.Sprintf(SPOT_V3_ORDER_INFO, orderId, instrumentId)
	header := ok.buildHeader("GET", reqUrl, "")

	var resp *V3OrderInfo

	err := HttpGet4(ok.client, SPOT_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ToOrder(), nil
}
