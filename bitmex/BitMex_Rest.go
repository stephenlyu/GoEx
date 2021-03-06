package bitmex

import (
	"net/http"
	"github.com/stephenlyu/GoEx"
	"strings"
	"encoding/json"
	"sort"
	"time"
	"fmt"
	"strconv"
	"github.com/qiniu/api.v6/url"
)

const (
	BASE_URL = "https://www.bitmex.com/api/v1"
	ROOT_URL = "/api/v1"
	TRADE_URL = "/trade"
	ORDERBOOK_URL = "/orderBook/L2"
	MARGIN_URL = "/user/margin"
	POSITION_GET_URL = "/position"
	TRADE_HISTORY_URL = "/execution/tradeHistory"
	ORDER_URL = "/order"
	ORDER_ALL_URL = "/order/all"
	WALLET_HISTORY_URL = "/user/walletHistory"
	INSTRUMENT_INDICES_URL = "/instrument/indices"
)

type BitMexRest struct {
	apiKey string
	apiSecretKey string
	client *http.Client
}

func NewBitMexRest(apiKey string, apiSecretKey string) *BitMexRest {
	return &BitMexRest{
		apiKey: apiKey,
		apiSecretKey: apiSecretKey,

		client: http.DefaultClient,
	}
}

func (bitmex *BitMexRest) map2Query(params map[string]string) string {
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

func (bitmex *BitMexRest) buildSigHeader(method string, path string, data string) map[string]string {
	now := time.Now().Unix()
	expires := now + 30
	signature := BuildSignature(bitmex.apiSecretKey, method, ROOT_URL + path, expires, data)
	return map[string]string{
		"api-key": bitmex.apiKey,
		"api-signature": signature,
		"api-expires": fmt.Sprintf("%d", expires),
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

func (BitMexRest *BitMexRest) handleRespHeader(header http.Header) {
	//"x-ratelimit-limit": 300
	//"x-ratelimit-remaining": 297
	//"x-ratelimit-reset": 1489791662
	if header == nil {
		return
	}
	fmt.Printf("x-ratelimit-limit: %s x-ratelimit-remaining: %s x-ratelimit-reset: %s\n",
		header.Get("x-ratelimit-limit"),
		header.Get("x-ratelimit-remaining"),
		header.Get("x-ratelimit-reset"))
}

func (bitmex *BitMexRest) GetTrade(symbol string, reverse bool) (error, []goex.Trade) {
	filter := map[string]string {
		"symbol": symbol,
	}
	bytes, _ := json.Marshal(filter)
	params := map[string]string {
		"filter": string(bytes),
		"count": "1",
		"reverse": strconv.FormatBool(reverse),
	}

	var data []struct {
		Timestamp string
		Symbol string
		Side string
		Size float64
		Price float64
	}

	query := bitmex.map2Query(params)
	query = url.Escape(query)
	header := bitmex.buildSigHeader("GET", TRADE_URL + "?" + query, "")
	header = map[string]string{}
	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+TRADE_URL+"?"+ query, header, &data)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	ret := make([]goex.Trade, len(data))
	for i := range data {
		r := &data[i]
		_, ts := ParseTimestamp(r.Timestamp)
		ret[i] = goex.Trade{
			Tid: ts,
			Type: strings.ToLower(r.Side),
			Amount: r.Size,
			Price: r.Price,
			Date: ts,
		}
	}

	return nil, ret
}

func (bitmex *BitMexRest) GetOrderBook(symbol string) (error, *goex.Depth) {
	params := map[string]string{"symbol": symbol, "depth": "10"}

	var data []struct {
		Symbol string
		Id int64
		Side string
		Size int64
		Price float64
	}

	query := bitmex.map2Query(params)
	query = url.Escape(query)
	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+ORDERBOOK_URL+"?"+ query, map[string]string{}, &data)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	var depth goex.Depth

	for _, r := range data {
		if r.Side == "Sell" {
			depth.AskList = append(depth.AskList, goex.DepthRecord{Price: r.Price, Amount: float64(r.Size)})
		} else if r.Side == "Buy" {
			depth.BidList = append(depth.BidList, goex.DepthRecord{Price: r.Price, Amount: float64(r.Size)})
		}
	}

	sort.SliceStable(depth.AskList, func(i,j int) bool {
		return depth.AskList[i].Price < depth.AskList[j].Price
	})

	return nil, &depth
}

func (bitmex *BitMexRest) GetAccount() (error, *goex.FutureAccount) {
	params := map[string]string{"currency":"XBt"}
	query := bitmex.map2Query(params)
	query = url.Escape(query)
	header := bitmex.buildSigHeader("GET", MARGIN_URL + "?" + query, "")

	var margin Margin

	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+MARGIN_URL+"?"+query, header, &margin)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	return nil, margin.ToFutureAccount()
}

