package biki

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

// consts
const (
	OrerSell = "SELL"
	OrderBuy = "BUY"

	OrderTypeLimit  = 1
	OrderTypeMarket = 2

	OrderTypeLimitStr  = "1"
	OrderTypeMarketStr = "2"
)

// Urls
const (
	apiBaseURL     = "http://openapi.biki.com"
	commonSymbols  = "/open/api/common/symbols"
	getTicker      = "/open/api/get_ticker?symbol=%s"
	getMarketDepth = "/open/api/market_dept?symbol=%s&type=step0"
	getTrades      = "/open/api/get_trades?symbol=%s"
	account        = "/open/api/user/account"
	createOrder    = "/open/api/create_order"
	massReplace    = "/open/api/mass_replaceV2"
	cancelOrder    = "/open/api/cancel_order"
	newOrder       = "/open/api/new_order"
	orderInfo      = "/open/api/order_info"
	allOrder       = "/open/api/all_order"
)

// Biki Biki api
type Biki struct {
	APIKey    string
	SecretKey string
	client    *http.Client

	symbolNameMap map[string]string

	ws               *goex.WsConn
	createWsLock     sync.Mutex
	wsDepthHandleMap map[string]func(*goex.DepthDecimal)
	wsTradeHandleMap map[string]func(string, []goex.TradeDecimal)
	errorHandle      func(error)
	wsSymbolMap      map[string]string
}

// NewBiki Biki constructor
func NewBiki(APIKey string, SecretKey string) *Biki {
	biki := new(Biki)
	biki.APIKey = APIKey
	biki.SecretKey = SecretKey
	biki.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	biki.symbolNameMap = make(map[string]string)
	return biki
}

