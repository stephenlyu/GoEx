package binancefuture

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"sync"
	"github.com/shopspring/decimal"
)

const (
	API_BASE_URL = "https://fapi.binance.com/"
	API_V1       = API_BASE_URL + "fapi/v1/"

	EXCHANGE_INFO_URI 	   = "exchangeInfo"
	TICKER_URI             = "ticker/24hr?symbol=%s"
	TRADES_URI            = "trades?symbol=%s&limit=1"
	DEPTH_URI              = "depth?symbol=%s&limit=%d"
	ACCOUNT_URI            = "account?"
	ORDER_URI              = "order?"
	UNFINISHED_ORDERS_INFO = "openOrders?"
)

type Binance struct {
	accessKey,
	secretKey          string
	httpClient         *http.Client

	wsData             *WsConn
	wsLock             sync.Mutex
	wsLoginHandle      func(err error)
	wsDepthHandleMap   map[string]func(*DepthDecimal)
	wsTradeHandleMap   map[string]func(string, []TradeDecimal)
	wsAccountHandleMap map[string]func(*SubAccountDecimal)
	wsOrderHandleMap   map[string]func([]OrderDecimal)
	errorHandle        func(error)

	depthManagers 	   map[string]*DepthManager
}

func (bn *Binance) buildParamsSigned(postForm *url.Values) error {
	postForm.Set("recvWindow", "6000000")
	tonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	postForm.Set("timestamp", tonce)
	payload := postForm.Encode()
	sign, _ := GetParamHmacSHA256Sign(bn.secretKey, payload)
	postForm.Set("signature", sign)
	return nil
}

func New(client *http.Client, api_key, secret_key string) *Binance {
	return &Binance{
		accessKey: api_key,
		secretKey: secret_key,
		httpClient: client}
}

func (bn *Binance) GetExchangeName() string {
	return BINANCE
}

func (bn *Binance) GetExchangeInfo() (*Exchange, error) {
	tickerUri := API_V1 + EXCHANGE_INFO_URI
	var exchange *Exchange
	err := HttpGet4(bn.httpClient, tickerUri, nil, &exchange)

	if err != nil {
		log.Println("GetExchangeInfo error:", err)
		return nil, err
	}

	if exchange.Code != 0 {
		return nil, fmt.Errorf("error_code: %d", exchange.Code)
	}

	return exchange, nil
}

func (bn *Binance) GetTicker(currency CurrencyPair) (*TickerDecimal, error) {
	currency2 := bn.adaptCurrencyPair(currency)
	tickerUri := API_V1 + fmt.Sprintf(TICKER_URI, currency2.ToSymbol(""))

	var resp struct {
		Code int
		CloseTime int64
		LastPrice decimal.Decimal
		LowPrice decimal.Decimal
		HighPrice decimal.Decimal
		Volume decimal.Decimal
	}

	err := HttpGet4(bn.httpClient, tickerUri, nil, &resp)

	if err != nil {
		log.Println("GetTicker error:", err)
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("error_code: %d", resp.Code)
	}

	t := new(TickerDecimal)
	t.Date = uint64(resp.CloseTime)
	t.Low = resp.LowPrice
	t.High = resp.HighPrice
	t.Last = resp.LastPrice
	t.Vol = resp.Volume

	return t, nil
}

func (bn *Binance) GetDepthInternal(size int, currencyPair CurrencyPair) (*DepthData, error) {
	if size < 5 {
		size = 5
	}
	currencyPair2 := bn.adaptCurrencyPair(currencyPair)

	apiUrl := fmt.Sprintf(API_V1 + DEPTH_URI, currencyPair2.ToSymbol(""), size)

	var data DepthData

	err := HttpGet4(bn.httpClient, apiUrl, nil, &data)
	if err != nil {
		log.Println("GetDepth error:", err)
		return nil, err
	}
	return &data, nil
}

