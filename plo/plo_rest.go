package plo

import (
	"net/http"
	"sort"
	"strings"
	"fmt"
	"github.com/stephenlyu/GoEx"
	"github.com/stephenlyu/tds/util"
	"encoding/json"
	"strconv"
	"encoding/base64"
	"github.com/shopspring/decimal"
)

const (
	BASE_URL = "https://api.plo.one/"
	TRADE_URL = "/m_api/trade"
	ORDER_BOOK_URL = "/m_api/orderbookL2"
	CONFIG_LIST_URL = "/hapi/Config/ConfList"
	BALANCES_URL = "/hapi/BatchOperation/balances"
	PLACE_ORDER_URL = "/hapi/BatchOperation/batchPosExec"
	SELF_TRADE_URL = "/hapi/BatchOperation/selfTrade"
	SIMPLE_SELF_TRADE_URL = "/hapi/BatchOperation/simpleSelfTrade"
	CANCEL_ORDER_URL = "/hapi/BatchOperation/batchOrderCancel"
	BATCH_ORDER_URL = "/hapi/BatchOperation/batchOrderCondquery"
	ORDERS_URL = "/hapi/BatchOperation/orders"
	POSITIONS_URL = "/hapi/BatchOperation/positions"
	POS_RANK_URL = "/hapi/BatchOperation/posRanking"
)

type PloRest struct {
	apiKey string
	apiSecretKey string
	client *http.Client
}

func NewPloRest(apiKey string, apiSecretKey string) *PloRest {
	return &PloRest{
		apiKey: apiKey,
		apiSecretKey: apiSecretKey,

		client: http.DefaultClient,
	}
}

func (bitmex *PloRest) map2Query(params map[string]string) string {
	keys := make([]string, len(params))
	var i int
	for k := range params {
		keys[i] = k
		i++
	}
	sort.SliceStable(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	parts := make([]string, len(params))
	for i, k := range keys {
		v := params[k]
		parts[i] = k + "=" + v
		i++
	}
	return strings.Join(parts, "&")
}

func (this *PloRest) GetTrade(pair goex.CurrencyPair) (error, []goex.TradeDecimal) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data struct {
		Data []struct {
			Timestamp int64
			Side string
			Symbol string
			Size decimal.Decimal
			Price decimal.Decimal
		}
	}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+TRADE_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	if len(data.Data) == 0 {
		return nil, nil
	}

	ret := make([]goex.TradeDecimal, len(data.Data))
	for i, r := range data.Data {
		ret[i] = goex.TradeDecimal{
			Tid: r.Timestamp,
			Type: strings.ToLower(r.Side),
			Amount: r.Size,
			Price: r.Price,
			Date: r.Timestamp,
		}
	}

	return nil, ret
}

