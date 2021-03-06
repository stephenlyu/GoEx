package okcoin

import (
	. "github.com/stephenlyu/GoEx"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"fmt"
	"strconv"
	"strings"
	"errors"
	"sync"
	"github.com/shopspring/decimal"
)

const (
	FUTURE_V3_API_BASE_URL    = "https://www.okex.com"
	FUTURE_V3_TICKER 		  = "/api/futures/v3/instruments/%s/ticker"
	FUTURE_V3_TRADES 		  = "/api/futures/v3/instruments/%s/trades"
	FUTURE_V3_DEPTH 		  = "/api/futures/v3/instruments/%s/book?size=10"
	FUTURE_V3_INSTRUMENTS 	  = "/api/futures/v3/instruments"
	FUTURE_V3_POSITION 		  = "/api/futures/v3/position"
	FUTURE_V3_ACCOUNTS 		  = "/api/futures/v3/accounts"
	FUTURE_V3_CURRENCY_ACCOUNTS = "/api/futures/v3/accounts/%s"
	FUTURE_V3_INSTRUMENT_POSITION = "/api/futures/v3/%s/position"
	FUTURE_V3_INSTRUMENT_TICKER = "/api/futures/v3/instruments/%s/ticker"
	FUTURE_V3_INSTRUMENT_INDEX = "/api/futures/v3/instruments/%s/index"
	FUTURE_V3_ORDER			   = "/api/futures/v3/order"
	FUTURE_V3_ORDERS 		   = "/api/futures/v3/orders"
	FUTURE_V3_CANCEL_ORDERS    = "/api/futures/v3/cancel_batch_orders/%s"
	FUTURE_V3_CANCEL_ORDER		= "/api/futures/v3/cancel_order/%s/%s"
	FUTURE_V3_INSTRUMENT_ORDERS = "/api/futures/v3/orders/%s"
	FUTURE_V3_ORDER_INFO 		= "/api/futures/v3/orders/%s/%s"
	FUTURE_V3_FILLS 			= "/api/futures/v3/fills"
	WALLET_V3_TRANSFER 			= "/api/account/v3/transfer"
	WALLET_V3_INFO 				= "/api/account/v3/wallet/%s"
	V3_WITHDRAW_FEE				= "/api/account/v3/withdrawal/fee"
	V3_WITHDRAW					= "/api/account/v3/withdrawal"
	V3_DEPOSIT_HISTORY 			= "/api/account/v3/deposit/history/%s"
	V3_WITHDRAW_HISTORY 		= "/api/account/v3/withdrawal/history/%s"
)

const (
	V3_DATE_FORMAT = "2006-01-02T15:04:05.000Z"
)

const (
	V3_TYPE_BUY_OPEN = "1"
	V3_TYPE_SELL_OPEN = "2"
	V3_TYPE_SELL_CLOSE = "3"
	V3_TYPE_BUY_CLOSE = "4"
)

const (
	V3_ORDER_TYPE_NORMAL = 0
	V3_ORDER_TYPE_POST_ONLY = 1
	V3_ORDER_TYPE_FOK = 2
	V3_ORDER_TYPE_IOC = 3
)

type V3Instrument struct {
	ContractVal string 		`json:"contract_val"`
	Delivery string
	InstrumentId string 	`json:"instrument_id"`
	Listing string
	QuoteCurrency string 	`json:"quote_currency"`
	TickSize string 		`json:"tick_size"`
	TradeIncrement string 	`json:"trade_increment"`
	UnderlyingIndex string 	`json:"underlying_index"`
	Alias string 			`json:"alias"`
}

