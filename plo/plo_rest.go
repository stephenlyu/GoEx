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
)

const (
	BASE_URL = "https://api.plo.one/"
	TRADE_URL = "/m_api/trade"
	ORDER_BOOK_URL = "/m_api/orderbookL2"
	CONFIG_LIST_URL = "/hapi/Config/ConfList"
	BALANCES_URL = "/hapi/BatchOperation/balances"
	PLACE_ORDER_URL = "/hapi/BatchOperation/batchPosExec"
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

func (this *PloRest) GetTrade(pair goex.CurrencyPair) (error, interface{}) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data interface{}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+TRADE_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
}

func (this *PloRest) GetOrderBook(pair goex.CurrencyPair) (error, interface{}) {
	symbol := fmt.Sprintf("%s%s", pair.CurrencyA, pair.CurrencyB)
	params := map[string]string{
		"symbol": symbol,
	}

	var data interface{}
	query := this.map2Query(params)
	err := goex.HttpGet4(this.client, BASE_URL+ORDER_BOOK_URL+"?"+ query, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
}

func (this *PloRest) GetConfigList() (error, interface{}) {
	var data interface{}
	err := goex.HttpGet4(this.client, BASE_URL+CONFIG_LIST_URL, map[string]string{}, &data)
	if err != nil {
		return err, nil
	}

	return nil, data
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
}

type OrderResp struct {
	Error int 				`json:"err"`
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
	ts := util.Tick()
	bytes, _ := json.Marshal(reqOrders)
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+PLACE_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	fmt.Println(string(bytes))

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

func (this *PloRest) CancelOrders(orderIds []string) (error, []error) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(map[string]interface{}{"data": data})
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

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

type _PloOrder struct {
	AccountId string 		`json:"accountId"`
	OwnerType int 			`json:"ownerType"`
	Symbol string 			`json:"symbol"`
	Type string 			`json:"type"`
	Side string 			`json:"side"`
	ClientId string 		`json:"clientId"`
	Price string 			`json:"price"`
	PosAction string 		`json:"posAction"`
	AutoCancel int 			`json:"autoCancel"`
	CurrentQty string 		`json:"currentQty"`
	TotalQty string 		`json:"totalQty"`
	Status int 				`json:"status"`			// 订单状态(0取消，1未成交，2部分成交，3完全成交)
	Timestamp int64 		`json:"timestamp"`
	PosMargin string 		`json:"posMargin"`
	OpenFee string 			`json:"openFee"`
	CloseFee string 		`json:"closeFee"`
	PosId string 			`json:"posId"`
	OrderId string 			`json:"orderId"`
}

func (o *_PloOrder) ToPloOrder() PloOrder {
	posAction, _ := strconv.Atoi(o.PosAction)
	price, _ := strconv.ParseFloat(o.Price, 64)
	currentQty, _ := strconv.ParseFloat(o.CurrentQty, 64)
	totalQty, _ := strconv.ParseFloat(o.TotalQty, 64)
	posMargin, _ := strconv.ParseFloat(o.PosMargin, 64)
	openFee, _ := strconv.ParseFloat(o.OpenFee, 64)
	closeFee, _ := strconv.ParseFloat(o.CloseFee, 64)

	return PloOrder{
		AccountId: o.AccountId,
		OwnerType: o.OwnerType,
		Symbol: o.Symbol,
		Type: o.Type,
		Side: o.Type,
		ClientId: o.ClientId,
		Price: price,
		PosAction: posAction,
		AutoCancel: o.AutoCancel,
		CurrentQty: currentQty,
		TotalQty: totalQty,
		Status: o.Status,
		Timestamp: o.Timestamp,
		PosMargin: posMargin,
		OpenFee: openFee,
		CloseFee: closeFee,
		PosId: o.PosId,
		OrderId: o.OrderId,
	}
}

type PloOrder struct {
	AccountId string 		`json:"accountId"`
	OwnerType int 			`json:"ownerType"`
	Symbol string 			`json:"symbol"`
	Type string 			`json:"type"`
	Side string 			`json:"side"`
	ClientId string 		`json:"clientId"`
	Price float64 			`json:"price"`
	PosAction int 			`json:"posAction"`
	AutoCancel int 			`json:"autoCancel"`
	CurrentQty float64 		`json:"currentQty"`
	TotalQty float64 		`json:"totalQty"`
	Status int 				`json:"status"`			// 订单状态(0取消，1未成交，2部分成交，3完全成交)
	Timestamp int64 		`json:"timestamp"`
	PosMargin float64 		`json:"posMargin"`
	OpenFee float64 			`json:"openFee"`
	CloseFee float64 		`json:"closeFee"`
	PosId string 			`json:"posId"`
	OrderId string 			`json:"orderId"`
}

func (this *PloRest) BatchOrders(orderIds []string) (error, []PloOrder) {
	data := make([]map[string]string, len(orderIds))
	for i, orderId := range orderIds {
		data[i] = map[string]string {
			"orderId": orderId,
		}
	}

	bytes, _ := json.Marshal(map[string]interface{}{"data": data})
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+BATCH_ORDER_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []_PloOrder    `json:"data"`
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

	ret := make([]PloOrder, len(resp.Data))
	for i := range resp.Data {
		ret[i] = resp.Data[i].ToPloOrder()
	}

	return nil, ret
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

	print(message)

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+ORDERS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	var resp struct {
		Data []_PloOrder    `json:"data"`
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

	ret := make([]PloOrder, len(resp.Data))
	for i := range resp.Data {
		ret[i] = resp.Data[i].ToPloOrder()
	}

	return nil, ret
}

type _PloPosition struct {
	PosId string 			`json:"posId"`
	Symbol string 			`json:"symbol"`
	AccountId string 		`json:"accountId"`
	OwnerType int 			`json:"ownerType"`
	Type string 			`json:"type"`
	OpenPrice string 		`json:"openPrice"`
	ClosePrice string 		`json:"closePrice"`
	AlarmPrice string 		`json:"alarmPrice"`
	LiquidationPrice string `json:"liquidationPrice"`
	BankruptcyPrice string 	`json:"bankruptcyPrice"`
	TotalQty string 		`json:"totalQty"`
	CurrentQty string 		`json:"currentQty"`
	AvailableQty string 	`json:"availableQty"`
	Margin string 			`json:"margin"`
	Leverage int 			`json:"leverage"`
	RealisedPNL string 		`json:"realisedPNL"`
	Status int 				`json:"status"`				// 0开仓单，1平仓单，2强平，3自动减仓4自动减仓对手方
	StopLossPrice string 	`json:"stopLossPrice"`
	StopWinPrice string 	`json:"stopWinPrice"`
	MaintMargin string 		`json:"maintMargin"`
	TakerFee string 		`json:"takerFee"`
	MakerFee string 		`json:"makerFee"`
	Fund string 			`json:"fund"`
	CreateTime int64 		`json:"createTime"`
	OpenTime int64 			`json:"openTime"`
	CloseTime int64 		`json:"closeTime"`
}

func (p *_PloPosition) ToPloPosition() PloPosition {
	toFloat := func (s string) float64 {
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	return PloPosition{
		PosId: p.PosId,
		Symbol: p.Symbol,
		AccountId: p.AccountId,
		OwnerType: p.OwnerType,
		Type: p.Type,
		OpenPrice: toFloat(p.OpenPrice),
		ClosePrice: toFloat(p.ClosePrice),
		AlarmPrice: toFloat(p.AlarmPrice),
		LiquidationPrice: toFloat(p.LiquidationPrice),
		BankruptcyPrice: toFloat(p.BankruptcyPrice),
		TotalQty: toFloat(p.TotalQty),
		CurrentQty: toFloat(p.CurrentQty),
		AvailableQty: toFloat(p.AvailableQty),
		Margin: toFloat(p.MaintMargin),
		Leverage: p.Leverage,
		RealisedPNL: toFloat(p.RealisedPNL),
		Status: p.Status,
		StopLossPrice: toFloat(p.StopLossPrice),
		StopWinPrice: toFloat(p.StopWinPrice),
		MaintMargin: toFloat(p.MaintMargin),
		TakerFee: toFloat(p.TakerFee),
		MakerFee: toFloat(p.MakerFee),
		Fund: toFloat(p.Fund),
		CreateTime: p.CreateTime,
		OpenTime: p.OpenTime,
		CloseTime: p.CloseTime,
	}
}

type PloPosition struct {
	PosId string 			`json:"posId"`
	Symbol string 			`json:"symbol"`
	AccountId string 		`json:"accountId"`
	OwnerType int 			`json:"ownerType"`
	Type string 			`json:"type"`
	OpenPrice float64 		`json:"openPrice"`
	ClosePrice float64 		`json:"closePrice"`
	AlarmPrice float64 		`json:"alarmPrice"`
	LiquidationPrice float64 `json:"liquidationPrice"`
	BankruptcyPrice float64 `json:"bankruptcyPrice"`
	TotalQty float64 		`json:"totalQty"`
	CurrentQty float64 		`json:"currentQty"`
	AvailableQty float64 	`json:"availableQty"`
	Margin float64 			`json:"margin"`
	Leverage int 			`json:"leverage"`
	RealisedPNL float64 	`json:"realisedPNL"`
	Status int 				`json:"status"`				// 0开仓单，1平仓单，2强平，3自动减仓4自动减仓对手方
	StopLossPrice float64 	`json:"stopLossPrice"`
	StopWinPrice float64 	`json:"stopWinPrice"`
	MaintMargin float64 	`json:"maintMargin"`
	TakerFee float64 		`json:"takerFee"`
	MakerFee float64 		`json:"makerFee"`
	Fund float64 			`json:"fund"`
	CreateTime int64 		`json:"createTime"`
	OpenTime int64 			`json:"openTime"`
	CloseTime int64 		`json:"closeTime"`
}

func (this *PloRest) QueryPositions(pair goex.CurrencyPair, status int) (error, []PloPosition) {
	params := map[string]interface{} {
		"symbol": pair.ToSymbol(""),
		"status": status,
	}

	bytes, _ := json.Marshal(params)
	ts := util.Tick()
	message, signature := BuildSignature(this.apiKey, this.apiSecretKey, ts, base64.StdEncoding.EncodeToString(bytes))

	message += "&sign=" + signature

	bytes, err := goex.HttpPostForm3(this.client, BASE_URL+POSITIONS_URL, message, map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err, nil
	}

	fmt.Println(string(bytes))

	var resp struct {
		Data []_PloPosition 	`json:"data"`
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

	ret := make([]PloPosition, len(resp.Data))
	for i := range resp.Data {
		ret[i] = resp.Data[i].ToPloPosition()
	}

	return nil, ret
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

	fmt.Println(string(bytes))

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
