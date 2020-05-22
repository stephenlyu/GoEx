package ztb

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"

	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

func (ztb *Ztb) createWsConn() {
	if ztb.ws == nil {
		//connect wsx
		ztb.createWsLock.Lock()
		defer ztb.createWsLock.Unlock()

		if ztb.ws == nil {
			ztb.wsDepthHandleMap = make(map[string]func(*goex.DepthDecimal))
			ztb.wsTradeHandleMap = make(map[string]func(string, []goex.TradeDecimal))
			ztb.wsSymbolMap = make(map[string]string)
			ztb.depthManagers = make(map[string]*depthManager)

			ztb.ws = goex.NewWsConn("wss://ws.ztb.com/ws")
			ztb.ws.SetErrorHandler(ztb.errorHandle)
			ztb.ws.ReConnect()
			ztb.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				var err error
				println(string(msg))

				var data struct {
					Method string
					Ping   int64
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					ztb.ws.WriteJSON(map[string]interface{}{"pong": data.Ping})
					ztb.ws.UpdateActivedTime()
					return
				}

				switch data.Method {
				case "depth.update":
					depth := ztb.parseDepth(msg)
					channel := fmt.Sprintf("depth.subscribe_%s", depth.Pair.ToSymbol("_"))
					ztb.wsDepthHandleMap[channel](depth)
				case "deals.update":
					symbol, trades := ztb.parseTrade(msg)
					channel := fmt.Sprintf("deals.subscribe_%s", symbol)
					ztb.wsTradeHandleMap[channel](symbol, trades)
				}
			})
		}
	}
}

// GetDepthWithWs is for subscribing market depth
func (ztb *Ztb) GetDepthWithWs(oSymbol string, handle func(*goex.DepthDecimal)) error {
	ztb.createWsConn()
	symbol := ztb.transSymbol(oSymbol)

	channel := fmt.Sprintf("depth.subscribe_%s", symbol)

	ztb.wsDepthHandleMap[channel] = handle
	ztb.depthManagers[symbol] = newDepthManager()
	return ztb.ws.Subscribe(map[string]interface{}{
		"method": "depth.subscribe",
		"params": []interface{}{
			symbol,
			50,
			"0.00000001",
		},
		"id": rand.Int31(),
	})
}

// GetTradeWithWs is for subscribing latest trades
func (ztb *Ztb) GetTradeWithWs(oSymbol string, handle func(string, []goex.TradeDecimal)) error {
	ztb.createWsConn()
	symbol := ztb.transSymbol(oSymbol)

	channel := fmt.Sprintf("deals.subscribe_%s", symbol)

	ztb.wsTradeHandleMap[channel] = handle
	return ztb.ws.Subscribe(map[string]interface{}{
		"method": "deals.subscribe",
		"params": []interface{}{symbol},
		"id":     rand.Int31(),
	})
}

func (ztb *Ztb) parseTrade(msg []byte) (string, []goex.TradeDecimal) {
	var data *struct {
		Params []interface{}
	}

	json.Unmarshal(msg, &data)
	symbol := data.Params[0].(string)

	bytes, _ := json.Marshal(data.Params[1])
	var l []struct {
		ID     int64
		Time   float64
		Type   string
		Price  decimal.Decimal
		Amount decimal.Decimal
	}
	json.Unmarshal(bytes, &l)

	ret := make([]goex.TradeDecimal, len(l))
	for i, o := range l {
		t := &ret[i]

		t.Tid = o.ID
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Type
		t.Date = int64(o.Time * 1000)
	}
	return symbol, ret
}

func (ztb *Ztb) parseDepth(msg []byte) *goex.DepthDecimal {
	var resp *struct {
		Params []interface{}
	}

	json.Unmarshal(msg, &resp)

	isFull := resp.Params[0].(bool)
	symbol := resp.Params[2].(string)

	bytes, _ := json.Marshal(resp.Params[1])
	var data struct {
		Bids [][]decimal.Decimal
		Asks [][]decimal.Decimal
	}
	json.Unmarshal(bytes, &data)

	dm := ztb.depthManagers[symbol]

	var d = new(goex.DepthDecimal)
	d.Pair = goex.NewCurrencyPair2(symbol)

	d.AskList, d.BidList = dm.update(isFull, data.Asks, data.Bids)

	return d
}

// CloseWs is for close websocket
func (ztb *Ztb) CloseWs() {
	ztb.ws.CloseWs()
}

// SetErrorHandler is for set an error handler
func (ztb *Ztb) SetErrorHandler(handle func(error)) {
	ztb.errorHandle = handle
}

type depthManager struct {
	buyMap  map[string]goex.DepthRecordDecimal
	sellMap map[string]goex.DepthRecordDecimal
}

func newDepthManager() *depthManager {
	return &depthManager{
		buyMap:  make(map[string]goex.DepthRecordDecimal),
		sellMap: make(map[string]goex.DepthRecordDecimal),
	}
}

func (mgr *depthManager) update(isFull bool, askList, bidList [][]decimal.Decimal) (goex.DepthRecordsDecimal, goex.DepthRecordsDecimal) {
	if isFull {
		mgr.buyMap = make(map[string]goex.DepthRecordDecimal)
		mgr.sellMap = make(map[string]goex.DepthRecordDecimal)
	}

	for _, o := range askList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(mgr.sellMap, key)
		} else {
			price := o[0]
			amount := o[1]
			mgr.sellMap[key] = goex.DepthRecordDecimal{Price: price, Amount: amount}
		}
	}

	for _, o := range bidList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(mgr.buyMap, key)
		} else {
			price := o[0]
			amount := o[1]
			mgr.buyMap[key] = goex.DepthRecordDecimal{Price: price, Amount: amount}
		}
	}

	bids := make(goex.DepthRecordsDecimal, len(mgr.buyMap))
	i := 0
	for _, item := range mgr.buyMap {
		bids[i] = item
		i++
	}
	sort.SliceStable(bids, func(i, j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	asks := make(goex.DepthRecordsDecimal, len(mgr.sellMap))
	i = 0
	for _, item := range mgr.sellMap {
		asks[i] = item
		i++
	}
	sort.SliceStable(asks, func(i, j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})
	return asks, bids
}