func (biki *Biki) getPairByName(name string) string {
	name = strings.ToUpper(name)
	c, ok := biki.symbolNameMap[name]
	if ok {
		return c
	}

	var err error
	var l []Symbol
	for i := 0; i < 5; i++ {
		l, err = biki.GetSymbols()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}

	for _, o := range l {
		biki.symbolNameMap[strings.ToUpper(o.Symbol)] = fmt.Sprintf("%s_%s", o.BaseCoin, o.CountCoin)
	}
	c, ok = biki.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

// GetSymbols Get symbols
func (biki *Biki) GetSymbols() ([]Symbol, error) {
	url := apiBaseURL + commonSymbols
	resp, err := biki.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Data []Symbol
		Msg  string
		Code decimal.Decimal
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	for i := range data.Data {
		s := &data.Data[i]
		s.Symbol = strings.ToUpper(fmt.Sprintf("%s_%s", s.BaseCoin, s.CountCoin))
	}

	return data.Data, nil
}

func (biki *Biki) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

// GetTicker Get ticker
func (biki *Biki) GetTicker(symbol string) (*goex.TickerDecimal, error) {
	symbol = biki.transSymbol(symbol)
	url := apiBaseURL + getTicker
	resp, err := biki.client.Get(fmt.Sprintf(url, symbol))
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
		Data struct {
			High decimal.Decimal
			Vol  decimal.Decimal
			Last decimal.Decimal
			Low  decimal.Decimal
			Buy  decimal.Decimal
			Sell decimal.Decimal
			Time int64
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data.Data

	ticker := new(goex.TickerDecimal)
	ticker.Date = uint64(r.Time)
	ticker.Buy = r.Buy
	ticker.Sell = r.Sell
	ticker.Last = r.Last
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol

	return ticker, nil
}

// GetDepth Get depth
func (biki *Biki) GetDepth(symbol string) (*goex.DepthDecimal, error) {
	inputSymbol := symbol
	symbol = biki.transSymbol(symbol)
	url := fmt.Sprintf(apiBaseURL+getMarketDepth, symbol)
	resp, err := biki.client.Get(url)
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
		Data struct {
			Tick struct {
				Asks [][]decimal.Decimal
				Bids [][]decimal.Decimal
			}
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	r := data.Data.Tick

	depth := new(goex.DepthDecimal)
	depth.Pair = goex.NewCurrencyPair2(inputSymbol)

	depth.AskList = make([]goex.DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]goex.DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

// GetTrades Get trades
func (biki *Biki) GetTrades(symbol string) ([]goex.TradeDecimal, error) {
	symbol = biki.transSymbol(symbol)
	url := fmt.Sprintf(apiBaseURL+getTrades, symbol)
	resp, err := biki.client.Get(url)
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
		Data []struct {
			Amount decimal.Decimal
			Price  decimal.Decimal
			ID     int64
			Type   string
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	var trades = make([]goex.TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Tid = o.ID
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Type
	}

	return trades, nil
}

func (biki *Biki) signData(data string) string {
	message := data + biki.SecretKey
	sign, _ := goex.GetParamMD5Sign(biki.SecretKey, message)

	return sign
}

func (biki *Biki) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	param["api_key"] = biki.APIKey
	param["time"] = timestamp

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+param[k])
	}
	data := strings.Join(parts, "")

	sign := biki.signData(data)
	param["sign"] = sign
	return param
}

func (biki *Biki) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

// GetAccount Get account
func (biki *Biki) GetAccount() ([]goex.SubAccountDecimal, error) {
	params := map[string]string{}
	params = biki.sign(params)

	url := apiBaseURL + account + "?" + biki.buildQueryString(params)

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			TotalAsset decimal.Decimal `json:"total_asset"`
			CoinList   []struct {
				Coin        string
				Normal      decimal.Decimal
				Locked      decimal.Decimal
				BtcValuatin decimal.Decimal
			} `json:"coin_list"`
		}
	}

	err := goex.HttpGet4(biki.client, url, map[string]string{}, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret []goex.SubAccountDecimal
	for _, o := range resp.Data.CoinList {
		currency := strings.ToUpper(o.Coin)
		if currency == "" {
			continue
		}
		ret = append(ret, goex.SubAccountDecimal{
			Currency:        goex.Currency{Symbol: currency},
			AvailableAmount: o.Normal,
			FrozenAmount:    o.Locked,
			Amount:          o.Normal.Add(o.Locked),
		})
	}

	return ret, nil
}

// PlaceOrder Place order
func (biki *Biki) PlaceOrder(volume decimal.Decimal, side string, _type int, symbol string, price decimal.Decimal) (string, error) {
	symbol = biki.transSymbol(symbol)
	params := map[string]string{
		"side":   side,
		"volume": volume.String(),
		"type":   strconv.Itoa(_type),
		"symbol": symbol,
		"price":  price.String(),
	}

	params = biki.sign(params)

	data := biki.buildQueryString(params)
	url := apiBaseURL + createOrder
	body, err := goex.HttpPostForm3(biki.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return "", err
	}

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			OrderID decimal.Decimal `json:"order_id"`
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 0 {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Data.OrderID.String(), nil
}

// CancelOrder Cancel order
func (biki *Biki) CancelOrder(symbol string, orderID string) error {
	symbol = biki.transSymbol(symbol)
	params := map[string]string{
		"symbol":   symbol,
		"order_id": orderID,
	}
	params = biki.sign(params)

	data := biki.buildQueryString(params)
	url := apiBaseURL + cancelOrder
	body, err := goex.HttpPostForm3(biki.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

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

	if resp.Code.IntPart() != 0 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

// MassReplace Mass replace orders
func (biki *Biki) MassReplace(symbol string, cancelOrderIDs []string, reqList []OrderReq) (orderIDs []string, placeErrors, cancelErrors []error, err error) {
	orderIDs = make([]string, len(reqList))
	placeErrors = make([]error, len(reqList))
	cancelErrors = make([]error, len(cancelOrderIDs))

	symbol = biki.transSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
	}

	if len(cancelOrderIDs) > 0 {
		massCancel, _ := json.Marshal(cancelOrderIDs)
		params["mass_cancel"] = string(massCancel)
	}

	if len(reqList) > 0 {
		massPlace, _ := json.Marshal(reqList)
		println(string(massPlace))
		params["mass_place"] = string(massPlace)
	}

	params = biki.sign(params)

	data := biki.buildQueryString(params)
	println(data)

	url := apiBaseURL + massReplace
	var body []byte
	body, err = goex.HttpPostForm3(biki.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return
	}

	println(string(body))

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			MassPlace []struct {
				Msg      string
				Code     decimal.Decimal
				OrderIDs []decimal.Decimal `json:"order_id"`
			} `json:"mass_place"`
			MassCancel []struct {
				Msg      string
				Code     decimal.Decimal
				OrderIDs []decimal.Decimal `json:"order_id"`
			} `json:"mass_cancel"`
		}
	}

	bodyStr := strings.Replace(string(body), `,"order_id":""`, "", -1)

	err = json.Unmarshal([]byte(bodyStr), &resp)
	if err != nil {
		return
	}

	if resp.Code.IntPart() != 0 {
		err = fmt.Errorf("error code: %s", resp.Code.String())
		return
	}

	if len(cancelOrderIDs) > 0 {
		var count int
		m := make(map[string]int)
		for i, orderID := range cancelOrderIDs {
			m[orderID] = i
		}
		for _, o := range resp.Data.MassCancel {
			count += len(o.OrderIDs)
			if o.Code.IsZero() {
				continue
			}
			err1 := fmt.Errorf("error code: %s", o.Code.String())
			for _, orderID := range o.OrderIDs {
				index, ok := m[orderID.String()]
				if !ok {
					panic("")
				}
				cancelErrors[index] = err1
			}
		}
		if count != len(cancelOrderIDs) {
			panic("")
		}
	}

	if len(reqList) > 0 {
		if len(resp.Data.MassPlace) != 1 {
			panic("")
		}
		o := resp.Data.MassPlace[0]
		if o.Code.IsZero() {
			if len(o.OrderIDs) != len(reqList) {
				panic("")
			}
			for i := range orderIDs {
				orderIDs[i] = o.OrderIDs[i].String()
			}
		} else {
			err1 := fmt.Errorf("error code: %s", o.Code.String())
			for i := range placeErrors {
				placeErrors[i] = err1
			}
		}
	}

	return
}

// QueryPendingOrders Query pending orders
func (biki *Biki) QueryPendingOrders(symbol string, page, pageSize int) ([]goex.OrderDecimal, error) {
	param := map[string]string{
		"symbol": biki.transSymbol(symbol),
	}
	if page > 0 {
		param["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		param["pageSize"] = strconv.Itoa(pageSize)
	}
	param = biki.sign(param)

	url := fmt.Sprintf(apiBaseURL + newOrder + "?" + biki.buildQueryString(param))

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			Count      int
			ResultList []OrderInfo
		}
	}

	err := goex.HttpGet4(biki.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]goex.OrderDecimal, len(resp.Data.ResultList))
	for i := range resp.Data.ResultList {
		ret[i] = *resp.Data.ResultList[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

// QueryAllOrders Query all orders
func (biki *Biki) QueryAllOrders(symbol string, page, pageSize int) ([]goex.OrderDecimal, error) {
	param := map[string]string{
		"symbol": biki.transSymbol(symbol),
	}
	if page > 0 {
		param["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		param["pageSize"] = strconv.Itoa(pageSize)
	}
	param = biki.sign(param)

	url := fmt.Sprintf(apiBaseURL + allOrder + "?" + biki.buildQueryString(param))

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			Count     int
			OrderList []OrderInfo
		}
	}

	err := goex.HttpGet4(biki.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]goex.OrderDecimal, len(resp.Data.OrderList))
	for i := range resp.Data.OrderList {
		ret[i] = *resp.Data.OrderList[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

// QueryOrder Query an order
func (biki *Biki) QueryOrder(symbol string, orderID string) (*goex.OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	param := biki.sign(map[string]string{
		"symbol":   biki.transSymbol(symbol),
		"order_id": orderID,
	})

	url := fmt.Sprintf(apiBaseURL + orderInfo + "?" + biki.buildQueryString(param))

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			OrderInfo *OrderInfo `json:"order_info"`
		}
	}

	err := goex.HttpGet4(biki.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	if resp.Data.OrderInfo == nil {
		return nil, nil
	}

	return resp.Data.OrderInfo.ToOrderDecimal(symbol), nil
}
