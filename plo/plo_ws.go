package plo

import (
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"time"
	"sync"
	"encoding/json"
	"log"
	"github.com/stephenlyu/tds/util"
	"strings"
	"sort"
	"github.com/shopspring/decimal"
)

type PloWs struct {
	apiKey,
	apiSecretKey     string
	ws               *WsConn
	createWsLock     sync.Mutex

	depthManagers	 map[string]*DepthManager

	wsDepthHandleMap map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(CurrencyPair, bool, []TradeDecimal)
	authHandle 		 func()
	orderHandle      func([]PloOrder)
	accountHandle    func(*FutureAccount)
	positionHandle   func([]PloPosition)
	errorHandle      func(error)
}

func NewPloWs(apiKey, apiSecretyKey string) *PloWs {
	return &PloWs{apiKey: apiKey, apiSecretKey: apiSecretyKey}
}

func (ploWs *PloWs) createWsConn() {
	if ploWs.ws == nil {
		//connect wsx
		ploWs.createWsLock.Lock()
		defer ploWs.createWsLock.Unlock()

		if ploWs.ws == nil {
			ploWs.depthManagers = make(map[string]*DepthManager)
			ploWs.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			ploWs.wsTradeHandleMap = make(map[string]func(CurrencyPair, bool, []TradeDecimal))

			ploWs.ws = NewWsConn("wss://api.plo.one/ws")
			ploWs.ws.SetErrorHandler(ploWs.errorHandle)
			ploWs.ws.ReConnect()
			ploWs.ws.Heartbeat(func() interface{} {
				t := time.Now()
				return map[string]interface{}{"ping": t.UnixNano() / 1000000}
				}, 30*time.Second)
			ploWs.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//println(string(msg))
				var resp struct {
					Success bool
					Subscribe string
					Table string
					Action string
					Ping int64
					Pong int64
				}

				err := json.Unmarshal(msg, &resp)
				if err != nil {
					log.Print(err)
					return
				}

				if resp.Pong > 0 {
					ploWs.ws.UpdateActivedTime()
					return
				} else if resp.Ping > 0 {
					ploWs.ws.UpdateActivedTime()
					ploWs.ws.WriteJSON(map[string]interface{}{"pong": resp.Ping})
					return
				}

				if resp.Subscribe != "" {
					return
				}

				switch resp.Table {
				case "connect":
					ploWs.authHandle()
				case "trade":
					symbol, trades := ploWs.parseTrade(msg)
					if symbol != "" {
						isIndex := strings.HasPrefix(symbol, ".")
						topic := fmt.Sprintf("trade:%s", symbol)
						symbol = strings.TrimLeft(symbol, ".")
						util.Assert(strings.HasSuffix(symbol, "USD"), "")
						pair := CurrencyPair{
							CurrencyA: Currency{Symbol: strings.TrimSuffix(symbol, "USD")},
							CurrencyB: USD,
						}
						ploWs.wsTradeHandleMap[topic](pair, isIndex, trades)
					}
				case "orderBookL2":
					depth := ploWs.parseDepth(msg)
					if depth != nil {
						topic := fmt.Sprintf("orderBookL2:%s", depth.Pair.ToSymbol(""))
						ploWs.wsDepthHandleMap[topic](depth)
					}
				case "order":
					orders := ploWs.parseOrder(msg)
					if len(orders) > 0 && ploWs.orderHandle != nil {
						ploWs.orderHandle(orders)
					}
				case "balance":
					account := ploWs.parseBalance(msg)
					if account != nil && ploWs.accountHandle != nil {
						ploWs.accountHandle(account)
					}
				case "position":
					positions := ploWs.parsePosition(msg)
					if len(positions) > 0 && ploWs.positionHandle != nil {
						ploWs.positionHandle(positions)
					}
				}
			})
		}
	}
}

func (ploWs *PloWs) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data struct {
		Data []struct {
			 Timestamp int64
			 Side string
			 Symbol string
			 Size decimal.Decimal
			 Price decimal.Decimal
			 }
	}
	json.Unmarshal(msg, &data)

	if len(data.Data) == 0 {
		return "", nil
	}

	ret := make([]TradeDecimal, len(data.Data))
	for i, r := range data.Data {
		ret[i] = TradeDecimal{
			Tid: r.Timestamp,
			Type: strings.ToLower(r.Side),
			Amount: r.Size,
			Price: r.Price,
			Date: r.Timestamp,
		}

	}

	return data.Data[0].Symbol, ret
}

type DepthItem struct {
	Symbol string
	Side string
	Price decimal.Decimal
	Size decimal.Decimal
}

