package ceohk

import (
	. "github.com/stephenlyu/GoEx"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"strconv"
	"sort"
	"time"
	"net/url"
)

const (
	API_BASE_URL    = "https://ceohk.bi"
	TICKER 	  		= "/api/market/ticker?market=%s"
	ALL_TICKS = "/api/market/allTicker"
	USER = "/api/deal/accountInfo"
	DEAL_ORDER = "/api/deal/order"
	CANCEL_ORDER = "/api/deal/cancelOrder"
	GET_ORDER = "/api/deal/getOrder"
	GET_ORDERS = "/api/deal/getOrders"
)

const (
	TRADE_TYPE_BUY = 1
	TRADE_TYPE_SELL = 2
)

const (
	TRADE_STATUS_TRADING = "0"
	TRADE_STATUS_FILLED = "1"
	TRADE_STATUS_CANCELED = "2"
	TRADE_STATUS_PARTIAL_FILLED = "3"
)

type CEOHK struct {
	ApiKey    string
	SecretKey string
	client            *http.Client
}

func NewCEOHK(ApiKey string, SecretKey string) *CEOHK {
	this := new(CEOHK)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = http.DefaultClient
	return this
}

func (ok *CEOHK) GetAllTickers() (map[string]*TickerDecimal, error) {
	url := API_BASE_URL + ALL_TICKS
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
		Code decimal.Decimal
		Data map[string]*struct {
				 Buy decimal.Decimal
				 Sell decimal.Decimal
				 Last decimal.Decimal
				 Vol decimal.Decimal
				 High decimal.Decimal
				 Low decimal.Decimal
				 Time decimal.Decimal
			 }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		err = fmt.Errorf("body: %s", string(body))
		return nil, err
	}

	if data.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	ret := make(map[string]*TickerDecimal)
	for symbol, r := range data.Data {
		ticker := new(TickerDecimal)
		ticker.Date = uint64(r.Time.IntPart()) * 1000
		ticker.Buy = r.Buy
		ticker.Sell = r.Sell
		ticker.Last = r.Last
		ticker.High = r.High
		ticker.Low = r.Low
		ticker.Vol = r.Vol
		ret[symbol] = ticker
	}

	return ret, nil
}

func (ok *CEOHK) GetTicker(market string) (*TickerDecimal, error) {
	market = strings.ToLower(market)
	url := API_BASE_URL + TICKER
	resp, err := ok.client.Get(fmt.Sprintf(url, market))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var data struct {
		Code decimal.Decimal
		Data struct {
			Buy decimal.Decimal
			Sell decimal.Decimal
			Last decimal.Decimal
			Vol decimal.Decimal
			High decimal.Decimal
			Low decimal.Decimal
			Time decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		err = fmt.Errorf("body: %s", string(body))
		return nil, err
	}

	if data.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	ticker := new(TickerDecimal)
	ticker.Date = uint64(data.Data.Time.IntPart()) * 1000
	ticker.Buy = data.Data.Buy
	ticker.Sell = data.Data.Sell
	ticker.Last = data.Data.Last
	ticker.High = data.Data.High
	ticker.Low = data.Data.Low
	ticker.Vol = data.Data.Vol

	return ticker, nil
}

func (this *CEOHK) signData(data string) string {
	message := data
	sign, _ := GetParamHmacMD5Sign(this.SecretKey, message)

	return sign
}

func (this *CEOHK) sign(param map[string]string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	param["accesskey"] = this.ApiKey
	param["reqTime"] = timestamp

	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + "=" + param[k])
	}
	data := strings.Join(parts, "&")
	sign := this.signData(data)
	param["sign"] = sign
	return param
}

func (this *CEOHK) buildQueryString(param map[string]string) string {
	var parts []string
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	return strings.Join(parts, "&")
}


func (this *CEOHK) GetAccount() ([]SubAccountDecimal, error) {
	params := map[string]string {
		"method": "accountInfo",
	}
	params = this.sign(params)

	url := API_BASE_URL + USER + "?" + this.buildQueryString(params)

	var resp struct {
		Code decimal.Decimal
		Message string
		Data struct {
				Coins []struct {
					EnName string
					Available decimal.Decimal
					Freez decimal.Decimal
					UnitDecimal decimal.Decimal
				}
			}
	}

	err := HttpGet4(this.client, url, map[string]string{}, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", resp.Code.String())
	}

	var ret []SubAccountDecimal
	for _, o := range resp.Data.Coins {
		currency := strings.ToUpper(o.EnName)
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: currency},
			AvailableAmount: o.Available,
			FrozenAmount: o.Freez,
			Amount: o.Available.Add(o.Freez),
		})
	}

	return ret, nil
}

