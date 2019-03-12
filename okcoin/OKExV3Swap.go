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
)

const (
	SWAP_V3_API_BASE_URL    = "https://www.okex.com"
	SWAP_V3_INSTRUMENTS 	  = "/api/swap/v3/instruments"
	SWAP_V3_POSITION 		  = "/api/swap/v3/position"
	SWAP_V3_ACCOUNTS 		  = "/api/swap/v3/accounts"
	SWAP_V3_INSTRUMENT_ACCOUNTS = "/api/swap/v3/%s/accounts"
	SWAP_V3_INSTRUMENT_POSITION = "/api/swap/v3/%s/position"
	SWAP_V3_INSTRUMENT_TICKER = "/api/swap/v3/instruments/%s/ticker"
	SWAP_V3_INSTRUMENT_INDEX = "/api/swap/v3/instruments/%s/index"
	SWAP_V3_ORDER			   = "/api/swap/v3/order"
	SWAP_V3_ORDERS 		   = "/api/swap/v3/orders"
	SWAP_V3_CANCEL_ORDERS    = "/api/swap/v3/cancel_batch_orders/%s"
	SWAP_V3_CANCEL_ORDER		= "/api/swap/v3/cancel_order/%s/%s"
	SWAP_V3_INSTRUMENT_ORDERS = "/api/swap/v3/orders/%s"
	SWAP_V3_ORDER_INFO 		= "/api/swap/v3/orders/%s/%s"
)

const (
	V3_SWAP_DATE_FORMAT = "2006-01-02T15:04:05.000Z"
)

const (
	V3_SWAP_ORDER_TYPE_NORMAL = 0
	V3_SWAP_ORDER_TYPE_POST_ONLY = 1
	V3_SWAP_ORDER_TYPE_FOK = 2
	V3_SWAP_ORDER_TYPE_IOC = 3
)

type V3_SWAPInstrument struct {
	ContractVal string 		`json:"contract_val"`
	InstrumentId string 	`json:"instrument_id"`
	Coin string 			`json:"coin"`
	Listing string
	QuoteCurrency string 	`json:"quote_currency"`
	TickSize string 		`json:"tick_size"`
	SizeIncrement string 	`json:"size_increment"`
	UnderlyingIndex string 	`json:"underlying_index"`
}

type V3_SWAPPosition struct {
	MarginMode string 		`json:"margin_mode"`
	LiquidationPrice string `json:"liquidation_price"`
	Position string 		`json:"position"`
	AvailPosition string 	`json:"avail_position"`
	Margin string 			`json:"margin"`
	AvgCost string 			`json:"avg_cost"`
	SettlementPrice string 	`json:"settlement_price"`
	InstrumentId string 	`json:"instrument_id"`
	Leverage string 		`json:"leverage"`
	RealisedPnl string 		`json:"realised_pnl"`
	Side string 			`json:"side"`
	Timestamp string 		`json:"timestamp"`
}

func (this *V3_SWAPPosition) ToFuturePosition() *FuturePosition {
	p := &FuturePosition{}
	if this.Side == "long" {
		p.BuyAmount, _ = strconv.ParseFloat(this.Position, 64)
		p.BuyAvailable, _ = strconv.ParseFloat(this.AvailPosition, 64)
		p.BuyPriceAvg, _ = strconv.ParseFloat(this.AvgCost, 64)
	} else if this.Side == "short" {
		p.SellAmount, _ = strconv.ParseFloat(this.Position, 64)
		p.SellAvailable, _ = strconv.ParseFloat(this.AvailPosition, 64)
		p.SellPriceAvg, _ = strconv.ParseFloat(this.AvgCost, 64)
	}

	if this.Timestamp != "" {
		p.CreateDate = V3_SWAPParseDate(this.Timestamp)
	}
	p.LeverRate, _ = strconv.Atoi(this.Leverage)
	p.ForceLiquPrice, _ = strconv.ParseFloat(this.LiquidationPrice, 64)
	p.InstrumentId = this.InstrumentId
	p.Symbol = V3SWAPInstrumentId2CurrencyPair(this.InstrumentId)

	return p
}

func V3_SWAPParseDate(s string) int64 {
	t, _ := time.ParseInLocation(V3_SWAP_DATE_FORMAT, s, time.UTC)
	return t.UnixNano() / int64(time.Millisecond)
}

func V3SWAPInstrumentId2CurrencyPair(instrumentId string) CurrencyPair {
	parts := strings.Split(instrumentId, "-")
	return CurrencyPair{
		Currency{Symbol: parts[0]},
		Currency{Symbol: parts[1]},
	}
}