func (bitmex *BitMexRest) GetPosition(symbol string, count int) (error, []goex.FuturePosition) {
	filter := map[string]string {
		"symbol": symbol,
	}
	bytes, _ := json.Marshal(filter)

	params := map[string]string{"filter":string(bytes), "count": fmt.Sprintf("%d", count)}
	query := bitmex.map2Query(params)
	query = url.Escape(query)
	header := bitmex.buildSigHeader("GET", POSITION_GET_URL + "?" + query, "")
	var positions []BitmexPosition

	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+POSITION_GET_URL+"?"+query, header, &positions)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	var ret []goex.FuturePosition = make([]goex.FuturePosition, len(positions))

	for i, p := range positions {
		ret[i] = *p.ToFuturePosition()
	}

	return nil, ret
}

func (bitmex *BitMexRest) PlaceOrder(symbol string, side goex.TradeSide, price float64, orderQty int, clientOrderId string) (error, *goex.FutureOrder) {
	var _side, ordType string
	switch side {
	case goex.SELL:
		_side = "Sell"; ordType = "Limit"
	case goex.BUY:
		_side = "Buy"; ordType = "Limit"
	case goex.SELL_MARKET:
		_side = "Sell"; ordType = "Market"
	case goex.BUY_MARKET:
		_side = "Buy"; ordType = "Market"
	}

	params := map[string]string{
		"symbol": symbol,
		"side": _side,
		"ordType": ordType,
		"orderQty": fmt.Sprintf("%d", orderQty),
		"clOrdID": clientOrderId,
	}
	if price != 0 {
		params["price"] = fmt.Sprintf("%.f", price)
	}
	data := bitmex.map2Query(params)
	data = url.Escape(data)
	header := bitmex.buildSigHeader("POST", ORDER_URL, data)

	bytes, respHeader, err := goex.NewHttpRequestEx(bitmex.client, "POST", BASE_URL+ORDER_URL, data, header)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	var order BitmexOrder
	err = json.Unmarshal(bytes, &order)
	if err != nil {
		return err, nil
	}

	return err, order.ToFutureOrder()
}

func (bitmex *BitMexRest) CancelOrder(orderId string, clientOrderId string) (error, *goex.FutureOrder) {
	params := map[string]string {
	}
	if orderId != "" {
		params["orderID"] = orderId
	}
	if clientOrderId != "" {
		params["clOrdID"] = clientOrderId
	}
	data := bitmex.map2Query(params)
	header := bitmex.buildSigHeader("DELETE", ORDER_URL, data)
	data = url.Escape(data)
	bytes, respHeader, err := goex.NewHttpRequestEx(bitmex.client, "DELETE", BASE_URL + ORDER_URL, data, header)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	var orders []BitmexOrder
	err = json.Unmarshal(bytes, &orders)
	if err != nil {
		return err, nil
	}

	return err, orders[0].ToFutureOrder()
}