func (this *CEOHK) PlaceOrder(amount decimal.Decimal, _type int, symbol string, price decimal.Decimal) (string, error) {
	symbol = strings.ToLower(symbol)
	params := map[string]string {
		"method": "order",
		"tradeType": strconv.Itoa(_type),
		"amount": amount.String(),
		"currency": symbol,
		"price": price.String(),
	}

	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + DEAL_ORDER + "?" + data

	bytes, err := HttpGet6(this.client, url, nil)

	if err != nil {
		return "", err
	}
	var baseResp struct {
		Message string
		Code decimal.Decimal
	}
	err = json.Unmarshal(bytes, &baseResp)
	if err != nil {
		return "", err
	}

	if baseResp.Code.IntPart() != 1000 {
		return "", fmt.Errorf("error code: %s", baseResp.Code.String())
	}

	var resp struct {
		Message string
		Code decimal.Decimal
		Data *struct {
			OrderId string
		}
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return "", err
	}

	return resp.Data.OrderId, nil
}

func (this *CEOHK) CancelOrder(symbol string, orderId string) error {
	symbol = strings.ToLower(symbol)
	params := map[string]string {
		"method": "cancelOrder",
		"currency": symbol,
		"id": orderId,

	}
	params = this.sign(params)

	data := this.buildQueryString(params)
	url := API_BASE_URL + CANCEL_ORDER + "?" + data

	var resp struct {
		Msg string
		Code decimal.Decimal
	}
	err := HttpGet4(this.client, url, nil, &resp)

	if err != nil {
		return err
	}

	if resp.Code.IntPart() != 1000 {
		return fmt.Errorf("error code: %s", resp.Code.String())
	}

	return nil
}

func (this *CEOHK) QueryOrders(symbol string, page, pageSize int, tradeType int, tradeStatus string) ([]OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}
	param := map[string]string {
		"method": "getOrders",
		"currency": strings.ToLower(symbol),
		"pageIndex": strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	}
	if tradeType > 0 {
		param["tradeType"] = strconv.Itoa(tradeType)
	}
	if tradeStatus != "" {
		param["tradeStatus"] = tradeStatus
	}

	param = this.sign(param)

	url := fmt.Sprintf(API_BASE_URL + GET_ORDERS + "?" + this.buildQueryString(param))
	bytes, err := HttpGet6(this.client, url, nil)
	if err != nil {
		return nil, err
	}
	var baseResp struct {
		Message string
		Code decimal.Decimal
	}
	err = json.Unmarshal(bytes, &baseResp)
	if err != nil {
		return nil, err
	}

	if baseResp.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", baseResp.Code.String())
	}

	var resp struct {
		Message string
		Code decimal.Decimal
		Data []OrderInfo
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	var ret = make([]OrderDecimal, len(resp.Data))
	for i := range resp.Data {
		ret[i] = *resp.Data[i].ToOrderDecimal(symbol)
	}

	return ret, nil
}

func (this *CEOHK) QueryOrder(symbol string, orderId string) (*OrderDecimal, error) {
	symbol = strings.ToUpper(symbol)
	param := this.sign(map[string]string {
		"method": "getOrder",
		"currency": strings.ToLower(symbol),
		"id": orderId,
	})

	url := fmt.Sprintf(API_BASE_URL + GET_ORDER + "?" + this.buildQueryString(param))
	bytes, err := HttpGet6(this.client, url, nil)
	if err != nil {
		return nil, err
	}
	var baseResp struct {
		Message string
		Code decimal.Decimal
	}
	err = json.Unmarshal(bytes, &baseResp)
	if err != nil {
		return nil, err
	}

	if baseResp.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", baseResp.Code.String())
	}

	var resp struct {
		Message string
		Code decimal.Decimal
		Data *OrderInfo
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data.ToOrderDecimal(symbol), nil
}
