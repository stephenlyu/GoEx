package bibull

import (
	"encoding/json"
	"errors"
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

// Order side
const (
	OrderSell = "SELL"
	OrderBuy  = "BUY"
)

// Order type
const (
	OrderTypeLimit  = "1"
	OrderTypeMarket = "2"
)

// Order status
const (
	OrderStatusInit            = 0
	OrderStatusNew             = 1
	OrderStatusPartiallyFilled = 3
	OrderStatusFilled          = 2
	OrderStatusCanceled        = 4
	OrderStatusPendingCancel   = 5
	OrderStatusRejected        = 6
)

const (
	_ApiBaseURL        = "https://openapi.bibull.co"
	_CommonSymbolURL   = "/open/api/common/symbols"
	_GetTickerURL      = "/open/api/get_ticker?symbol=%s"
	_GetMarketDepthURL = "/open/api/market_dept?symbol=%s&type=step0"
	_GetTradesURL      = "/open/api/get_trades?symbol=%s"
	_AccountURL        = "/open/api/user/account"
	_CreateOrderURL    = "/open/api/create_order"
	_BatchReplaceURL   = "/open/api/mass_replace"
	_CancelOrderURL    = "/open/api/cancel_order"
	_NewOrderURL       = "/open/api/v2/new_order"
	_OrderInfoURL      = "/open/api/order_info"
)

// ErrNotExist order not exist error
var ErrNotExist = errors.New("NOT EXISTS")

// BiBull Bibull API
type BiBull struct {
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

// NewBiBull BiBull constructore
func NewBiBull(client *http.Client, APIKey string, SecretKey string) *BiBull {
	api := new(BiBull)
	api.APIKey = APIKey
	api.SecretKey = SecretKey
	api.client = client

	api.symbolNameMap = make(map[string]string)
	return api
}

func (api *BiBull) getPairByName(name string) string {
	name = strings.ToUpper(name)
	c, ok := api.symbolNameMap[name]
	if ok {
		return c
	}

	var err error
	var l []Symbol
	for i := 0; i < 5; i++ {
		l, err = api.GetSymbols()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}

	for _, o := range l {
		oSymbol := fmt.Sprintf("%s%s", o.BaseCoin, o.CountCoin)
		api.symbolNameMap[strings.ToUpper(oSymbol)] = fmt.Sprintf("%s_%s", o.BaseCoin, o.CountCoin)
	}
	c, ok = api.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

// GetSymbols Query all supported symbols
func (api *BiBull) GetSymbols() ([]Symbol, error) {
	url := _ApiBaseURL + _CommonSymbolURL
	resp, err := api.client.Get(url)
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

func (api *BiBull) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

// GetTicker Get ticker of a symbol
func (api *BiBull) GetTicker(symbol string) (*goex.TickerDecimal, error) {
	symbol = api.transSymbol(symbol)
	url := fmt.Sprintf(_ApiBaseURL+_GetTickerURL, symbol)
	resp, err := api.client.Get(url)
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

// GetDepth Get market depth of a symbol
func (api *BiBull) GetDepth(symbol string) (*goex.DepthDecimal, error) {
	inputSymbol := symbol
	symbol = api.transSymbol(symbol)
	url := fmt.Sprintf(_ApiBaseURL+_GetMarketDepthURL, symbol)
	println(url)
	resp, err := api.client.Get(url)
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

// GetTrades Get trades of a symbol
func (api *BiBull) GetTrades(symbol string) ([]goex.TradeDecimal, error) {
	symbol = api.transSymbol(symbol)
	url := fmt.Sprintf(_ApiBaseURL+_GetTradesURL, symbol)
	resp, err := api.client.Get(url)
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
		t.Type = strings.ToLower(o.Type)
	}

	return trades, nil
}

func (api *BiBull) signData(data string) string {
	message := data + api.SecretKey
	sign, _ := goex.GetParamMD5Sign(api.SecretKey, message)

	return sign
}

func (api *BiBull) buildQueryString(param map[string]string) string {
	var parts []string

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var sign string
	for _, k := range keys {
		v := param[k]
		if k == "sign" {
			sign = v
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	parts = append(parts, fmt.Sprintf("sign=%s", url.QueryEscape(sign)))
	return strings.Join(parts, "&")
}

func (api *BiBull) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	param["time"] = timestamp
	param["api_key"] = api.APIKey

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, key := range keys {
		value := param[key]
		if value == "" {
			continue
		}
		parts = append(parts, key+value)
	}
	data := strings.Join(parts, "")

	sign := api.signData(data)
	param["sign"] = sign
	return param
}

func (api *BiBull) getAuthHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

// GetAccount Query account info
func (api *BiBull) GetAccount() ([]goex.SubAccountDecimal, error) {
	params := map[string]string{}
	params = api.sign(params)

	url := _ApiBaseURL + _AccountURL + "?" + api.buildQueryString(params)

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			TotalAsset decimal.Decimal `json:"total_asset"`
			CoinList   []struct {
				Coin   string
				Normal decimal.Decimal
				Locked decimal.Decimal
			} `json:"coin_list"`
		}
	}

	header := api.getAuthHeader()
	err := goex.HttpGet4(api.client, url, header, &resp)

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

// PlaceOrder Place an order
func (api *BiBull) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	symbol = api.transSymbol(symbol)
	params := map[string]string{
		"symbol": symbol,
		"side":   side,
		"type":   _type,
		"volume": volume.String(),
		"price":  price.String(),
	}

	params = api.sign(params)

	data := api.buildQueryString(params)

	url := _ApiBaseURL + _CreateOrderURL
	body, err := goex.HttpPostForm3(api.client, url, data, api.getAuthHeader())

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

// CancelOrder Cancel an order
func (api *BiBull) CancelOrder(symbol, orderID string) error {
	symbol = api.transSymbol(symbol)
	params := map[string]string{
		"symbol":   symbol,
		"order_id": orderID,
	}
	params = api.sign(params)

	data := api.buildQueryString(params)
	url := _ApiBaseURL + _CancelOrderURL
	body, err := goex.HttpPostForm3(api.client, url, data, api.getAuthHeader())
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

	if resp.Code.IntPart() == 22 {
		return ErrNotExist
	}

	if resp.Code.IntPart() != 0 && resp.Code.IntPart() != -1 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

// QueryPendingOrders Query pending orders
func (api *BiBull) QueryPendingOrders(symbol string, page, pageSize int) ([]goex.OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 20
	}
	param := map[string]string{
		"symbol": api.transSymbol(symbol),
	}
	param["page"] = strconv.Itoa(page)
	param["pageSize"] = strconv.Itoa(pageSize)
	param = api.sign(param)

	url := fmt.Sprintf(_ApiBaseURL + _NewOrderURL + "?" + api.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data struct {
			Count      int
			ResultList []OrderInfo
		}
	}

	err := goex.HttpGet4(api.client, url, api.getAuthHeader(), &resp)
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

// QueryOrder Query an order
func (api *BiBull) QueryOrder(symbol, orderID string) (*goex.OrderDecimal, error) {
	param := api.sign(map[string]string{
		"symbol":   api.transSymbol(symbol),
		"order_id": orderID,
	})

	url := fmt.Sprintf(_ApiBaseURL + _OrderInfoURL + "?" + api.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data struct {
			OrderInfo *OrderInfo `json:"order_info"`
		}
	}

	err := goex.HttpGet4(api.client, url, api.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	if resp.Data.OrderInfo == nil {
		return nil, ErrNotExist
	}

	return resp.Data.OrderInfo.ToOrderDecimal(symbol), nil
}

// BatchReplace Batch replace orders
func (api *BiBull) BatchReplace(symbol string, cancelOrderIds []string, placeReqList []OrderReq) (cancelErrors []error, orderIds []string, placeErrList []error, err error) {
	cancelErrors = make([]error, len(cancelOrderIds))
	orderIds = make([]string, len(placeReqList))
	placeErrList = make([]error, len(placeReqList))

	symbol = api.transSymbol(symbol)
	params := map[string]string{
		"symbol": symbol,
	}

	if len(cancelOrderIds) > 0 {
		bytes, _ := json.Marshal(cancelOrderIds)
		params["mass_cancel"] = string(bytes)
	}

	if len(placeReqList) > 0 {
		bytes, _ := json.Marshal(placeReqList)
		params["mass_place"] = string(bytes)
	}

	params = api.sign(params)

	data := api.buildQueryString(params)
	url := _ApiBaseURL + _BatchReplaceURL
	var body []byte
	body, err = goex.HttpPostForm3(api.client, url, data, api.getAuthHeader())

	if err != nil {
		return
	}

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
			MassCancel []struct {
				Code    decimal.Decimal
				Msg     string
				OrderID decimal.Decimal `json:"order_id"`
			} `json:"mass_cancel"`
			MassPlace []struct {
				Code    decimal.Decimal
				Msg     string
				OrderID decimal.Decimal `json:"order_id"`
			} `json:"mass_place"`
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}

	if resp.Code.IntPart() != 0 {
		err = fmt.Errorf("error code: %s", resp.Code.String())
		return
	}

	if len(resp.Data.MassCancel) != len(cancelOrderIds) {
		panic("cancel order id count not matched")
	}

	if len(resp.Data.MassPlace) != len(placeReqList) {
		panic("place order count not matched")
	}

	for i, o := range resp.Data.MassCancel {
		if o.Code.IntPart() != 0 {
			cancelErrors[i] = fmt.Errorf("error code: %s", o.Code.String())
		}
	}

	for i, o := range resp.Data.MassPlace {
		if o.Code.IntPart() != 0 {
			placeErrList[i] = fmt.Errorf("error code: %s", o.Code.String())
		} else {
			orderIds[i] = o.OrderID.String()
		}
	}

	return
}
