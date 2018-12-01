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
)

const (
	FUTURE_V3_API_BASE_URL    = "https://www.okex.com"
	FUTURE_V3_INSTRUMENTS 	  = "/api/futures/v3/instruments"
	FUTURE_V3_POSITION 		  = "/api/futures/v3/position"
	FUTURE_V3_ACCOUNTS 		  = "/api/futures/v3/accounts"
	FUTURE_V3_INSTRUMENT_POSITION = "/api/futures/v3/%s/position"
	FUTURE_V3_INSTRUMENT_TICKER = "/api/futures/v3/instruments/%s/ticker"
	FUTURE_V3_INSTRUMENT_INDEX = "/api/futures/v3/instruments/%s/index"
	FUTURE_V3_ORDER			   = "/api/futures/v3/order"
	FUTURE_V3_CANCEL_ORDER		= "/api/futures/v3/cancel_order/%s/%s"
	FUTURE_V3_ORDERS 			= "/api/futures/v3/orders/%s"
	FUTURE_V3_ORDER_INFO 		= "/api/futures/v3/orders/%s/%s"
)

const (
	V3_DATE_FORMAT = "2006-01-02T15:04:05.000Z"
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
}

type V3Position struct {
	CreateAt string 		`json:"create_at"`
	InstrumentId string 	`json:"instrument_id"`
	Leverage string 		`json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
	LongAvailQty string 	`json:"long_avail_qty"`
	LongAvgCost string 		`json:"long_avg_cost"`
	LongQty string 			`json:"LongQty"`
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
	p.CreateDate = V3ParseDate(this.CreateAt)
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
	return instruments, err
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
	RealizedPnl string 		`json:"realized_pnl"`
	TotalAvailBalance string `json:"total_avail_balance"`
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
	Info struct {
		Btc V3CurrencyInfo `json:btc`
		Ltc V3CurrencyInfo `json:ltc`
		Etc V3CurrencyInfo `json:"etc"`
		Eth V3CurrencyInfo `json:"eth"`
		Bch V3CurrencyInfo `json:"bch"`
		Xrp V3CurrencyInfo `json:"xrp"`
		Eos V3CurrencyInfo `json:"eos"`
		Btg V3CurrencyInfo `json:"btg"`
	} `json:info`
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

	account.FutureSubAccounts[BTC] = *resp.Info.Btc.ToFutureSubAccount(BTC)
	account.FutureSubAccounts[LTC] = *resp.Info.Ltc.ToFutureSubAccount(LTC)
	account.FutureSubAccounts[BCH] = *resp.Info.Bch.ToFutureSubAccount(BCH)
	account.FutureSubAccounts[ETH] = *resp.Info.Eth.ToFutureSubAccount(ETH)
	account.FutureSubAccounts[ETC] = *resp.Info.Etc.ToFutureSubAccount(ETC)
	account.FutureSubAccounts[XRP] = *resp.Info.Xrp.ToFutureSubAccount(XRP)
	account.FutureSubAccounts[EOS] = *resp.Info.Eos.ToFutureSubAccount(EOS)
	account.FutureSubAccounts[BTG] = *resp.Info.Btg.ToFutureSubAccount(BTG)

	return account, nil
}

func (ok *OKExV3) PlaceFutureOrder(clientOid string, instrumentId string, price, size string, orderType, matchPrice, leverage int) (string, error) {
	params := map[string]string {
		"client_oid": clientOid,
		"instrument_id": instrumentId,
		"type": strconv.Itoa(orderType),
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
		ErrorCode int 	`json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Result bool `json:"result"`
	}

	err = json.Unmarshal(body, &ret)
	if err != nil {
		return "", err
	}

	if ret.ErrorCode != 0 {
		return "", fmt.Errorf("error code: %d", ret.ErrorCode)
	}

	return ret.OrderId, nil
}

func (ok *OKExV3) FutureCancelOrder(instrumentId, orderId string) error {
	reqUrl := fmt.Sprintf(FUTURE_V3_CANCEL_ORDER, instrumentId, orderId)

	header := ok.buildHeader("POST", reqUrl, "")

	reqPath := FUTURE_V3_API_BASE_URL + reqUrl
	body, err := HttpPostJson(ok.client, reqPath, "", header)

	respMap := make(map[string]interface{})
	err = json.Unmarshal(body, &respMap)
	if err != nil {
		return err
	}
	print(string(body))
	if respMap["result"] != nil && !respMap["result"].(bool) {
		if respMap["error_code"] != nil {
			return fmt.Errorf("error code: %s", respMap["error_code"].(string))
		}
		return errors.New(string(body))
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
	return o
}

func (ok *OKExV3) GetInstrumentOrders(instrumentId string, status, from, to, limit string) ([]FutureOrder, error) {
	reqUrl := fmt.Sprintf(FUTURE_V3_ORDERS, instrumentId)
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