func (this *PloRest) GetOrderBook(pair goex.CurrencyPair) (error, *goex.DepthDecimal) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data struct {
		Data []struct {
			Price string
			Side string
			Size string
			Symbol string
		}
	}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+ORDER_BOOK_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	var asks, bids goex.DepthRecordsDecimal
	for _, r := range data.Data {
		price, _ := decimal.NewFromString(r.Price)
		amount, _ := decimal.NewFromString(r.Size)
		if r.Side == "buy" {
			bids = append(bids, goex.DepthRecordDecimal{Price: price, Amount: amount})
		} else {
			asks = append(asks, goex.DepthRecordDecimal{Price: price, Amount: amount})
		}
	}

	sort.SliceStable(asks, func(i,j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})

	sort.SliceStable(bids, func(i,j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	return nil, &goex.DepthDecimal{
		Pair: pair,
		AskList: asks,
		BidList: bids,
	}
}

type PloConfig struct {
	Currency struct {
		Decimals 	int			`json:"decimals"`
		Symbol 		string		`json:"symbol"`
	}							`json:"currency"`
	IndexDecimalDigits int		`json:"indexDecimalDigits"`
	IndexSymbol string			`json:"indexSymbol"`
	LastPrice float64			`json:"lastPrice"`
	MaintMargin string 			`json:"maintMargin"`
	MakerFee string 			`json:"makerFee"`
	MaxLeverage string 			`json:"maxLeverage"`
	MaxOrderQty int 			`json:"maxOrderQty"`
	MaxPositionQty int 			`json:"maxPositionQty"`
	PriceDecimalDigits int		`json:"priceDecimalDigits"`
	QuoteCurrency struct {
	 	Decimals 	int			`json:"decimals"`
	 	Symbol 		string		`json:"symbol"`
	}							`json:"quoteCurrency"`
	RiseOrFall int				`json:"riseOrFall"`
	Sort int					`json:"sort"`
	Symbol string 				`json:"symbol"`
	TakerFee string 			`json:"takerFee"`
	Type string 				`json:"type"`
	UnitValue string 			`json:"unitValue"`
}

func (this *PloRest) GetConfigList() (error, []PloConfig) {
	var data struct {
		Error string		`json:"err"`
		Msg string 			`json:"msg"`
		Data []PloConfig	`json:"data"`
	}
	err := goex.HttpGet4(this.client, BASE_URL+CONFIG_LIST_URL, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	if data.Error != "0" {
		return fmt.Errorf("error: %s", data.Error), nil
	}

	return nil, data.Data
}

type PloBalance struct {
	AccountId string 		`json:"accountId"`
	Address string 			`json:"address"`
	Balance string 			`json:"balance"`
	Currency string 		`json:"currency"`
	OrderMargin string 		`json:"orderMargin"`
	PositionMargin string 	`json:"positionMargin"`
}

func (r PloBalance) ToFutureSubAccount() goex.FutureSubAccount {
	currency := goex.Currency{Symbol: r.Currency}
	balance, _ := strconv.ParseFloat(r.Balance, 64)
	orderMargin, _ := strconv.ParseFloat(r.OrderMargin, 64)
	positionMargin, _ := strconv.ParseFloat(r.PositionMargin, 64)

	return goex.FutureSubAccount{
		Currency: currency,
		AccountRights: balance,
		KeepDeposit: orderMargin + positionMargin,
	}
}

func (this *PloRest) GetBalances() (error, *goex.FutureAccount) {
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, "")

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+BALANCES_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []PloBalance	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error > 0 {
		return fmt.Errorf("error: %d", resp.Error), nil
	}

	ret := new(goex.FutureAccount)
	ret.FutureSubAccounts = make(map[goex.Currency]goex.FutureSubAccount)

	for _, r := range resp.Data {
		sa := r.ToFutureSubAccount()
		ret.FutureSubAccounts[sa.Currency] = sa
	}
	return nil, ret
}

type OrderReq struct {
	PosAction int 			`json:"posAction"`
	AutoCancel int 			`json:"autoCancel"`
	ClientId string 		`json:"clientId"`
	Side string 			`json:"side"`
	Symbol string 			`json:"symbol"`
	TotalQty int64 			`json:"totalQty"`
	Price float64 			`json:"price"`
	Type string 			`json:"type"`
	Leverage int 			`json:"leverage"`
	PosId string 			`json:"posId"`
	PostOnly int 			`json:"postOnly"`
}

type OrderResp struct {
	Error int 				`json:"status"`
	Msg string 				`json:"msg"`
	Data *json.RawMessage	`json:"data"`
	Order *struct {
		AccountId string 		`json:"accountId"`
		Symbol string 			`json:"symbol"`
		Type string 			`json:"type"`
		Side string 			`json:"side"`
		ClientId string 		`json:"clientId"`
		Price string 			`json:"price"`
		PosAction int 			`json:"posAction"`
		CurrentQty int64 		`json:"currentQty"`
		TotalQty int64 			`json:"totalQty"`
		Status int 				`json:"status"`
		Timestamp int64 		`json:"timestamp"`
		PosMargin string 		`json:"posMargin"`
		OpenFee string 			`json:"openFee"`
		CloseFee string 		`json:"closeFee"`
		PosId string 			`json:"posId"`
		OrderId string 			`json:"orderId"`
	}
}

func (this *PloRest) PlaceOrders(reqOrders []OrderReq) (error, []OrderResp) {
	data, _ := json.MarshalIndent(reqOrders, "", "  ")
	println(string(data))

	ts := util.Tick()
	bytes, _ := json.Marshal(reqOrders)
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("placeorders", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+PLACE_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}
	println(string(bytes))
	var resp struct {
		Data []OrderResp	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	for i := range resp.Data {
		item := &resp.Data[i]
		if item.Error == 0 {
			json.Unmarshal([]byte(*item.Data), &item.Order)
			item.Data = nil
		}
	}

	return nil, resp.Data
}

func (this *PloRest) SelfTrade(reqOrders []OrderReq) (error) {
	data, _ := json.MarshalIndent(reqOrders, "", "  ")
	println(string(data))

	ts := util.Tick()
	bytes, _ := json.Marshal(reqOrders)
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("selftrade", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+SELF_TRADE_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err
	}

	var resp struct {
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg)
	}

	return nil
}

func (this *PloRest) SimpleSelfTrade(reqOrders []OrderReq) (error) {
	data, _ := json.MarshalIndent(reqOrders, "", "  ")
	println(string(data))

	ts := util.Tick()
	bytes, _ := json.Marshal(reqOrders)
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("simpleSelftrade", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+SIMPLE_SELF_TRADE_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err
	}
	println(string(bytes))
	var resp struct {
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg)
	}

	return nil
}