func (bn *Binance) GetDepth(size int, currencyPair CurrencyPair) (*DepthDecimal, error) {
	data, err := bn.GetDepthInternal(size, currencyPair)
	if err != nil {
		return nil, err
	}

	if data.Code != 0 {
		return nil, fmt.Errorf("error_code: %d", data.Code)
	}

	depth := new(DepthDecimal)
	depth.Pair = currencyPair

	depth.AskList = make([]DepthRecordDecimal, len(data.Asks), len(data.Asks))
	for i, o := range data.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(data.Bids), len(data.Bids))
	for i, o := range data.Bids {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

func (bn *Binance) GetTrades(currencyPair CurrencyPair) ([]TradeDecimal, error) {
	url := fmt.Sprintf(API_V1 + TRADES_URI, currencyPair.ToSymbol(""))
	println(url)
	var data []struct {
		Qty decimal.Decimal
		Price  decimal.Decimal
		Id     decimal.Decimal
		Time   int64
		IsBuyerMaker bool
	}

	err := HttpGet4(bn.httpClient, url, nil, &data)
	if err != nil {
		log.Println("GetTrades error:", err)
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
		t.Tid = o.Id.IntPart()
		t.Date = o.Time
	}

	return trades, nil
}

func (bn *Binance) placeOrder(amount, price string, pair CurrencyPair, orderType, orderSide string) (*Order, error) {
	pair = bn.adaptCurrencyPair(pair)
	path := API_V1 + ORDER_URI
	params := url.Values{}
	params.Set("symbol", pair.ToSymbol(""))
	params.Set("side", orderSide)
	params.Set("type", orderType)

	params.Set("quantity", amount)
	params.Set("type", "LIMIT")
	params.Set("timeInForce", "GTC")

	switch orderType {
	case "LIMIT":
		params.Set("price", price)
	}

	bn.buildParamsSigned(&params)

	resp, err := HttpPostForm2(bn.httpClient, path, params,
		map[string]string{"X-MBX-APIKEY": bn.accessKey})
	//log.Println("resp:", string(resp), "err:", err)
	if err != nil {
		return nil, err
	}

	respmap := make(map[string]interface{})
	err = json.Unmarshal(resp, &respmap)
	if err != nil {
		log.Println(string(resp))
		return nil, err
	}

	orderId := ToInt(respmap["orderId"])
	if orderId <= 0 {
		return nil, errors.New(string(resp))
	}

	side := BUY
	if orderSide == "SELL" {
		side = SELL
	}

	return &Order{
		Currency:   pair,
		OrderID:    orderId,
		OrderID2:   fmt.Sprint(orderId),
		Price:      ToFloat64(price),
		Amount:     ToFloat64(amount),
		DealAmount: 0,
		AvgPrice:   0,
		Side:       TradeSide(side),
		Status:     ORDER_UNFINISH,
		OrderTime:  int(time.Now().Unix())}, nil
}

func (bn *Binance) GetAccount() (*Account, error) {
	params := url.Values{}
	bn.buildParamsSigned(&params)
	path := API_V1 + ACCOUNT_URI + params.Encode()
	respmap, err := HttpGet2(bn.httpClient, path, map[string]string{"X-MBX-APIKEY": bn.accessKey})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	//log.Println("respmap:", respmap)
	if _, isok := respmap["code"]; isok == true {
		return nil, errors.New(respmap["msg"].(string))
	}
	acc := Account{}
	acc.Exchange = bn.GetExchangeName()
	acc.SubAccounts = make(map[Currency]SubAccount)

	balances := respmap["balances"].([]interface{})
	for _, v := range balances {
		//log.Println(v)
		vv := v.(map[string]interface{})
		currency := NewCurrency(vv["asset"].(string), "").AdaptBccToBch()
		acc.SubAccounts[currency] = SubAccount{
			Currency:     currency,
			Amount:       ToFloat64(vv["free"]),
			ForzenAmount: ToFloat64(vv["locked"]),
		}
	}

	return &acc, nil
}

func (bn *Binance) LimitBuy(amount, price string, currencyPair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, currencyPair, "LIMIT", "BUY")
}

func (bn *Binance) LimitSell(amount, price string, currencyPair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, currencyPair, "LIMIT", "SELL")
}

func (bn *Binance) MarketBuy(amount, price string, currencyPair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, currencyPair, "MARKET", "BUY")
}

func (bn *Binance) MarketSell(amount, price string, currencyPair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, currencyPair, "MARKET", "SELL")
}