type V3Position struct {
	CreateAt string 		`json:"create_at"`
	InstrumentId string 	`json:"instrument_id"`
	Leverage string 		`json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
	LongAvailQty string 	`json:"long_avail_qty"`
	LongAvgCost string 		`json:"long_avg_cost"`
	LongQty string 			`json:"long_qty"`
	LongSettlementPrice string `json:"long_settlement_price"`
	MarginMode string 		`json:"margin_mode"`
	RealisedPnl string 		`json:"realised_pnl"`
	ShortAvailQty string 	`json:"short_avail_qty"`
	ShortAvgCost string 	`json:"short_avg_cost"`
	ShortQty string 		`json:"short_qty"`
	ShortSettlementPrice string `json:"short_settlement_price"`
	UpdatedAt string 		`json:"updated_at"`
}

func (this *V3Position) ToFuturePosition() *FuturePosition {
	p := &FuturePosition{}
	p.BuyAmount, _ = strconv.ParseFloat(this.LongQty, 64)
	p.BuyAvailable, _ = strconv.ParseFloat(this.LongAvailQty, 64)
	p.BuyPriceAvg, _ = strconv.ParseFloat(this.LongAvgCost, 64)
	if this.CreateAt != "" {
		p.CreateDate = V3ParseDate(this.CreateAt)
	}
	p.LeverRate, _ = strconv.Atoi(this.Leverage)
	p.SellAmount, _ = strconv.ParseFloat(this.ShortQty, 64)
	p.SellAvailable, _ = strconv.ParseFloat(this.ShortAvailQty, 64)
	p.SellPriceAvg, _ = strconv.ParseFloat(this.ShortAvgCost, 64)
	p.ForceLiquPrice, _ = strconv.ParseFloat(this.LiquidationPrice, 64)
	p.InstrumentId = this.InstrumentId
	p.Symbol = InstrumentId2CurrencyPair(this.InstrumentId)

	return p
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

type OKExV3 struct {
	apiKey,
	apiSecretKey string
	passphrase string
	client            *http.Client

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*Depth)
	wsTradeHandleMap map[string]func(string, []Trade)
	wsIndexTickerHandleMap map[string]func(string, []Ticker)
	wsFundingRateHandleMap map[string]func(SWAPFundingRate)
	wsPositionHandleMap  map[string]func([]FuturePosition)
	wsAccountHandleMap  map[string]func(bool, *FutureAccount)
	wsOrderHandleMap  map[string]func([]FutureOrder)
	depthManagers	 map[string]*DepthManager
	errorHandle      func(error)
}

func NewOKExV3(client *http.Client, api_key, secret_key, passphrase string) *OKExV3 {
	ok := new(OKExV3)
	ok.apiKey = api_key
	ok.apiSecretKey = secret_key
	ok.passphrase = passphrase
	ok.client = client
	return ok
}

func (ok *OKExV3) buildHeader(method, requestPath, body string) map[string]string {
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

func (ok *OKExV3) GetInstruments() ([]V3Instrument, error) {
	resp, err := ok.client.Get(FUTURE_V3_API_BASE_URL + FUTURE_V3_INSTRUMENTS)
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
	if err != nil {
		return nil, fmt.Errorf("error: %s", string(body))
	}
	return instruments, err
}

//{
//"instrument_id":"EOS-USD-190628",
//"last":"3.708",
//"best_bid":"3.707",
//"best_ask":"3.708",
//"high_24h":"3.799",
//"low_24h":"3.494",
//"volume_24h":"24884595",
//"timestamp":"2019-03-21T03:15:50.144Z"
//}
func (this *OKExV3) GetTicker(instrumentId string) (*TickerDecimal, error) {
	url := FUTURE_V3_API_BASE_URL + FUTURE_V3_TICKER
	resp, err := this.client.Get(fmt.Sprintf(url, instrumentId))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data struct {
		High decimal.Decimal	`json:"high_24h"`
		Vol decimal.Decimal		`json:"volume_24h"`
		Last decimal.Decimal
		Low decimal.Decimal		`json:"low_24h"`
		Buy decimal.Decimal		`json:"best_bid"`
		Sell decimal.Decimal	`json:"best_ask"`
		Timestamp string
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	ticker := new(TickerDecimal)
	ticker.Date = uint64(V3ParseDate(data.Timestamp))
	ticker.Buy = data.Buy
	ticker.Sell = data.Sell
	ticker.Last = data.Last
	ticker.High = data.High
	ticker.Low = data.Low
	ticker.Vol = data.Vol

	return ticker, nil
}

func (this *OKExV3) GetDepth(instrumentId string) (*DepthDecimal, error) {
	url := fmt.Sprintf(FUTURE_V3_API_BASE_URL + FUTURE_V3_DEPTH, instrumentId)
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
		 Asks [][]decimal.Decimal
		 Bids [][]decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	r := data

	depth := new(DepthDecimal)
	depth.Pair = InstrumentId2CurrencyPair(instrumentId)

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

//[
//[
//{
//"trade_id":"2522253054345222",
//"side":"sell",
//"price":"3.625",
//"qty":"24",
//"timestamp":"2019-03-22T02:42:07.323Z"
//}
//]
func (this *OKExV3) GetTrades(instrumentId string) ([]TradeDecimal, error) {
	url := fmt.Sprintf(FUTURE_V3_API_BASE_URL + FUTURE_V3_TRADES, instrumentId)
	resp, err := this.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data []struct {
		Qty decimal.Decimal
		Price decimal.Decimal
		Tid decimal.Decimal				`json:"trade_id"`
		Side string
		Timestamp string
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	var trades = make([]TradeDecimal, len(data))

	for i, o := range data {
		t := &trades[i]
		t.Tid = o.Tid.IntPart()
		t.Amount = o.Qty
		t.Price = o.Price
		t.Type = o.Side
		t.Date = V3ParseDate(o.Timestamp)
	}

	return trades, nil
}

func (ok *OKExV3) GetPosition() ([]FuturePosition, error) {
	var result struct {
		Holding [][]V3Position
	}
	header := ok.buildHeader("GET", FUTURE_V3_POSITION, "")
	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + FUTURE_V3_POSITION, header, &result)
	if err != nil {
		return nil, err
	}

	var ret []FuturePosition
	for _, positions := range result.Holding {
		for _, p := range positions {
			ret = append(ret, *p.ToFuturePosition())
		}
	}

	return ret, err
}

func (ok *OKExV3) GetInstrumentPosition(instrumentId string) ([]FuturePosition, error) {
	var result struct {
		Holding []V3Position
	}
	reqPath := fmt.Sprintf(FUTURE_V3_INSTRUMENT_POSITION, instrumentId)
	header := ok.buildHeader("GET", reqPath, "")
	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqPath, header, &result)
	if err != nil {
		return nil, err
	}

	var ret []FuturePosition
	for _, p := range result.Holding {
		ret = append(ret, *p.ToFuturePosition())
	}

	return ret, err
}


func (ok *OKExV3) GetInstrumentTicker(instrumentId string) (*Ticker, error) {
	url := FUTURE_V3_API_BASE_URL + FUTURE_V3_INSTRUMENT_TICKER
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

	fmt.Println(tickerMap)

	ticker := new(Ticker)
	ticker.Date = uint64(V3ParseDate(tickerMap["timestamp"].(string)))
	ticker.Buy, _ = strconv.ParseFloat(tickerMap["best_bid"].(string), 64)
	ticker.Sell, _ = strconv.ParseFloat(tickerMap["best_ask"].(string), 64)
	ticker.Last, _ = strconv.ParseFloat(tickerMap["last"].(string), 64)
	ticker.High, _ = strconv.ParseFloat(tickerMap["high_24h"].(string), 64)
	ticker.Low, _ = strconv.ParseFloat(tickerMap["low_24h"].(string), 64)
	ticker.Vol, _ = strconv.ParseFloat(tickerMap["volume_24h"].(string), 64)

	return ticker, nil
}

func (ok *OKExV3) GetInstrumentIndex(instrumentId string) (float64, error) {
	resp, err := ok.client.Get(fmt.Sprintf(FUTURE_V3_API_BASE_URL+FUTURE_V3_INSTRUMENT_INDEX, instrumentId))
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return 0, err
	}

	bodyMap := make(map[string]interface{})

	err = json.Unmarshal(body, &bodyMap)
	if err != nil {
		return 0, err
	}

	v, yes := bodyMap["index"]
	if !yes {
		println(string(body))
		return 0, errors.New("No future_index field")
	}

	ret, yes := v.(string)
	if !yes {
		return 0, errors.New("Bad future_index")
	}

	return strconv.ParseFloat(ret, 64)
}

type V3CurrencyInfo struct {
	Equity string
	Margin string
	MarginMode string		`json:"margin_mode"`
	MarginRatio string 		`json:"margin_ratio"`
	TotalAvailBalance string `json:"total_avail_balance"`
	RealizedPnl string 		`json:"realized_pnl"`
	UnrealizedPnl string 	`json:"unrealized_pnl"`
}

func (this *V3CurrencyInfo) ToFutureSubAccount(currency Currency) *FutureSubAccount {
	a := new(FutureSubAccount)

	a.Currency = currency
	a.AccountRights, _ = strconv.ParseFloat(this.Equity, 64)
	a.KeepDeposit, _ = strconv.ParseFloat(this.TotalAvailBalance, 64)
	a.RiskRate, _ = strconv.ParseFloat(this.MarginRatio, 64)

	a.ProfitReal, _ = strconv.ParseFloat(this.RealizedPnl, 64)
	a.ProfitUnreal, _ = strconv.ParseFloat(this.UnrealizedPnl, 64)
	return a
}

type V3AccountsResponse struct {
	Info map[string]V3CurrencyInfo `json:info`
	Result     bool `json:"result,bool"`
	Error_code int  `json:"error_code"`
}

func (ok *OKExV3) GetAccount() (*FutureAccount, error) {
	var resp *V3AccountsResponse
	header := ok.buildHeader("GET", FUTURE_V3_ACCOUNTS, "")
	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + FUTURE_V3_ACCOUNTS, header, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Result && resp.Error_code > 0 {
		return nil, fmt.Errorf("error code: %d", resp.Error_code)
	}

	account := new(FutureAccount)
	account.FutureSubAccounts = make(map[Currency]FutureSubAccount)

	for key, info := range resp.Info {
		currency := NewCurrency(key, "")
		account.FutureSubAccounts[currency] = *info.ToFutureSubAccount(currency)
	}

	return account, nil
}

func (ok *OKExV3) GetCurrencyAccount(currency Currency) (*FutureSubAccount, error) {
	var resp *V3CurrencyInfo
	reqUrl := fmt.Sprintf(FUTURE_V3_CURRENCY_ACCOUNTS, currency)
	header := ok.buildHeader("GET", reqUrl, "")
	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	return resp.ToFutureSubAccount(currency), nil
}

func (ok *OKExV3) PlaceFutureOrder(clientOid string, instrumentId string, price, size string, _type, orderType, matchPrice, leverage int) (string, error) {
	params := map[string]string {
		"client_oid": clientOid,
		"instrument_id": instrumentId,
		"type": strconv.Itoa(_type),
		"order_type": strconv.Itoa(orderType),
		"price": price,
		"size": size,
		"match_price": strconv.Itoa(matchPrice),
		"leverage": strconv.Itoa(leverage),
	}
	bytes, _ := json.Marshal(params)
	data := string(bytes)

	header := ok.buildHeader("POST", FUTURE_V3_ORDER, data)

	placeOrderUrl := FUTURE_V3_API_BASE_URL + FUTURE_V3_ORDER
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return "", err
	}

	var ret *struct {
		OrderId string `json:"order_id"`
		ClientOid string `json:"client_oid"`
		ErrorCode decimal.Decimal 	`json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Result bool `json:"result"`
	}

	err = json.Unmarshal(body, &ret)
	if err != nil {
		panic(err)
		return "", err
	}

	if ret.ErrorCode.IntPart() != 0 {
		return "", fmt.Errorf("error code: %d", ret.ErrorCode.IntPart())
	}

	return ret.OrderId, nil
}

