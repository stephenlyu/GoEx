package gateiospot

import (
	"sync"
	"net/http"
	. "github.com/stephenlyu/GoEx"
	"io/ioutil"
	"encoding/json"
	"strings"
	"github.com/shopspring/decimal"
	"errors"
	"fmt"
	"sort"
)

const (
	API_BASE_URL = "https://data.gateio.io"

	PAIRS = "/api2/1/pairs"
	MARKET_INFO = "/api2/1/marketinfo"
	MARKET_LIST = "/api2/1/marketlist"
	TICKERS = "/api2/1/tickers"
	TICKER = "/api2/1/ticker/%s"
	ORDER_BOOKS = "/api2/1/orderBooks"
	ORDER_BOOK = "/api2/1/orderBook/%s"
	TRADE_HISTORY = "/api2/1/tradeHistory/%s"
	BALANCES = "/api2/1/private/balances"
	PRIVATE_BUY = "/api2/1/private/buy"
	PRIVATE_SELL = "/api2/1/private/sell"
	CANCEL_ORDER = "/api2/1/private/cancelOrder"
	CANCEL_ORDERS = "/api2/1/private/cancelOrders"
	CANCEL_ALL_ORDERS = "/api2/1/private/cancelAllOrders"
	GET_ORDER = "/api2/1/private/getOrder"
	OPEN_ORDERS = "/api2/1/private/openOrders"
)

type GateIOSpot struct {
	apiKey,
	apiSecretKey string
	client            *http.Client

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(CurrencyPair, []TradeDecimal)
	wsAccountHandleMap  map[string]func(*SubAccountDecimal)
	wsOrderHandleMap  map[string]func([]OrderDecimal)
	depthManagers	 map[string]*DepthManager
}

func NewGateIOSpot(	apiKey, apiSecretKey string) *GateIOSpot {
	return &GateIOSpot{
		apiKey: apiKey,
		apiSecretKey: apiSecretKey,
		client: http.DefaultClient,
	}
}

func (this *GateIOSpot) GetPairs() ([]CurrencyPair, error) {
	resp, err := this.client.Get(API_BASE_URL + PAIRS)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var pairs []string
	err = json.Unmarshal(body, &pairs)
	if err != nil {
		return nil, err
	}

	ret := make([]CurrencyPair, len(pairs))
	for i, p := range pairs {
		ret[i] = NewCurrencyPair2(strings.ToUpper(p))
	}

	return ret, err
}

type MarketInfo struct {
	Pair CurrencyPair
	DecimalPlaces int				`json:"decimal_places"`
	MinAmount decimal.Decimal		`json:"min_amount"`
	MinAmountA decimal.Decimal		`json:"min_amount_a"`
	MinAmountB decimal.Decimal		`json:"min_amount_b"`
	Fee decimal.Decimal				`json:"fee"`
	TradeDisabled int				`json:"trade_disabled"`
}

func (this *GateIOSpot) GetMarketInfo() ([]MarketInfo, error) {
	resp, err := this.client.Get(API_BASE_URL + MARKET_INFO)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	println(string(body))
	var data struct {
		Result string
		Pairs []map[string]MarketInfo
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail")
	}

	ret := make([]MarketInfo, len(data.Pairs))
	for i, o := range data.Pairs {
		for k, v := range o {
			v.Pair = NewCurrencyPair2(strings.ToUpper(k))
			ret[i] = v
			break
		}
	}

	return ret, err
}

func (this *GateIOSpot) GetTicker(pair CurrencyPair) (*Ticker, error) {
	resp, err := this.client.Get(API_BASE_URL + fmt.Sprintf(TICKER, strings.ToLower(pair.ToSymbol("_"))))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var ticker struct {
		Result string
		Last decimal.Decimal
		LowestAsk decimal.Decimal
		HighestBid decimal.Decimal
		PercentChange decimal.Decimal
		BaseVolume decimal.Decimal
		QuoteVolume decimal.Decimal
		High24hr decimal.Decimal
		Low24hr decimal.Decimal
	}
	err = json.Unmarshal(body, &ticker)
	if err != nil {
		return nil, err
	}

	if ticker.Result != "true" {
		return nil, errors.New("fail")
	}

	ret := new(Ticker)
	ret.Buy, _ = ticker.HighestBid.Float64()
	ret.Sell, _ = ticker.LowestAsk.Float64()
	ret.High, _ = ticker.High24hr.Float64()
	ret.Low, _ = ticker.Low24hr.Float64()
	ret.Vol, _ = ticker.BaseVolume.Float64()
	ret.Last, _ = ticker.Last.Float64()
	ret.Pair = pair

	return ret, err
}

