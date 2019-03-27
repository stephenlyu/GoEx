package gateiospot

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"time"
	"sort"
	"github.com/shopspring/decimal"
	"strconv"
	"sync/atomic"
	"strings"
)

const _LOGIN_ID = int64(0xFFFFFFFFFF)

var __id int64 = 0

func _NextId() int64 {
	return atomic.AddInt64(&__id, 1)
}

func _ParseId(id interface{}) int64 {
	if id == nil {
		return 0
	}
	var ret int64
	switch id.(type) {
	case int:
		ret = int64(id.(int))
	case int64:
		ret = id.(int64)
	case float64:
		ret = int64(id.(float64))
	default:
		panic("bad id")
	}
	return ret
}

func (this *GateIOSpot) createWsConn() {
	if this.ws == nil {
		//connect wsx
		this.createWsLock.Lock()
		defer this.createWsLock.Unlock()

		if this.ws == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(CurrencyPair, []TradeDecimal))
			this.wsAccountHandleMap = make(map[string]func(*AccountDecimal))
			this.wsOrderHandleMap = make(map[string]func(*OrderDecimal))
			this.depthManagers = make(map[string]*DepthManager)

			this.ws = NewWsConn("wss://ws.gate.io/v3/")
			this.ws.SetErrorHandler(this.errorHandle)
			this.ws.Heartbeat(func() interface{} {
				return map[string]interface{} {
					"id": _NextId(),
					"method": "server.ping",
					"params": []interface{}{},
				}
			}, 20*time.Second)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//println(string(msg))
				var data struct {
					Id interface{}
					Error *struct {
						Code int
						Message string
					}
					Result interface{}
					Method string
					Params interface{}
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if result, ok := data.Result.(string); ok {
					if result == "pong" {
						this.ws.UpdateActivedTime()
						return
					}
				}

				id := _ParseId(data.Id)
				if id == _LOGIN_ID {
					var err error
					if data.Error != nil {
						err = fmt.Errorf("code: %d message: %s", data.Error.Code, data.Error.Message)
					}
					if this.wsLoginHandle != nil {
						this.wsLoginHandle(err)
					}
					return
				}

				switch data.Method  {
				case "trades.update":
				symbol, trades := this.parseTrade(msg)
				if symbol != "" {
					topic := "trades.subscribe"
					this.wsTradeHandleMap[topic](NewCurrencyPair2(symbol), trades)
				}
				case "depth.update":
					depth := this.parseDepth(msg)
					if depth != nil {
						topic := "depth.subscribe"
						this.wsDepthHandleMap[topic](depth)
					}
				case "balance.update":
					account := this.parseAccount(msg)
					if account != nil {
						topic := "balance.subscribe"
						this.wsAccountHandleMap[topic](account)
					}
				case "order.update":
					order := this.parseOrder(msg)
					if order != nil {
						topic := "order.subscribe"
						this.wsOrderHandleMap[topic](order)
					}
				}
			})
		}
	}
}

func (this *GateIOSpot) Login(handle func(error)) error {
	this.createWsConn()
	this.wsLoginHandle = handle

	nonce := time.Now().UnixNano() / 1000000
	sign, _ := GetParamHmacSHA512Base64SignEx(this.apiSecretKey, strconv.FormatInt(nonce, 10))
	params := []interface{}{this.apiKey, sign, nonce}

	return this.ws.Subscribe(map[string]interface{}{
		"id":   _LOGIN_ID,
		"method": "server.sign",
		"params": params,
	})
}

func (this *GateIOSpot) GetDepthWithWs(pairs []CurrencyPair, intervals []float64, limit int, handle func(*DepthDecimal)) error {
	if len(pairs) != len(intervals) {
		panic("len(pairs) != len(intervals)")
	}
	this.createWsConn()

	params := make([][]interface{}, len(pairs))
	for i := range pairs {
		symbol := pairs[i].ToSymbol("_")
		interval := decimal.NewFromFloat(intervals[i])
		params[i] = []interface{} {
			symbol,
			limit,
			interval.String(),
		}
		this.depthManagers[symbol] = NewDepthManager()
	}

	method := "depth.subscribe"
	this.wsDepthHandleMap[method] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"id": _NextId(),
		"method":   method,
		"params": params})
}

func (this *GateIOSpot) GetTradeWithWs(pairs []CurrencyPair, handle func(CurrencyPair, []TradeDecimal)) error {
	this.createWsConn()

	symbols := make([]string, len(pairs))
	for i := range pairs {
		symbol := pairs[i].ToSymbol("_")
		symbols[i] = symbol
	}

	method := "trades.subscribe"
	this.wsTradeHandleMap[method] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"id": _NextId(),
		"method":   method,
		"params": symbols})
}

func (this *GateIOSpot) GetAccountWithWs(currencies []Currency, handle func(*AccountDecimal)) error {
	this.createWsConn()

	params := make([]string, len(currencies))
	for i, c := range currencies {
		params[i] = c.Symbol
	}

	method := "balance.subscribe"
	this.wsAccountHandleMap[method] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"id":   _NextId(),
		"method": method,
		"params": params})
}

func (this *GateIOSpot) GetOrderWithWs(pairs []CurrencyPair, handle func(*OrderDecimal)) error {
	this.createWsConn()

	params := make([]string, len(pairs))
	for i := range pairs {
		symbol := pairs[i].ToSymbol("_")
		params[i] = symbol
	}
	
	method := "order.subscribe"
	this.wsOrderHandleMap[method] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"id":   _NextId(),
		"method": method,
		"params": params,
	})
}