func V3SWAPInstrumentId2Currency(instrumentId string) Currency {
	parts := strings.Split(instrumentId, "-")
	return Currency{Symbol: parts[0]}
}

type OKExV3_SWAP struct {
	apiKey,
	apiSecretKey string
	passphrase string
	client            *http.Client

	ws                *WsConn
	createWsLock      sync.Mutex
	wsDepthHandleMap  map[string]func(*Depth)
	wsTradeHandleMap map[string]func(string, []Trade)
	depthManagers	 map[string]*DepthManager
}

func NewOKExV3_SWAP(client *http.Client, api_key, secret_key, passphrase string) *OKExV3_SWAP {
	ok := new(OKExV3_SWAP)
	ok.apiKey = api_key
	ok.apiSecretKey = secret_key
	ok.passphrase = passphrase
	ok.client = client
	return ok
}

func (ok *OKExV3_SWAP) buildHeader(method, requestPath, body string) map[string]string {
	now := time.Now().In(time.UTC)
	timestamp := now.Format(V3_SWAP_DATE_FORMAT)
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

func (ok *OKExV3_SWAP) GetInstruments() ([]V3_SWAPInstrument, error) {
	resp, err := ok.client.Get(SWAP_V3_API_BASE_URL + SWAP_V3_INSTRUMENTS)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var instruments []V3_SWAPInstrument
	err = json.Unmarshal(body, &instruments)
	return instruments, err
}

func (ok *OKExV3_SWAP) GetPosition() ([]FuturePosition, error) {
	var result []struct {
		MarginMode string 	`json:"margin_mode"`
		Holding []V3_SWAPPosition
	}
	header := ok.buildHeader("GET", SWAP_V3_POSITION, "")
	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + SWAP_V3_POSITION, header, &result)
	if err != nil {
		return nil, err
	}
	var ret []FuturePosition
	for _, item := range result {
		if item.MarginMode == "fixed" {
			panic("Fixed margin mode not supported")
		}
		for _, p := range item.Holding {
			ret = append(ret, *p.ToFuturePosition())
		}
	}

	return ret, err
}

func (ok *OKExV3_SWAP) GetInstrumentPosition(instrumentId string) ([]FuturePosition, error) {
	var result struct {
		MarginMode string `json:"margin_mode"`
		Holding []V3_SWAPPosition
	}
	reqPath := fmt.Sprintf(SWAP_V3_INSTRUMENT_POSITION, instrumentId)
	header := ok.buildHeader("GET", reqPath, "")
	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + reqPath, header, &result)
	if err != nil {
		return nil, err
	}

	if result.MarginMode == "fixed" {
		panic("fixed margin mode not supported")
	}

	var ret []FuturePosition
	for _, p := range result.Holding {
		ret = append(ret, *p.ToFuturePosition())
	}

	return ret, err
}


func (ok *OKExV3_SWAP) GetInstrumentTicker(instrumentId string) (*Ticker, error) {
	url := SWAP_V3_API_BASE_URL + SWAP_V3_INSTRUMENT_TICKER
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

	ticker := new(Ticker)
	ticker.Date = uint64(V3_SWAPParseDate(tickerMap["timestamp"].(string)))
	ticker.Buy, _ = strconv.ParseFloat(tickerMap["best_bid"].(string), 64)
	ticker.Sell, _ = strconv.ParseFloat(tickerMap["best_ask"].(string), 64)
	ticker.Last, _ = strconv.ParseFloat(tickerMap["last"].(string), 64)
	ticker.High, _ = strconv.ParseFloat(tickerMap["high_24h"].(string), 64)
	ticker.Low, _ = strconv.ParseFloat(tickerMap["low_24h"].(string), 64)
	ticker.Vol, _ = strconv.ParseFloat(tickerMap["volume_24h"].(string), 64)

	return ticker, nil
}