func (this *GateIOSpot) GetOrderBook(pair CurrencyPair) (*DepthDecimal, error) {
	resp, err := this.client.Get(API_BASE_URL + fmt.Sprintf(ORDER_BOOK, strings.ToLower(pair.ToSymbol("_"))))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data struct {
		Result string
		Asks [][]decimal.Decimal
		Bids [][]decimal.Decimal
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail")
	}

	ret := new(DepthDecimal)
	ret.Pair = pair

	ret.AskList = make(DepthRecordsDecimal, len(data.Asks))
	for i, v := range data.Asks {
		ret.AskList[i] = DepthRecordDecimal{
			Price: v[0],
			Amount: v[1],
		}
	}
	sort.Slice(ret.AskList, func(i,j int) bool {
		return ret.AskList[i].Price.LessThan(ret.AskList[j].Price)
	})

	ret.BidList = make(DepthRecordsDecimal, len(data.Bids))
	for i, v := range data.Bids {
		ret.BidList[i] = DepthRecordDecimal{
			Price: v[0],
			Amount: v[1],
		}
	}

	return ret, err
}

func (this *GateIOSpot) GetTrades(pair CurrencyPair) ([]TradeDecimal, error) {
	resp, err := this.client.Get(API_BASE_URL + fmt.Sprintf(TRADE_HISTORY, strings.ToLower(pair.ToSymbol("_"))))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data struct {
		Result string
		Data []struct {
			TradeID decimal.Decimal
			Date string
			Timestamp decimal.Decimal
			Type string
			Rate decimal.Decimal
			Amount decimal.Decimal
			Total decimal.Decimal
		}
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail")
	}

	ret := make([]TradeDecimal, len(data.Data))
	for i, o := range data.Data {
		t := TradeDecimal{}

		t.Amount = o.Amount
		t.Date = o.Timestamp.IntPart()
		t.Price = o.Rate
		t.Tid = o.TradeID.IntPart()
		t.Type = o.Type

		ret[i] = t
	}

	return ret, err
}

