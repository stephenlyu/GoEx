package bicc

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	. "github.com/stephenlyu/GoEx"
	"strconv"
	"sort"
	"net/url"
	"sync"
	"errors"
)

const (
	OrderSell = "SELL"
	OrderBuy = "BUY"
)

const (
	OrderTypeLimit = "1"
	OrderTypeMarket = "2"
)

const (
	OrderStatusInit = 0
	OrderStatusNew = 1
	OrderStatusPartiallyFilled = 3
	OrderStatusFilled = 2
	OrderStatusCanceled = 4
	OrderStatusPendingCancel = 5
	OrderStatusRejected = 6
)

const (
	API_BASE_URL = "https://openapi.bi.cc"
	COMMON_SYMBOLS = "/open/api/common/symbols"
	GET_TICKER = "/open/api/get_ticker?symbol=%s"
	GET_MARKET_DEPH = "/open/api/market_dept?symbol=%s&type=step0"
	GET_TRADES = "/open/api/get_trades?symbol=%s"
	ACCOUNT = "/open/api/user/account"
	CREATE_ORDER = "/open/api/create_order"
	BATCH_REPLACE = "/open/api/mass_replace"
	CANCEL_ORDER = "/open/api/cancel_order"
	NEW_ORDER = "/open/api/v2/new_order"
	ORDER_INFO = "/open/api/order_info"
)

var ErrNotExist = errors.New("NOT EXISTS")

type Bicc struct {
	ApiKey           string
	SecretKey        string
	client           *http.Client

	symbolNameMap    map[string]string

	ws               *WsConn
	createWsLock     sync.Mutex
	wsDepthHandleMap map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(string, []TradeDecimal)
	errorHandle      func(error)
	wsSymbolMap      map[string]string
}

func NewBicc(client *http.Client, ApiKey string, SecretKey string) *Bicc {
	this := new(Bicc)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = client

	this.symbolNameMap = make(map[string]string)
	return this
}

func (this *Bicc) getPairByName(name string) string {
	name = strings.ToUpper(name)
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
		oSymbol := fmt.Sprintf("%s%s", o.BaseCoin, o.CountCoin)
		this.symbolNameMap[strings.ToUpper(oSymbol)] = fmt.Sprintf("%s_%s", o.BaseCoin, o.CountCoin)
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (ok *Bicc) GetSymbols() ([]Symbol, error) {
	url := API_BASE_URL + COMMON_SYMBOLS
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

func (this *Bicc) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Bicc) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_TICKER, symbol)
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

	ticker := new(TickerDecimal)
	ticker.Date = uint64(r.Time)
	ticker.Buy = r.Buy
	ticker.Sell = r.Sell
	ticker.Last = r.Last
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol

	return ticker, nil
}

func (this *Bicc) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_MARKET_DEPH, symbol)
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

func (this *Bicc) GetTrades(symbol string) ([]TradeDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_TRADES, symbol)
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

	var trades = make([]TradeDecimal, len(data.Data))

	for i, o := range data.Data {
		t := &trades[i]
		t.Tid = o.ID
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = strings.ToLower(o.Type)
	}

	return trades, nil
}

func (this *Bicc) signData(data string) string {
	message := data + this.SecretKey
	sign, _ := GetParamMD5Sign(this.SecretKey, message)

	return sign
}

func (this *Bicc) buildQueryString(param map[string]string) string {
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

func (this *Bicc) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["time"] = timestamp
	param["api_key"] = this.ApiKey

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
		parts = append(parts, key + value)
	}
	data := strings.Join(parts, "")

	sign := this.signData(data)
	param["sign"] = sign
	return param
}

func (this *Bicc) getAuthHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (this *Bicc) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string{}
	params = this.sign(params)

	url := API_BASE_URL + ACCOUNT + "?" + this.buildQueryString(params)

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

	header := this.getAuthHeader()
	err := HttpGet4(this.client, url, header, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret []SubAccountDecimal
	for _, o := range resp.Data.CoinList {
		currency := strings.ToUpper(o.Coin)
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Normal,
			FrozenAmount: o.Locked,
			Amount: o.Normal.Add(o.Locked),
		})
	}

	return ret, nil
}

func (this *Bicc) PlaceOrder(volume decimal.Decimal, side string, _type string, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"symbol": symbol,
		"side": side,
		"type": _type,
		"volume": volume.String(),
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	println(data)
	url := API_BASE_URL + CREATE_ORDER
	body, err := HttpPostForm3(this.client, url, data, this.getAuthHeader())

	if err != nil {
		return "", err
	}
	println(string(body))

	var resp struct {
		Msg  string
		Code decimal.Decimal
		Data struct {
				 OrderId decimal.Decimal `json:"order_id"`
			 }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 0 {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Data.OrderId.String(), nil
}

func (this *Bicc) CancelOrder(symbol, orderId string) error {
	symbol = this.transSymbol(symbol)
	params := map[string]string{
		"symbol": symbol,
		"order_id": orderId,
	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CANCEL_ORDER
	body, err := HttpPostForm3(this.client, url, data, this.getAuthHeader())
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

func (this *Bicc) QueryPendingOrders(symbol string, page, pageSize int) ([]OrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 20
	}
	param := map[string]string{
		"symbol": this.transSymbol(symbol),
	}
	param["page"] = strconv.Itoa(page)
	param["pageSize"] = strconv.Itoa(pageSize)
	param = this.sign(param)

	url := fmt.Sprintf(API_BASE_URL + NEW_ORDER + "?" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data struct {
				 Count      int
				 ResultList []OrderInfo
			 }
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 0 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret = make([]OrderDecimal, len(resp.Data.ResultList))
	for i := range resp.Data.ResultList {
		ret[i] = *resp.Data.ResultList[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *Bicc) QueryOrder(symbol, orderId string) (*OrderDecimal, error) {
	param := this.sign(map[string]string{
		"symbol": this.transSymbol(symbol),
		"order_id": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + ORDER_INFO + "?" + this.buildQueryString(param))

	var resp struct {
		Code decimal.Decimal
		Msg  string
		Data struct {
				 OrderInfo *OrderInfo `json:"order_info"`
			 }
	}

	err := HttpGet4(this.client, url, this.getAuthHeader(), &resp)
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

func (this *Bicc) BatchReplace(symbol string, cancelOrderIds []string, placeReqList []OrderReq) (cancelErrors []error, orderIds []string, placeErrList []error, err error) {
	cancelErrors = make([]error, len(cancelOrderIds))
	orderIds = make([]string, len(placeReqList))
	placeErrList = make([]error, len(placeReqList))

	symbol = this.transSymbol(symbol)
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

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + BATCH_REPLACE
	var body []byte
	body, err = HttpPostForm3(this.client, url, data, this.getAuthHeader())

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
					 OrderId decimal.Decimal `json:"order_id"`
				 }`json:"mass_cancel"`
				 MassPlace  []struct {
					 Code    decimal.Decimal
					 Msg     string
					 OrderId decimal.Decimal `json:"order_id"`
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
			orderIds[i] = o.OrderId.String()
		}
	}

	return
}