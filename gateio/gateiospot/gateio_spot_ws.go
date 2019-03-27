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
			this.wsAccountHandleMap = make(map[string]func(*SubAccountDecimal))
			this.wsOrderHandleMap = make(map[string]func([]OrderDecimal))
			this.depthManagers = make(map[string]*DepthManager)

			this.ws = NewWsConn("wss://ws.gate.io/v3/")
			this.ws.Heartbeat(func() interface{} {
				return map[string]interface{} {
					"id": _NextId(),
					"method": "server.ping",
					"params": []interface{}{},
				}
			}, 20*time.Second)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				println(string(msg))
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
				//case "balance.update":
				//	account := this.parseAccount(msg)
				//	if account != nil {
				//		channel := fmt.Sprintf("%s:%s", data.Table, account.Currency)
				//		this.wsAccountHandleMap[channel](account)
				//	}
				//case "order.update":
				//	instrumentId, orders := this.parseOrder(msg)
				//	if orders != nil {
				//		topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
				//		this.wsOrderHandleMap[topic](orders)
				//	}
				}
			})
		}
	}
}

func (this *GateIOSpot) Login(handle func(error)) error {
	this.createWsConn()
	this.wsLoginHandle = handle

	nonce := time.Now().UnixNano() / 1000000
	sign, _ := GetParamHmacSHA256Base64Sign(this.apiSecretKey, strconv.FormatInt(nonce, 10))

	return this.ws.Subscribe(map[string]interface{}{
		"id":   _LOGIN_ID,
		"method": "server.sign",
		"params": []interface{}{this.apiKey, sign, nonce},
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

func (okSpot *GateIOSpot) GetAccountWithWs(currency Currency, handle func(*SubAccountDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/account:%s", currency.Symbol)
	okSpot.wsAccountHandleMap[channel] = handle
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okSpot *GateIOSpot) GetOrderWithWs(instrumentId string, handle func([]OrderDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/order:%s", instrumentId)
	okSpot.wsOrderHandleMap[channel] = handle
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
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

func (this *GateIOSpot) parseAccount(msg []byte) *SubAccountDecimal {
	var data *struct {
		Table  string
		Action string
		Data   []struct {
			Balance decimal.Decimal
			Available decimal.Decimal
			Hold decimal.Decimal
			Id string
			Currency string
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Data[0]
	currency := Currency{Symbol: r.Currency}
	return &SubAccountDecimal{
		Currency: currency,
		Amount: r.Balance,
		AvailableAmount: r.Available,
		FrozenAmount: r.Hold,
	}
}

func (this *GateIOSpot) parseOrder(msg []byte) (string, []OrderDecimal) {
	return "", nil
	//var data *struct {
	//	Table  string
	//	Action string
	//}
	//
	//json.Unmarshal(msg, &data)
	//
	//instrumentId := data.Data[0].InstrumentId
	//
	//ret := make([]OrderDecimal, len(data.Data))
	//for i := range data.Data {
	//	ret[i] = *data.Data[i].ToOrder()
	//}
	//
	//return instrumentId, ret
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