func (ploWs *PloWs) parseDepth(msg []byte) *DepthDecimal {
	var data struct {
		Action string
		Data []DepthItem
	}
	json.Unmarshal(msg, &data)

	if len(data.Data) == 0 {
		return nil
	}

	symbol := data.Data[0].Symbol
	pair := CurrencyPair{
		CurrencyA: Currency{Symbol: strings.TrimSuffix(symbol, "USD")},
		CurrencyB: USD,
	}

	depthManager, ok := ploWs.depthManagers[symbol]
	if !ok {
		panic("no depth manager for " + symbol)
	}

	asks, bids := depthManager.Update(data.Action, data.Data)

	return &DepthDecimal{
		Pair: pair,
		AskList: asks,
		BidList: bids,
	}
}

func (ploWs *PloWs) parseBalance(msg []byte) *FutureAccount {
	var data struct {
		Data []PloBalance
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	if len(data.Data) == 0 {
		return nil
	}

	ret := new(FutureAccount)
	ret.FutureSubAccounts = make(map[Currency]FutureSubAccount)

	for _, r := range data.Data {
		sa := r.ToFutureSubAccount()
		ret.FutureSubAccounts[sa.Currency] = sa
	}
	return ret
}

func (ploWs *PloWs) parseOrder(msg []byte) []PloOrder {
	var data struct {
		Data []PloOrder
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		println(err.Error())
		return nil
	}

	return data.Data
}

func (ploWs *PloWs) parsePosition(msg []byte) []PloPosition {
	var data struct {
		Data []PloPosition
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	return data.Data
}

func (ploWs *PloWs) GetDepthWithWs(pair CurrencyPair, handle func(*DepthDecimal)) error {
	ploWs.createWsConn()
	symbol := pair.ToSymbol("")
	topic := fmt.Sprintf("orderBookL2:%s", symbol)
	ploWs.wsDepthHandleMap[topic] = handle
	ploWs.depthManagers[symbol] = NewDepthManager()
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (ploWs *PloWs) GetTradeWithWs(pair CurrencyPair, isIndex bool, handle func(CurrencyPair, bool, []TradeDecimal)) error {
	ploWs.createWsConn()
	var topic string
	if isIndex {
		topic = fmt.Sprintf("trade:.%s", pair.ToSymbol(""))
	} else {
		topic = fmt.Sprintf("trade:%s", pair.ToSymbol(""))
	}
	ploWs.wsTradeHandleMap[topic] = handle
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (ploWs *PloWs) Authenticate(handle func()) error {
	ploWs.createWsConn()
	ploWs.authHandle = handle
	ts := util.Tick()
	sign := BuildWsSignature(ploWs.apiKey, ploWs.apiSecretKey, ts)
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "connect",
		"accessKey": ploWs.apiKey,
		"ts": ts,
		"sign": sign,
	})
}

func (ploWs *PloWs) GetAccountWithWs(handle func(*FutureAccount)) error {
	ploWs.createWsConn()
	topic := "balance"
	ploWs.accountHandle = handle
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (ploWs *PloWs) GetOrderWithWs(handle func([]PloOrder)) error {
	ploWs.createWsConn()
	topic := "order"
	ploWs.orderHandle = handle
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (ploWs *PloWs) GetPositionWithWs(handle func([]PloPosition)) error {
	ploWs.createWsConn()
	topic := "position"
	ploWs.positionHandle = handle
	return ploWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (ploWs *PloWs) SetErrorHandler(handle func(error)) {
	ploWs.errorHandle = handle
}

func (ploWs *PloWs) CloseWs() {
	ploWs.ws.CloseWs()
}

type DepthManager struct {
	buyMap map[string]DepthItem
	sellMap map[string]DepthItem
}

func NewDepthManager() *DepthManager {
	return &DepthManager{
		buyMap: make(map[string]DepthItem),
		sellMap: make(map[string]DepthItem),
	}
}

func (this *DepthManager) Update(action string, items []DepthItem) (DepthRecordsDecimal, DepthRecordsDecimal) {
	if action == "partial" {
		this.buyMap = make(map[string]DepthItem)
		this.sellMap = make(map[string]DepthItem)
	}

	if action == "delete" {
		for i := range items {
			item := &items[i]
			if item.Side == "buy" {
				delete(this.buyMap, item.Price.String())
			} else {
				delete(this.sellMap, item.Price.String())
			}
		}
	} else {
		for i := range items {
			item := &items[i]
			if item.Side == "buy" {
				this.buyMap[item.Price.String()] = *item
			} else {
				this.sellMap[item.Price.String()] = *item
			}
		}
	}

	bids := make(DepthRecordsDecimal, len(this.buyMap))
	i := 0
	for _, item := range this.buyMap {
		bids[i] = DepthRecordDecimal{Price: item.Price, Amount: item.Size}
		i++
	}
	sort.SliceStable(bids, func(i,j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	asks := make(DepthRecordsDecimal, len(this.sellMap))
	i = 0
	for _, item := range this.sellMap {
		asks[i] = DepthRecordDecimal{Price: item.Price, Amount: item.Size}
		i++
	}
	sort.SliceStable(asks, func(i,j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})
	return asks, bids
}