func (this *GateIOSpot) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data *struct {
		Method  string
		Params []interface{}
	}

	json.Unmarshal(msg, &data)

	symbol := data.Params[0].(string)

	bytes, _ := json.Marshal(data.Params[1])
	var trades []struct{
		Id int64
		Time float64
		Price decimal.Decimal
		Amount decimal.Decimal
		Type string
	}
	json.Unmarshal(bytes, &trades)

	ret := make([]TradeDecimal, len(trades))
	for i := range trades {
		o := &trades[i]
		ret[i] = TradeDecimal {
			Tid: o.Id,
			Type: o.Type,
			Amount: o.Amount,
			Price: o.Price,
			Date: int64(o.Time * 1000),
		}
	}

	return symbol, ret
}

func (this *GateIOSpot) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Params []interface{}
	}

	json.Unmarshal(msg, &data)

	clean := data.Params[0].(bool)
	symbol := data.Params[2].(string)
	pair := NewCurrencyPair2(symbol)

	bytes, _ := json.Marshal(data.Params[1])

	var depthData struct {
		Asks [][]decimal.Decimal
		Bids [][]decimal.Decimal
	}
	json.Unmarshal(bytes, &depthData)

	depthManager, _ := this.depthManagers[symbol]
	if depthManager == nil {
		panic("Illegal state error")
	}

	asks, bids := depthManager.Update(clean, depthData.Asks, depthData.Bids)
	return &DepthDecimal{
		Pair: pair,
		AskList: asks,
		BidList: bids,
	}
}

func (this *GateIOSpot) parseAccount(msg []byte) *AccountDecimal {
	var data *struct {
		Params []map[string]struct {
			Available decimal.Decimal
			Freeze decimal.Decimal
		}
	}

	json.Unmarshal(msg, &data)

	ret := new(AccountDecimal)
	ret.SubAccounts = make(map[Currency]SubAccountDecimal)

	for _, p := range data.Params {
		for key, o := range p {
			currency := NewCurrency(strings.ToUpper(key), "")
			ret.SubAccounts[currency] = SubAccountDecimal{
				Currency: currency,
				AvailableAmount: o.Available,
				FrozenAmount: o.Freeze,
				Amount: o.Available.Add(o.Freeze),
			}
			break
		}
	}

	return ret
}

func (this *GateIOSpot) parseOrder(msg []byte) *OrderDecimal {
	var data *struct {
		Params []interface{}
	}

	json.Unmarshal(msg, &data)
	if len(data.Params) != 2 {
		return nil
	}
	event := int(data.Params[0].(float64))

	bytes,_ := json.Marshal(data.Params[1])
	var order struct{
		Id decimal.Decimal
		Market string
		User int64
		CTime decimal.Decimal
		MTime decimal.Decimal
		Price decimal.Decimal
		Amount decimal.Decimal
		Left decimal.Decimal
		DealFee decimal.Decimal
		OrderType int
		Type int
		FilledAmount decimal.Decimal
		FilledTotal decimal.Decimal
	}
	json.Unmarshal(bytes, &order)

	var status TradeStatus
	switch event {
	case 1, 2:
		if order.FilledAmount.IsPositive() {
			status = ORDER_PART_FINISH
		} else {
			status = ORDER_UNFINISH
		}
	case 3:
		if order.FilledAmount.LessThan(order.Amount) {
			status = ORDER_CANCEL
		} else {
			status = ORDER_FINISH
		}
	}

	ret := new(OrderDecimal)

	ret.OrderID2 = order.Id.String()
	ret.Currency = NewCurrencyPair2(strings.ToUpper(order.Market))
	ret.Timestamp = order.CTime.Mul(decimal.New(1000, 0)).IntPart()
	ret.Price = order.Price
	ret.Amount = order.Amount
	ret.Fee = order.DealFee
	switch order.Type {
	case 1:
		ret.Side = SELL
	case 2:
		ret.Side = BUY
	default:
		panic("bad order type")
	}
	ret.Status = status
	ret.DealAmount = order.FilledAmount
	if order.FilledAmount.IsPositive() {
		ret.AvgPrice = order.FilledTotal.Div(order.FilledAmount)
	}

	return ret
}

func (this *GateIOSpot) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}

func (this *GateIOSpot) CloseWs() {
	this.ws.CloseWs()
}

type DepthManager struct {
	buyMap map[string][]decimal.Decimal
	sellMap map[string][]decimal.Decimal
}

func NewDepthManager() *DepthManager {
	return &DepthManager{
		buyMap: make(map[string][]decimal.Decimal),
		sellMap: make(map[string][]decimal.Decimal),
	}
}

func (this *DepthManager) Update(clean bool, askList, bidList [][]decimal.Decimal) (DepthRecordsDecimal, DepthRecordsDecimal) {
	if clean {
		this.buyMap = make(map[string][]decimal.Decimal)
		this.sellMap = make(map[string][]decimal.Decimal)
	}

	for _, o := range askList {
		price := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.sellMap, price)
		} else {
			this.sellMap[price] = o
		}
	}

	for _, o := range bidList {
		price := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.buyMap, price)
		} else {
			this.buyMap[price] = o
		}
	}

	bids := make(DepthRecordsDecimal, len(this.buyMap))
	i := 0
	for _, item := range this.buyMap {
		bids[i] = DepthRecordDecimal{Price: item[0], Amount: item[1]}
		i++
	}
	sort.SliceStable(bids, func(i,j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	asks := make(DepthRecordsDecimal, len(this.sellMap))
	i = 0
	for _, item := range this.sellMap {
		asks[i] = DepthRecordDecimal{Price: item[0], Amount: item[1]}
		i++
	}
	sort.SliceStable(asks, func(i,j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})
	return asks, bids
}
