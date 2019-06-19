package biki

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
	"github.com/qiniu/api.v6/url"
)

const (
	ORDER_SELL = "SELL"
	ORDER_BUY = "BUY"

	ORDER_TYPE_LIMIT = 1
	ORDER_TYPE_MARKET = 2
)

const (
	API_BASE_URL    = "https://openapi.biki.com"
	COMMON_SYMBOLS = "/open/api/common/symbols"
	GET_TICKER = "/open/api/get_ticker?symbol=%s"
	GET_MARKET_DEPH = "/open/api/market_dept?symbol=%s&type=step0"
	GET_TRADES = "/open/api/get_trades?symbol=%s"
	ACCOUNT = "/open/api/user/account"
	CREATE_ORDER = "/open/api/create_order"
	CANCEL_ORDER = "/open/api/cancel_order"
	NEW_ORDER = "/open/api/new_order"
	ORDER_INFO = "/open/api/order_info"
	ALL_ORDER = "/open/api/all_order"
)

type Biki struct {
	ApiKey    string
	SecretKey string
	client    *http.Client

	symbolNameMap map[string]string
}

func NewBiki(ApiKey string, SecretKey string) *Biki {
	this := new(Biki)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient

	this.symbolNameMap = make(map[string]string)
	return this
}


func (this *Biki) getPairByName(name string) string {
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
		this.symbolNameMap[strings.ToUpper(o.Symbol)] = fmt.Sprintf("%s_%s", o.BaseCoin, o.CountCoin)
	}
	c, ok = this.symbolNameMap[name]
	if !ok {
		return ""
	}
	return c
}

func (ok *Biki) GetSymbols() ([]Symbol, error) {
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
		Msg string
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

func (this *Biki) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Biki) GetTicker(symbol string) (*TickerDecimal, error) {
	symbol = this.transSymbol(symbol)
	url := API_BASE_URL + GET_TICKER
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
			High decimal.Decimal
			Vol decimal.Decimal
			Last decimal.Decimal
			Low decimal.Decimal
			Buy decimal.Decimal
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

func (this *Biki) GetDepth(symbol string) (*DepthDecimal, error) {
	inputSymbol := symbol
	symbol = this.transSymbol(symbol)
	url := fmt.Sprintf(API_BASE_URL + GET_MARKET_DEPH, symbol)
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

func (this *Biki) GetTrades(symbol string) ([]TradeDecimal, error) {
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
		Msg string
		Code decimal.Decimal
		Data []struct {
			Amount decimal.Decimal
			Price decimal.Decimal
			Id int64
			Type string
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
		t.Tid = o.Id
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Type
	}

	return trades, nil
}

func (this *Biki) signData(data string) string {
	message := data + this.SecretKey
	sign, _ := GetParamMD5Sign(this.SecretKey, message)

	return sign
}

func (this *Biki) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["api_key"] = this.ApiKey
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
	param["sign"] = sign
	return param
}

func (this *Biki) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}

func (this *Biki) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string {}
	params = this.sign(params)

	url := API_BASE_URL + ACCOUNT + "?" + this.buildQueryString(params)

	var resp struct {
		Msg string
		Code decimal.Decimal
		Data struct {
				TotalAsset decimal.Decimal 	`json:"total_asset"`
				CoinList []struct {
					Coin string
					Normal decimal.Decimal
					Locked decimal.Decimal
					BtcValuatin decimal.Decimal
				}	`json:"coin_list"`
			}
	}

	err := HttpGet4(this.client, url, map[string]string{}, &resp)

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

func (this *Biki) PlaceOrder(volume decimal.Decimal, side string, _type int, symbol string, price decimal.Decimal) (string, error) {
	symbol = this.transSymbol(symbol)
	params := map[string]string {
		"side": side,
		"volume": volume.String(),
		"type": strconv.Itoa(_type),
		"symbol": symbol,
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	println(data)
	url := API_BASE_URL + CREATE_ORDER
	body, err := HttpPostForm3(this.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return "", err
	}

	println(string(body))

	var resp struct {
		Msg string
		Code decimal.Decimal
		Data struct {
		   OrderId string		`json:"order_id"`
	   }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.Code.IntPart() != 0 {
		return "", fmt.Errorf("error code: %s", resp.Code.String())
	}

	return resp.Data.OrderId, nil
}

func (this *Biki) CancelOrder(symbol string, orderId string) error {
	symbol = this.transSymbol(symbol)
	params := map[string]string {
		"symbol": symbol,
		"order_id": orderId,

	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	println(data)
	url := API_BASE_URL + CANCEL_ORDER
	println(url)
	body, err := HttpPostForm3(this.client, url, data, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if err != nil {
		return err
	}
	println(string(body))

	var resp struct {
		Msg string
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

func (this *Biki) QueryPendingOrders(symbol string, page, pageSize int) ([]OrderDecimal, error) {
	param := this.sign(map[string]string {
		"symbol": this.transSymbol(symbol),
	})

	if page > 0 {
		param["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		param["pageSize"] = strconv.Itoa(pageSize)
	}

	url := fmt.Sprintf(API_BASE_URL + NEW_ORDER + "?" + this.buildQueryString(param))

	var resp struct {
	    Msg string
	    Code decimal.Decimal
		Data struct {
			Count int
			ResultList []OrderInfo
		}
	}

	err := HttpGet4(this.client, url, nil, &resp)
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

func (this *Biki) QueryOrder(symbol string, orderId string) (*OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	param := this.sign(map[string]string {
		"symbol": this.transSymbol(symbol),
		"order_id": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + ORDER_INFO + "?" + this.buildQueryString(param))

	var resp struct {
	    Msg string
	    Code decimal.Decimal
		Data struct {
			OrderInfo *OrderInfo			`json:"order_info"`
			 }
	}

	err := HttpGet4(this.client, url, nil, &resp)
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