func (ok *OKExV3_SWAP) GetInstrumentIndex(instrumentId string) (float64, error) {
	resp, err := ok.client.Get(fmt.Sprintf(SWAP_V3_API_BASE_URL+SWAP_V3_INSTRUMENT_INDEX, instrumentId))
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

type V3_SWAPCurrencyInfo struct {
	Equity string
	Margin string
	MarginMode string		`json:"margin_mode"`
	MarginRatio string 		`json:"margin_ratio"`
	RealizedPnl string 		`json:"realized_pnl"`
	TotalAvailBalance string `json:"total_avail_balance"`
	UnrealizedPnl string 	`json:"unrealized_pnl"`
	InstrumentId string 	`json:"instrument_id"`
}

func (this *V3_SWAPCurrencyInfo) ToFutureSubAccount(currency Currency) *FutureSubAccount {
	a := new(FutureSubAccount)

	a.Currency = currency
	a.AccountRights, _ = strconv.ParseFloat(this.Equity, 64)
	a.KeepDeposit, _ = strconv.ParseFloat(this.TotalAvailBalance, 64)
	a.RiskRate, _ = strconv.ParseFloat(this.MarginRatio, 64)
	a.ProfitReal, _ = strconv.ParseFloat(this.RealizedPnl, 64)
	a.ProfitUnreal, _ = strconv.ParseFloat(this.UnrealizedPnl, 64)
	return a
}

type V3_SWAPAccountsResponse struct {
	Info []V3_SWAPCurrencyInfo `json:"info"`
	Result     bool `json:"result,bool"`
	Error_code int  `json:"error_code"`
}

func (ok *OKExV3_SWAP) GetAccount() (*FutureAccount, error) {
	var resp *V3_SWAPAccountsResponse
	header := ok.buildHeader("GET", SWAP_V3_ACCOUNTS, "")
	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + SWAP_V3_ACCOUNTS, header, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Result && resp.Error_code > 0 {
		return nil, fmt.Errorf("error code: %d", resp.Error_code)
	}

	account := new(FutureAccount)
	account.FutureSubAccounts = make(map[Currency]FutureSubAccount)

	for _, item := range resp.Info {
		currency := V3SWAPInstrumentId2Currency(item.InstrumentId)
		account.FutureSubAccounts[currency] = *item.ToFutureSubAccount(currency)
	}

	return account, nil
}


func (ok *OKExV3_SWAP) GetInstrumentAccount(instrumentId string) (*FutureSubAccount, error) {
	var resp *struct {
		Info *V3_SWAPCurrencyInfo
	}
	reqUrl := fmt.Sprintf(SWAP_V3_INSTRUMENT_ACCOUNTS, instrumentId)
	header := ok.buildHeader("GET", reqUrl, "")
	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Info == nil {
		return nil, nil
	}

	currency := V3SWAPInstrumentId2Currency(instrumentId)
	return resp.Info.ToFutureSubAccount(currency), nil
}

func (ok *OKExV3_SWAP) PlaceFutureOrder(clientOid string, instrumentId string, price, size string, _type, orderType, matchPrice, leverage int) (string, error) {
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

	header := ok.buildHeader("POST", SWAP_V3_ORDER, data)

	placeOrderUrl := SWAP_V3_API_BASE_URL + SWAP_V3_ORDER
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return "", err
	}

	var ret *struct {
		OrderId string `json:"order_id"`
		ClientOid string `json:"client_oid"`
		ErrorCode string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Result string `json:"result"`
	}

	err = json.Unmarshal(body, &ret)
	if err != nil {
		return "", err
	}

	if ret.ErrorCode != "0" {
		return "", fmt.Errorf("error code: %s", ret.ErrorCode)
	}

	return ret.OrderId, nil
}

func (ok *OKExV3_SWAP) FutureCancelOrder(instrumentId, orderId string) error {
	reqUrl := fmt.Sprintf(SWAP_V3_CANCEL_ORDER, instrumentId, orderId)

	header := ok.buildHeader("POST", reqUrl, "")

	reqPath := SWAP_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, "", header)

	var resp struct {
		Result string
		ErrorCode string 	`json:"error_code"`
		ErrorMessage string `json:"error_message"`
		OrderId string 		`json:"order_id"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	print(string(body))
	if resp.ErrorCode != "" {
		return fmt.Errorf("error code: %s", resp.ErrorCode)
	}

	return nil
}

type V3SwapOrderItem struct {
	ClientOid string 	`json:"client_oid"`
	Type string 		`json:"type"`
	OrderType string 	`json:"order_type"`
	Price string 		`json:"price"`
	Size string 		`json:"size"`
	MatchPrice string 	`json:"match_price"`
}

type V3SwapBatchPlaceOrderReq struct {
	InstrumentId string 		`json:"instrument_id"`
	OrdersData []V3SwapOrderItem `json:"order_data"`
}

type V3SwapBatchPlaceOrderRespItem struct {
	ErrorMessage string 		`json:"error_message"`
	ErrorCode string 			`json:"error_code"`
	ClientOid string 			`json:"client_oid"`
	OrderId string 				`json:"order_id"`
}

func (ok *OKExV3_SWAP) PlaceFutureOrders(req V3SwapBatchPlaceOrderReq) ([]V3SwapBatchPlaceOrderRespItem, error) {
	bytes, _ := json.Marshal(req)
	data := string(bytes)

	header := ok.buildHeader("POST", SWAP_V3_ORDERS, data)

	placeOrderUrl := SWAP_V3_API_BASE_URL + SWAP_V3_ORDERS
	body, err := HttpPostJson(ok.client, placeOrderUrl, data, header)

	if err != nil {
		return nil, err
	}

	var ret *struct {
		Result string `json:"result"`
		Data []V3SwapBatchPlaceOrderRespItem		`json:"order_info"`
	}

	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if ret.Result != "true" {
		return nil, fmt.Errorf("place order fail, body: %s", string(body))
	}

	return ret.Data, nil
}

func (ok *OKExV3_SWAP) FutureCancelOrders(instrumentId string, orderIds []string) error {
	bytes, _ := json.Marshal(map[string]interface{} {
		"ids": orderIds,
	})

	reqUrl := fmt.Sprintf(SWAP_V3_CANCEL_ORDERS, instrumentId)

	header := ok.buildHeader("POST", reqUrl, string(bytes))

	reqPath := SWAP_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, string(bytes), header)
	if err != nil {
		return err
	}

	var resp struct {
		Result string 		`json:"result"`
		ErrorCode string 	`json:"error_code"`
		ErrorMessage string `json:"error_message"`
		OrderIds []string 	`json:"ids"`
		InstrumentId string `json:"instrument_id"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.ErrorCode != "" {
		return fmt.Errorf("error code: %s", resp.ErrorCode)
	}

	return nil
}