func (bn *Binance) CancelOrder(orderId string, currencyPair CurrencyPair) (bool, error) {
	currencyPair = bn.adaptCurrencyPair(currencyPair)
	path := API_V1 + ORDER_URI
	params := url.Values{}
	params.Set("symbol", currencyPair.ToSymbol(""))
	params.Set("orderId", orderId)

	bn.buildParamsSigned(&params)

	resp, err := HttpDeleteForm(bn.httpClient, path, params, map[string]string{"X-MBX-APIKEY": bn.accessKey})

	//log.Println("resp:", string(resp), "err:", err)
	if err != nil {
		return false, err
	}

	respmap := make(map[string]interface{})
	err = json.Unmarshal(resp, &respmap)
	if err != nil {
		log.Println(string(resp))
		return false, err
	}

	orderIdCanceled := ToInt(respmap["orderId"])
	if orderIdCanceled <= 0 {
		return false, errors.New(string(resp))
	}

	return true, nil
}

func (bn *Binance) GetOneOrder(orderId string, currencyPair CurrencyPair) (*Order, error) {
	params := url.Values{}
	currencyPair = bn.adaptCurrencyPair(currencyPair)
	params.Set("symbol", currencyPair.ToSymbol(""))
	if orderId != "" {
		params.Set("orderId", orderId)
	}
	params.Set("orderId", orderId)

	bn.buildParamsSigned(&params)
	path := API_V1 + ORDER_URI + params.Encode()

	respmap, err := HttpGet2(bn.httpClient, path, map[string]string{"X-MBX-APIKEY": bn.accessKey})
	//log.Println(respmap)
	if err != nil {
		return nil, err
	}
	status := respmap["status"].(string)
	side := respmap["side"].(string)

	ord := Order{}
	ord.Currency = currencyPair
	ord.OrderID = ToInt(orderId)
	ord.OrderID2 = orderId

	if side == "SELL" {
		ord.Side = SELL
	} else {
		ord.Side = BUY
	}

	switch status {
	case "FILLED":
		ord.Status = ORDER_FINISH
	case "PARTIALLY_FILLED":
		ord.Status = ORDER_PART_FINISH
	case "CANCELED":
		ord.Status = ORDER_CANCEL
	case "PENDING_CANCEL":
		ord.Status = ORDER_CANCEL_ING
	case "REJECTED":
		ord.Status = ORDER_REJECT
	}

	ord.Amount = ToFloat64(respmap["origQty"].(string))
	ord.Price = ToFloat64(respmap["price"].(string))
	ord.DealAmount = ToFloat64(respmap["executedQty"])
	ord.AvgPrice = ord.Price // response no avg price ， fill price

	return &ord, nil
}

func (bn *Binance) GetUnfinishOrders(currencyPair CurrencyPair) ([]Order, error) {
	params := url.Values{}
	currencyPair = bn.adaptCurrencyPair(currencyPair)
	params.Set("symbol", currencyPair.ToSymbol(""))

	bn.buildParamsSigned(&params)
	path := API_V1 + UNFINISHED_ORDERS_INFO + params.Encode()

	respmap, err := HttpGet3(bn.httpClient, path, map[string]string{"X-MBX-APIKEY": bn.accessKey})
	//log.Println("respmap", respmap, "err", err)
	if err != nil {
		return nil, err
	}

	orders := make([]Order, 0)
	for _, v := range respmap {
		ord := v.(map[string]interface{})
		side := ord["side"].(string)
		orderSide := SELL
		if side == "BUY" {
			orderSide = BUY
		}

		orders = append(orders, Order{
			OrderID:   ToInt(ord["orderId"]),
			OrderID2:  fmt.Sprint(ToInt(ord["id"])),
			Currency:  currencyPair,
			Price:     ToFloat64(ord["price"]),
			Amount:    ToFloat64(ord["origQty"]),
			Side:      TradeSide(orderSide),
			Status:    ORDER_UNFINISH,
			OrderTime: ToInt(ord["time"])})
	}
	return orders, nil
}

func (ba *Binance) adaptCurrencyPair(pair CurrencyPair) CurrencyPair {
	return pair.AdaptBchToBcc().AdaptUsdToUsdt()
}