func (this *PloRest) CancelOrders(orderIds []string) (error, []error) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(data)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("cancel orders", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+CANCEL_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []struct {
			Error int 		`json:"err"`
			Msg string 		`json:"msg"`
		}	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	errors := make([]error, len(resp.Data))
	for i := range resp.Data {
		item := &resp.Data[i]
		if item.Error > 0 {
			errors[i] = fmt.Errorf("error: %d msg: %s", item.Error, item.Msg)
		}
	}

	return nil, errors
}

type PloOrder struct {
	AccountId string 			`json:"accountId"`
	OwnerType int 				`json:"ownerType"`
	Symbol string 				`json:"symbol"`
	Type string 				`json:"type"`
	Side string 				`json:"side"`
	ClientId string 			`json:"clientId"`
	Price decimal.Decimal 		`json:"price"`
	PosAction decimal.Decimal 	`json:"posAction"`
	AutoCancel int 				`json:"autoCancel"`
	CurrentQty decimal.Decimal 	`json:"currentQty"`
	TotalQty decimal.Decimal 	`json:"totalQty"`
	Status int 					`json:"status"`			// 订单状态(0取消，1未成交，2部分成交，3完全成交)
	Timestamp int64 			`json:"timestamp"`
	PosMargin decimal.Decimal 	`json:"posMargin"`
	OpenFee decimal.Decimal 	`json:"openFee"`
	CloseFee decimal.Decimal 	`json:"closeFee"`
	PosId string 				`json:"posId"`
	OrderId string 				`json:"orderId"`
}

func (this *PloRest) BatchOrders(orderIds []string) (error, []PloOrder) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(data)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("batch orders", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+BATCH_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []PloOrder    `json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}

func (this *PloRest) QueryOrders(pair goex.CurrencyPair, status int) (error, []PloOrder) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"status": status,
	}

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("query orders", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+ORDERS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data struct {
			Total int			`json:"total"`
			PerPage int 		`json:"per_page"`
			CurrentPage int 	`json:"current_page"`
			Data []PloOrder    	`json:"data"`
		}			`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data.Data
}

type PloPosition struct {
	PosId string 					`json:"posId"`
	Symbol string 					`json:"symbol"`
	AccountId string 				`json:"accountId"`
	OwnerType int 					`json:"ownerType"`
	Type string 					`json:"type"`
	OpenPrice decimal.Decimal 		`json:"openPrice"`
	ClosePrice decimal.Decimal 		`json:"closePrice"`
	AlarmPrice decimal.Decimal 		`json:"alarmPrice"`
	LiquidationPrice decimal.Decimal `json:"liquidationPrice"`
	BankruptcyPrice decimal.Decimal `json:"bankruptcyPrice"`
	TotalQty decimal.Decimal 		`json:"totalQty"`
	CurrentQty decimal.Decimal 		`json:"currentQty"`
	AvailableQty decimal.Decimal 	`json:"availableQty"`
	Margin decimal.Decimal 			`json:"margin"`
	Leverage int 					`json:"leverage"`
	RealisedPNL decimal.Decimal 	`json:"realisedPNL"`
	Status int 						`json:"status"`				// 0开仓单，1平仓单，2强平，3自动减仓4自动减仓对手方
	StopLossPrice decimal.Decimal 	`json:"stopLossPrice"`
	StopWinPrice decimal.Decimal 	`json:"stopWinPrice"`
	MaintMargin decimal.Decimal 	`json:"maintMargin"`
	TakerFee decimal.Decimal 		`json:"takerFee"`
	MakerFee decimal.Decimal 		`json:"makerFee"`
	Fund decimal.Decimal 			`json:"fund"`
	CreateTime int64 				`json:"createTime"`
	OpenTime int64 					`json:"openTime"`
	CloseTime int64 				`json:"closeTime"`
}

func (this *PloRest) QueryPositions(pair goex.CurrencyPair, status int) (error, []PloPosition) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"status": status,
	}
	fmt.Println(params)

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature
	println("query positions", message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+POSITIONS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}
	println(string(bytes))
	var resp struct {
		Data []PloPosition 	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}

func (this *PloRest) QueryPosRanking(pair goex.CurrencyPair, posType string, count int) (error, interface{}) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"type": posType,
		"count": count,
	}

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+POS_RANK_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []interface{} 	`json:"data"`
		Error int 	`json:"err"`
		Msg string 	`json:"msg"`
	}
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return err, nil
	}

	if resp.Error != 0 {
		return fmt.Errorf("error: %d msg: %s", resp.Error, resp.Msg), nil
	}

	return nil, resp.Data
}