func (ok *OKExV3) FutureCancelOrder(instrumentId, orderId string) error {
	reqUrl := fmt.Sprintf(FUTURE_V3_CANCEL_ORDER, instrumentId, orderId)

	header := ok.buildHeader("POST", reqUrl, "")

	reqPath := FUTURE_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, "", header)
	if err != nil {
		fmt.Println("FutureCancelOrder fail, error: ", err)
		return err
	}

	respMap := make(map[string]interface{})
	err = json.Unmarshal(body, &respMap)
	if err != nil {
		return err
	}
	if respMap["result"] != nil && !respMap["result"].(bool) {
		if respMap["error_code"] != nil {
			return fmt.Errorf("error code: %s", respMap["error_code"].(string))
		}
		return errors.New(string(body))
	}

	return nil
}

type OrderItem struct {
	ClientOid string 	`json:"client_oid"`
	Type string 		`json:"type"`
	OrderType string 	`json:"order_type"`
	Price string 		`json:"price"`
	Size string 		`json:"size"`
	MatchPrice string 	`json:"match_price"`
}

type BatchPlaceOrderReq struct {
	InstrumentId string 		`json:"instrument_id"`
	OrdersData []OrderItem 		`json:"orders_data"`
	Leverage int 				`json:"leverage"`
}