func (this *GateIOSpot) buildHeader(body string) map[string]string {
	signature, _ := GetParamHmacSHA512Sign(this.apiSecretKey, body)
	return map[string]string {
		"key": this.apiKey,
		"sign": signature,
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *GateIOSpot) GetAccount() (*AccountDecimal, error) {
	header := this.buildHeader("")
	body, err := HttpPostForm3(this.client, API_BASE_URL + BALANCES, "", header)
	if err != nil {
		return nil, err
	}

	var data struct {
		Result string
		Available map[string]decimal.Decimal
		Locked map[string]decimal.Decimal
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail")
	}

	ret := new(AccountDecimal)
	ret.SubAccounts = make(map[Currency]SubAccountDecimal)

	for key, available := range data.Available {
		locked, _ := data.Locked[key]
		currency := Currency{Symbol: key}
		sa := SubAccountDecimal{
			Currency: currency,
			AvailableAmount: available,
			FrozenAmount: locked,
			Amount: available.Add(locked),
		}
		ret.SubAccounts[currency] = sa
	}

	return ret, err
}

func (this *GateIOSpot) PlaceOrder(side string, pair CurrencyPair, price, amount decimal.Decimal) (string, error) {
	var param string = "currencyPair=" + strings.ToLower(pair.ToSymbol("_")) + "&rate=" + price.String() + "&amount=" + amount.String()

	var reqUrl string
	if side == "sell" {
		reqUrl = PRIVATE_SELL
	} else if side == "buy" {
		reqUrl = PRIVATE_BUY
	} else {
		panic("Bad side " + side)
	}

	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + reqUrl, param, header)
	if err != nil {
		return "", err
	}
	var data struct {
		Result string
		OrderNumber decimal.Decimal
		Rate decimal.Decimal
		LeftAmount decimal.Decimal
		FilledAmount decimal.Decimal
		FilledRate decimal.Decimal
		Message string
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	if data.Result != "true" {
		return "", errors.New("fail, error:" + data.Message)
	}

	return data.OrderNumber.String(), err
}

func (this *GateIOSpot) CancelOrder(pair CurrencyPair, orderId string) error {
	var param string = "orderNumber=" + orderId + "&currencyPair=" + strings.ToLower(pair.ToSymbol("_"))

	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + CANCEL_ORDER, param, header)
	if err != nil {
		return err
	}
	var data struct {
		Result bool
		Message string
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	if !data.Result {
		if strings.Contains(data.Message, "already finished") {
			return nil
		}
		return errors.New("fail, error:" + data.Message)
	}

	return err
}

func (this *GateIOSpot) CancelOrders(pair CurrencyPair, orderIds []string) error {
	symbol := strings.ToLower(pair.ToSymbol("_"))
	var params []map[string]string
	for _, orderId := range orderIds {
		params = append(params, map[string]string{"orderNumber": orderId, "currencyPair": symbol})
	}
	bytes, _ := json.Marshal(params)
	param := string(bytes)

	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + CANCEL_ORDERS, param, header)
	if err != nil {
		return err
	}
	var data struct {
		Result bool
		Message string
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	if !data.Result {
		return errors.New("fail, error:" + data.Message)
	}

	return err
}

const (
	CancelAllOrdersTypeSell = "0"
	CancelAllOrdersTypeBuy = "1"
	CancelAllOrdersTypeAll = "-1"
)

func (this *GateIOSpot) CancelAllOrders(pair CurrencyPair, types string) error {
	var param string = "types=" + types + "&currencyPair=" + strings.ToLower(pair.ToSymbol("_"))

	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + CANCEL_ALL_ORDERS, param, header)
	if err != nil {
		return err
	}
	println(string(body))
	var data struct {
		Result bool
		Message string
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	if !data.Result {
		return errors.New("fail, error:" + data.Message)
	}

	return err
}

func (this *GateIOSpot) translateOrderStatus(status string, amount decimal.Decimal) TradeStatus {
	switch status {
	case "open":
		if amount.Equal(decimal.Zero) {
			return ORDER_UNFINISH
		}
		return ORDER_PART_FINISH
	case "cancelled":
		return ORDER_CANCEL
	case "closed":
		return ORDER_FINISH
	}
	panic(fmt.Errorf("bad order status %s", status))
	return ORDER_UNFINISH
}

func (this *GateIOSpot) translateType(_type string) TradeSide {
	switch _type {
	case "buy":
		return BUY
	case "sell":
		return SELL
	}
	panic(fmt.Errorf("bad order type %s", _type))
	return BUY
}

func (this *GateIOSpot) GetOrder(pair CurrencyPair, orderId string) (*OrderDecimal, error) {
	var param string = "orderNumber=" + orderId + "&currencyPair=" + strings.ToLower(pair.ToSymbol("_"))
	println(param)
	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + GET_ORDER, param, header)
	if err != nil {
		return nil, err
	}
	var data struct {
		Result string
		Message string
		Order struct {
			OrderNumber string
			Status string
			CurrencyPair string
			Type string
			FilledRate decimal.Decimal
			FilledAmount decimal.Decimal
			InitialRate decimal.Decimal
			InitialAmount decimal.Decimal
			Timestamp decimal.Decimal
			FeeValue decimal.Decimal
			FeeCurrency string
		}
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail, error:" + data.Message)
	}

	ret := new(OrderDecimal)
	o := &data.Order
	ret.OrderID2 = o.OrderNumber
	ret.Status = this.translateOrderStatus(o.Status, o.FilledAmount)
	ret.Currency = pair
	ret.Side = this.translateType(o.Type)
	ret.Price = o.InitialRate
	ret.Amount = o.InitialAmount
	ret.AvgPrice = o.FilledRate
	ret.DealAmount = o.FilledAmount
	ret.Fee = o.FeeValue
	ret.FeeCurrency = o.FeeCurrency
	ret.Timestamp = o.Timestamp.IntPart()

	return ret, err
}

func (this *GateIOSpot) GetOpenOrders(pair CurrencyPair) ([]OrderDecimal, error) {
	var param string = "currencyPair=" + strings.ToLower(pair.ToSymbol("_"))

	header := this.buildHeader(param)
	body, err := HttpPostForm3(this.client, API_BASE_URL + OPEN_ORDERS, param, header)
	if err != nil {
		return nil, err
	}
	var data struct {
		Result string
		Message string
		Orders []struct {
			   OrderNumber decimal.Decimal
			   Type string
			   InitialRate decimal.Decimal
			   InitialAmount decimal.Decimal
			   FilledRate decimal.Decimal
			   FilledAmount decimal.Decimal
			   CurrencyPair string
			   Timestamp decimal.Decimal
			   Status string
		   }
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Result != "true" {
		return nil, errors.New("fail, error:" + data.Message)
	}

	ret := make([]OrderDecimal, len(data.Orders))
	for i, o := range data.Orders {
		r := &ret[i]
		r.OrderID2 = o.OrderNumber.String()
		r.Status = this.translateOrderStatus(o.Status, o.FilledAmount)
		r.Currency = pair
		r.Side = this.translateType(o.Type)
		r.Price = o.InitialRate
		r.Amount = o.InitialAmount
		r.AvgPrice = o.FilledRate
		r.DealAmount = o.FilledAmount
		r.Timestamp = o.Timestamp.IntPart()
	}

	return ret, err
}