func (bitmex *BitMexRest) CancelAll() (error, []goex.FutureOrder) {
	params := map[string]string {
	}
	data := bitmex.map2Query(params)
	header := bitmex.buildSigHeader("DELETE", ORDER_ALL_URL, data)
	data = url.Escape(data)
	bytes, respHeader, err := goex.NewHttpRequestEx(bitmex.client, "DELETE", BASE_URL + ORDER_ALL_URL, data, header)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	var orders []BitmexOrder
	err = json.Unmarshal(bytes, &orders)
	if err != nil {
		return err, nil
	}

	ret := make([]goex.FutureOrder, len(orders))
	for i := range orders {
		ret[i] = *orders[i].ToFutureOrder()
	}

	return err, ret
}

func (bitmex *BitMexRest) ListOrders(symbol string, openOnly bool, startTime, endTime string, count int) (error, []goex.FutureOrder) {
	params := map[string]string{
		"symbol": symbol,
	}
	if openOnly {
		params["filter"] = `{"open": true}`
	}
	if startTime != "" {
		params["startTime"] = startTime
	}
	if endTime != "" {
		params["endTime"] = endTime
	}
	if count > 0 {
		params["count"] = strconv.Itoa(count)
	}
	query := bitmex.map2Query(params)
	header := bitmex.buildSigHeader("GET", ORDER_URL + "?" + query, "")
	query = url.Escape(query)

	var orders []BitmexOrder

	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+ORDER_URL+"?"+query, header, &orders)
	bitmex.handleRespHeader(respHeader)

	ret := make([]goex.FutureOrder, len(orders))
	for i := range orders {
		ret[i] = *orders[i].ToFutureOrder()
	}

	return err, ret
}

func (bitmex *BitMexRest) ListFills(symbol string, startTime, endTime string, count int) (error, []goex.FutureFill) {
	params := map[string]string{
		"symbol": symbol,
	}
	if startTime != "" {
		params["startTime"] = startTime
	}
	if endTime != "" {
		params["endTime"] = endTime
	}
	if count > 0 {
		params["count"] = strconv.Itoa(count)
	}
	query := bitmex.map2Query(params)
	header := bitmex.buildSigHeader("GET", TRADE_HISTORY_URL + "?" + query, "")
	query = url.Escape(query)

	var executions []Execution

	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+TRADE_HISTORY_URL+"?"+query, header, &executions)
	bitmex.handleRespHeader(respHeader)

	ret := make([]goex.FutureFill, len(executions))
	for i := range executions {
		ret[i] = *executions[i].ToFill()
	}

	return err, ret
}

type WalletTransaction struct {
	Account int64 		`json:"account"`
	Address string 		`json:"address"`
	Amount int64 		`json:"amount"`
	Currency string 	`json:"currency"`
	Fee float64 		`json:"fee"`
	MarginBalance int64 `json:"marginBalance"`
	Text string 		`json:"text"`
	Timestamp string 	`json:"timestamp"`
	TransactId string 	`json:"transactID"`
	TransactStatus string `json:"transactStatus"`
	TransactTime string `json:"transactTime"`
	TransactType string `json:"transactType"`
	Tx string 			`json:"tx"`
	WalletBalance int64 `json:"walletBalance"`
}

func (bitmex *BitMexRest) GetWalletHistory(start, count int) (error, []WalletTransaction) {
	params := map[string]string{"currency":"XBt", "count": strconv.Itoa(count), "start": strconv.Itoa(start)}
	query := bitmex.map2Query(params)
	query = url.Escape(query)
	header := bitmex.buildSigHeader("GET", WALLET_HISTORY_URL + "?" + query, "")
	var history []WalletTransaction

	err, respHeader := goex.HttpGet5(bitmex.client, BASE_URL+WALLET_HISTORY_URL+"?"+query, header, &history)
	bitmex.handleRespHeader(respHeader)
	if err != nil {
		return err, nil
	}

	return nil, history
}