type V3_SWAPOrderInfo struct {
	InstrumentId string 	`json:"instrument_id"`
	Size string
	Timestamp string
	FilledQty string 		`json:"filled_qty"`
	Fee string
	OrderId string 			`json:"order_id"`
	Price string
	PriceAvg string 		`json:"price_avg"`
	Status string
	Type string
	ContractVal string 		`json:"contract_val"`
	Leverage string
}

func (this *V3_SWAPOrderInfo) ToFutureOrder() *FutureOrder {
	if this.OrderId == "" {
		return nil
	}
	o := new(FutureOrder)
	o.Price, _ = strconv.ParseFloat(this.Price, 64)
	o.Amount, _  = strconv.ParseFloat(this.Size, 64)
	o.AvgPrice, _ = strconv.ParseFloat(this.PriceAvg, 64)
	o.DealAmount, _ = strconv.ParseFloat(this.FilledQty, 64)
	o.OrderID2 = this.OrderId
	o.OrderTime = V3_SWAPParseDate(this.Timestamp)
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
	o.Currency = V3SWAPInstrumentId2CurrencyPair(this.InstrumentId)
	o.OType, _ = strconv.Atoi(this.Type)
	o.Fee, _ = strconv.ParseFloat(this.Fee, 64)
	o.LeverRate, _ = strconv.Atoi(this.Leverage)
	o.ContractName = this.InstrumentId
	return o
}

func (ok *OKExV3_SWAP) GetInstrumentOrders(instrumentId string, status, from, to, limit string) ([]FutureOrder, error) {
	reqUrl := fmt.Sprintf(SWAP_V3_INSTRUMENT_ORDERS, instrumentId)
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
		reqUrl += "?" + strings.Join(params, "&")
	}

	header := ok.buildHeader("GET", reqUrl, "")

	var resp *struct{
		Orders []V3_SWAPOrderInfo		`json:"order_info"`
	}

	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	ret := make([]FutureOrder, len(resp.Orders))
	for i, o := range resp.Orders {
		ret[i] = *o.ToFutureOrder()
	}

	return ret, nil
}

func (ok *OKExV3_SWAP) GetInstrumentOrder(instrumentId string, orderId string) (*FutureOrder, error) {
	reqUrl := fmt.Sprintf(SWAP_V3_ORDER_INFO, instrumentId, orderId)
	header := ok.buildHeader("GET", reqUrl, "")

	var resp *V3_SWAPOrderInfo

	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ToFutureOrder(), nil
}

type V3FutureLedger struct {
	Amount string			`json:"amount"`
	Fee string				`json:"fee"`
	InstrumentId string 	`json:"instrument_id"`
	LedgerId string 		`json:"ledger_id"`
	Timestamp string		`json:"timestamp"`
	Type string				`json:"type"`
}

func (ok *OKExV3_SWAP) GetLedger(instrumentId string, from, to, limit string) ([]V3FutureLedger, error) {
	reqUrl := fmt.Sprintf("/api/swap/v3/accounts/%s/ledger", instrumentId)
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

	var resp []V3FutureLedger

	err := HttpGet4(ok.client, SWAP_V3_API_BASE_URL + reqUrl, header, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