type BatchPlaceOrderRespItem struct {
	ErrorMessage string 		`json:"error_message"`
	ErrorCode decimal.Decimal 	`json:"error_code"`
	ClientOid string 			`json:"client_oid"`
	OrderId string 				`json:"order_id"`
}

func (ok *OKExV3) PlaceFutureOrders(req BatchPlaceOrderReq) ([]BatchPlaceOrderRespItem, error) {
	bytes, _ := json.Marshal(req)
	data := string(bytes)

	header := ok.buildHeader("POST", FUTURE_V3_ORDERS, data)

	placeOrderUrl := FUTURE_V3_API_BASE_URL + FUTURE_V3_ORDERS
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return nil, err
	}

	var ret *struct {
		Result bool `json:"result"`
		Data []BatchPlaceOrderRespItem		`json:"order_info"`
	}

	err = json.Unmarshal(body, &ret)
	if err != nil {
		panic(err)
		return nil, err
	}

	if !ret.Result {
		return nil, fmt.Errorf("place order fail, body: %s", string(body))
	}

	return ret.Data, nil
}

func (ok *OKExV3) FutureCancelOrders(instrumentId string, orderIds, clientOids []string) error {
	p := map[string]interface{} {
	}
	if len(orderIds) > 0 {
		p["order_ids"] = orderIds
	} else if len(clientOids) > 0 {
		p["client_oids"] = clientOids
	} else {
		return errors.New("bad params")
	}

	bytes, _ := json.Marshal(p)

	reqUrl := fmt.Sprintf(FUTURE_V3_CANCEL_ORDERS, instrumentId)

	header := ok.buildHeader("POST", reqUrl, string(bytes))

	reqPath := FUTURE_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, string(bytes), header)
	if err != nil {
		return err
	}

	var resp struct {
		Result bool 		`json:"result"`
		OrderIds []string 	`json:"order_ids"`
		ClientOids []string `json:"client_oids"`
		InstrumentId string `json:"instrument_id"`
		ErrorMessage string `json:"error_message"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if !resp.Result {
		return errors.New(resp.ErrorMessage)
	}

	return nil
}

type V3OrderInfo struct {
	InstrumentId string 	`json:"instrument_id"`
	Size string
	Timestamp string
	FilledQty string 		`json:"filled_qty"`
	Fee string
	OrderId string 			`json:"order_id"`
	ClientOid string 		`json:"client_oid"`
	Price string
	PriceAvg string 		`json:"price_avg"`
	Status string
	Type string
	ContractVal string 		`json:"contract_val"`
	Leverage string
}

func (this *V3OrderInfo) ToFutureOrder() *FutureOrder {
	if this.OrderId == "" {
		return nil
	}
	o := new(FutureOrder)
	o.Price, _ = strconv.ParseFloat(this.Price, 64)
	o.Amount, _  = strconv.ParseFloat(this.Size, 64)
	o.AvgPrice, _ = strconv.ParseFloat(this.PriceAvg, 64)
	o.DealAmount, _ = strconv.ParseFloat(this.FilledQty, 64)
	o.OrderID2 = this.OrderId
	o.ClientOrderID = this.ClientOid
	o.OrderTime = V3ParseDate(this.Timestamp)
	switch this.Status {
	case "-1":
		o.Status = ORDER_CANCEL
	case "0":
		o.Status = ORDER_UNFINISH
	case "1":
		o.Status = ORDER_PART_FINISH
	case "2":
		o.Status = ORDER_FINISH
	case "4":
		o.Status = ORDER_CANCEL_ING
	}
	o.Currency = InstrumentId2CurrencyPair(this.InstrumentId)
	o.OType, _ = strconv.Atoi(this.Type)
	o.Fee, _ = strconv.ParseFloat(this.Fee, 64)
	o.LeverRate, _ = strconv.Atoi(this.Leverage)
	o.ContractName = this.InstrumentId
	return o
}

func (ok *OKExV3) GetInstrumentOrder(instrumentId string, orderId string) (*FutureOrder, error) {
	reqUrl := fmt.Sprintf(FUTURE_V3_ORDER_INFO, instrumentId, orderId)
	header := ok.buildHeader("GET", reqUrl, "")

	var resp *V3OrderInfo

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ToFutureOrder(), nil
}

type V3Fill struct {
	TradeId string 			`json:"trade_id"`
	InstrumentId string 	`json:"instrument_id"`
	Price decimal.Decimal
	OrderQty decimal.Decimal 	`json:"order_qty"`
	OrderId string			`json:"order_id"`
	CreatedAt string 		`json:"created_at"`
	ExecType string 		`json:"exec_type"`
	Fee decimal.Decimal
	Side string
}

func (this *V3Fill) ToFutureFill() *FutureFillDecimal {
	if this.TradeId == "" {
		return nil
	}
	o := new(FutureFillDecimal)
	o.FillId = this.TradeId
	o.ContractName = this.InstrumentId
	o.Price = this.Price
	o.Qty = this.OrderQty
	o.OrderId = this.OrderId
	o.TransactionTime = V3ParseDate(this.CreatedAt)
	o.Fee = this.Fee
	if this.Side == "buy" {
		o.Side = BUY
	} else {
		o.Side = SELL
	}
	if this.ExecType == "M" {
		o.IsMaker = true
	}
	return o
}

func (ok *OKExV3) GetOrderFills(instrumentId, orderId string, from, to, limit string) ([]FutureFillDecimal, error) {
	reqUrl := FUTURE_V3_FILLS
	var params = []string {
		"instrument_id=" + instrumentId,
		"order_id=" + orderId,
	}
	if from != "" {
		params = append(params, "before=" + from)
	}
	if to != "" {
		params = append(params, "after=" + to)
	}
	if limit != "" {
		params = append(params, "limit=" + limit)
	}
	if len(params) > 0 {
		reqUrl += "?" + strings.Join(params, "&")
	}

	header := ok.buildHeader("GET", reqUrl, "")

	var fills []V3Fill

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &fills)
	if err != nil {
		return nil, err
	}

	ret := make([]FutureFillDecimal, len(fills))
	for i, o := range fills {
		ret[i] = *o.ToFutureFill()
	}

	return ret, nil
}

func (ok *OKExV3) GetInstrumentOrders(instrumentId string, status, from, to, limit string) ([]FutureOrder, error) {
	reqUrl := fmt.Sprintf(FUTURE_V3_INSTRUMENT_ORDERS, instrumentId)
	var params []string
	if status != "" {
		params = append(params, "status=" + status)
	}
	if from != "" {
		params = append(params, "before=" + from)
	}
	if to != "" {
		params = append(params, "after=" + to)
	}
	if limit != "" {
		params = append(params, "limit=" + limit)
	}
	if len(params) > 0 {
		reqUrl += "?" + strings.Join(params, "&")
	}

	header := ok.buildHeader("GET", reqUrl, "")

	var resp *struct{
		Result bool
		Orders []V3OrderInfo		`json:"order_info"`
	}

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Result {
		return nil, errors.New("query orders fail")
	}

	ret := make([]FutureOrder, len(resp.Orders))
	for i, o := range resp.Orders {
		ret[i] = *o.ToFutureOrder()
	}

	return ret, nil
}

type FutureLedger struct {
	Amount string			`json:"amount"`
	Balance string			`json:"balance"`
	Currency string			`json:"currency"`
	Details struct {
		InstrumentId string `json:"instrument_id"`
		OrderId decimal.Decimal `json:"order_id"`
			}				`json:"details"`
	LedgerId string 		`json:"ledger_id"`
	Timestamp string		`json:"timestamp"`
	Type string				`json:"type"`
}

func (ok *OKExV3) GetLedger(currency Currency, from, to, limit string) ([]FutureLedger, error) {
	reqUrl := fmt.Sprintf("/api/futures/v3/accounts/%s/ledger", strings.ToLower(currency.Symbol))
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
		reqUrl += "?" + strings.Join(params, "&")
	}
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []FutureLedger

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

const (
	WalletLedgerTypeDeposit = "1"
	WalletLedgerTypeWithdraw = "2"
	WalletLedgerTypeCancelWithdraw = "13"
	WalletLedgerTypeToFuture = "18"
	WalletLedgerTypeFromFuture = "19"
	WalletLedgerTypeToSubAccount = "20"
	WalletLedgerTypeFromSubAccount = "21"
	WalletLedgerTypeToSpot = "37"
	WalletLedgerTypeFromSpot = "38"
	WalletLedgerTypeGet = "28"
)

type WalletLedger struct {
	Amount decimal.Decimal
	Balance decimal.Decimal
	Currency string
	Fee decimal.Decimal
	LedgerId decimal.Decimal 	`json:"ledger_id"`
	Timestamp string
	TypeName string 			`json:"typename"`
}

func (ok *OKExV3) GetWalletLedger(currency Currency, from, to, limit, _type string) ([]WalletLedger, error) {
	reqUrl := fmt.Sprintf("/api/account/v3/ledger")
	var params []string
	params = append(params, "currency=" + strings.ToLower(currency.Symbol))
	if _type != "" {
		params = append(params, "type=" + _type)
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
		reqUrl += "?" + strings.Join(params, "&")
	}
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []WalletLedger

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

const (
	WALLET_ACCOUNT_SUB = 0
	WALLET_ACCOUNT_SPOT = 1
	WALLET_ACCOUNT_FUTURE = 3
	WALLET_ACCOUNT_C2C = 4
	WALLET_ACCOUNT_LEVERAGE = 5
	WALLET_ACCOUNT_WALLET = 6
	WALLET_ACCOUNT_ETT = 7
	WALLET_ACCOUNT_FUND = 8
	WALLET_ACCOUNT_SWAP = 9
)

type TransferResp struct {
	TransferId decimal.Decimal 	`json:"transfer_id"`
	Result bool 				`json:"result"`
	Currency string 			`json:"currency"`
	From decimal.Decimal 		`json:"from"`
	Amount decimal.Decimal 		`json:"amount"`
	To decimal.Decimal 			`json:"to"`
}

func (ok *OKExV3) WalletTransfer(currency Currency, amount float64, from, to int, subAccount string, instrumentId string) (error, *TransferResp) {
	param := map[string]interface{} {
		"currency": currency.Symbol,
		"amount": amount,
		"from": from,
		"to": to,
		"sub_account": subAccount,
		"instrment_id": instrumentId,
	}
	bytes, _ := json.Marshal(param)

	header := ok.buildHeader("POST", WALLET_V3_TRANSFER, string(bytes))

	reqPath := FUTURE_V3_API_BASE_URL + WALLET_V3_TRANSFER
	body, err := HttpPostJson(ok.client, reqPath, string(bytes), header)
	if err != nil {
		return err, nil
	}
	var resp *TransferResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err, nil
	}
	if !resp.Result {
		return errors.New(string(body)), nil
	}

	return nil, resp
}

type WalletCurrency struct {
	Balance decimal.Decimal
	Hold decimal.Decimal
	Available decimal.Decimal
	Currency string
}

func (ok *OKExV3) GetWallet(currency Currency) (*WalletCurrency, error) {
	reqUrl := fmt.Sprintf(WALLET_V3_INFO, strings.ToLower(currency.Symbol))
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []WalletCurrency

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return &WalletCurrency{Currency: currency.Symbol}, nil
	}

	return &resp[0], nil
}

type WithDrawFee struct {
	Currency string
	MinFee decimal.Decimal 	`json:"min_fee"`
	MaxFee decimal.Decimal 	`json:"max_fee"`
}

func (ok *OKExV3) GetWithdrawFee(currency string) ([]WithDrawFee, error) {
	reqUrl := V3_WITHDRAW_FEE
	if currency != "" {
		reqUrl += "?currency=" + currency
	}
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []WithDrawFee

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, nil
	}

	return resp, nil
}

const (
	WithdrawDestinationOKCoin = 2
	WithdrawDestinationOkex = 3
	WithdrawDestinationOuter = 4
)

type WithdrawResp struct {
	Amount decimal.Decimal
	WithdrawalId decimal.Decimal		`json:"withdrawal_id"`
	Currency string
	Result bool
}

func (ok *OKExV3) Withdraw(currency Currency, amount float64, destination int, toAddress string, tradePwd string, fee float64) (error, *WithdrawResp) {
	param := map[string]interface{} {
		"currency": currency.Symbol,
		"amount": amount,
		"destination": destination,
		"to_address": toAddress,
		"trade_pwd": tradePwd,
		"fee": fee,
	}
	bytes, _ := json.Marshal(param)

	header := ok.buildHeader("POST", V3_WITHDRAW, string(bytes))

	reqPath := FUTURE_V3_API_BASE_URL + V3_WITHDRAW
	body, err := HttpPostJson(ok.client, reqPath, string(bytes), header)
	if err != nil {
		return err, nil
	}

	var resp *WithdrawResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err, nil
	}
	if !resp.Result {
		return errors.New(string(body)), nil
	}

	return nil, resp
}

type DepositRecord struct {
	Amount decimal.Decimal
	Txid string
	Currency string
	To string
	Timestamp string
	Status decimal.Decimal
}

func (ok *OKExV3) GetDepositHistory(currency string) ([]DepositRecord, error) {
	reqUrl := fmt.Sprintf(V3_DEPOSIT_HISTORY, currency)
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []DepositRecord

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, nil
	}

	return resp, nil
}

type WithdrawRecord struct {
	Amount	decimal.Decimal	//	数量
	WithdrawalId decimal.Decimal 	`json:"withdrawal_id"`
	Currency string
	Timestamp string		// 提币申请时间
	From string				// 提币地址(如果收币地址是OKEx平台地址，则此处将显示用户账户)
	To string				// 收币地址
	Tag	string				// 部分币种提币需要标签，若不需要则不返回此字段
	PaymentId string		`json:"payment_id"`		// 部分币种提币需要此字段，若不需要则不返回此字段
	Txid string				// 提币哈希记录(内部转账将不返回此字段)
	Fee	string				// 提币手续费和对应币种，如0.00000009btc
	Status decimal.Decimal			// 提现状态（-3:撤销中;-2:已撤销;-1:失败;0:等待提现;1:提现中;2:已汇出;3:邮箱确认;4:人工审核中5:等待身份认证）
}

func (ok *OKExV3) GetWithdrawHistory(currency string) ([]WithdrawRecord, error) {
	reqUrl := fmt.Sprintf(V3_WITHDRAW_HISTORY, currency)
	header := ok.buildHeader("GET", reqUrl, "")

	var resp []WithdrawRecord

	err := HttpGet4(ok.client, FUTURE_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, nil
	}

	return resp, nil
}