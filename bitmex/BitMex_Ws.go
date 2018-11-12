package bitmex

import (
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"time"
	"sync"
	"encoding/json"
	"log"
	"strings"
)

type BitMexWs struct {
	apiKey,
	apiSecretKey string
	ws                *WsConn
	createWsLock      sync.Mutex
	wsDepthHandleMap  map[string]func(*Depth)
	wsTradeHandleMap map[string]func(CurrencyPair, []Trade)
	orderHandle func([]FutureOrder)
	fillHandle func([]FutureFill)
	marginHandle func([]Margin)
}

func NewBitMexWs(apiKey, apiSecretyKey string) *BitMexWs {
	return &BitMexWs{apiKey: apiKey, apiSecretKey: apiSecretyKey}
}

func (bitmexWs *BitMexWs) createWsConn() {
	if bitmexWs.ws == nil {
		//connect wsx
		bitmexWs.createWsLock.Lock()
		defer bitmexWs.createWsLock.Unlock()

		if bitmexWs.ws == nil {
			bitmexWs.wsDepthHandleMap = make(map[string]func(*Depth))
			bitmexWs.wsTradeHandleMap = make(map[string]func(CurrencyPair, []Trade))

			bitmexWs.ws = NewWsConn("wss://www.bitmex.com/realtime")
			bitmexWs.ws.Heartbeat(func() interface{} { return "ping"}, 5*time.Second)
			bitmexWs.ws.ReConnect()
			bitmexWs.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//fmt.Println(string(msg))
				if string(msg) == "pong" {
					bitmexWs.ws.UpdateActivedTime()
					return
				}
				var resp struct {
					Success bool
					Subscribe string
					Table string
					Action string
				}
				err := json.Unmarshal(msg, &resp)
				if err != nil {
					log.Print(err)
					return
				}

				if resp.Subscribe != "" {
					return
				}

				if resp.Action == "partial" {
					return
				}

				switch resp.Table {
				case "trade":
					symbol, trades := bitmexWs.parseTrade(msg)
					if symbol != "" {
						pair := ParseSymbol(symbol)
						topic := fmt.Sprintf("trade:%s", symbol)
						bitmexWs.wsTradeHandleMap[topic](pair, trades)
					}
				case "orderBook10":
					depth := bitmexWs.parseDepth(msg)
					if depth != nil {
						topic := fmt.Sprintf("orderBook10:%s", depth.Pair.ToSymbol(""))
						bitmexWs.wsDepthHandleMap[topic](depth)
					}
				case "order":
					orders := bitmexWs.parseOrder(msg)
					if len(orders) > 0 && bitmexWs.orderHandle != nil {
						bitmexWs.orderHandle(orders)
					}
				case "execution":
					fills := bitmexWs.parseExecution(msg)
					if len(fills) > 0 && bitmexWs.fillHandle != nil {
						bitmexWs.fillHandle(fills)
					}
				case "margin":
					margins := bitmexWs.parseMargin(msg)
					if len(margins) > 0 && bitmexWs.marginHandle != nil {
						bitmexWs.marginHandle(margins)
					}
				case "position":
				}
			})
		}
	}
}

func (bitmexWs *BitMexWs) parseTrade(msg []byte) (string, []Trade) {
	var data struct {
		Data []struct {
			 Timestamp string
			 Symbol string
			 Side string
			 Size float64
			 Price float64
			 }
	}
	json.Unmarshal(msg, &data)

	if len(data.Data) == 0 {
		return "", nil
	}

	ret := make([]Trade, len(data.Data))
	for i, r := range data.Data {
		_, ts := ParseTimestamp(r.Timestamp)
		ret[i] = Trade{
			Tid: ts,
			Type: strings.ToLower(r.Side),
			Amount: r.Size,
			Price: r.Price,
			Date: ts,
		}
	}

	return data.Data[0].Symbol, ret
}

func (bitmexWs *BitMexWs) parseDepth(msg []byte) *Depth {
	var data struct {
		Data []struct {
			Timestamp string
			Symbol string
			Bids [][2]float64
			Asks [][2]float64
		}
	}
	json.Unmarshal(msg, &data)

	if len(data.Data) == 0 {
		return nil
	}

	r := data.Data[0]
	ret := &Depth{}
	ret.UTime, _ = time.Parse(UTC_FORMAT, r.Timestamp)
	ret.Pair = ParseSymbol(r.Symbol)
	ret.AskList = make(DepthRecords, len(r.Asks))
	ret.BidList = make(DepthRecords, len(r.Bids))

	for i, o := range r.Asks {
		ret.AskList[i] = DepthRecord{Price: o[0], Amount: o[1]}
	}

	for i, o := range r.Bids {
		ret.BidList[i] = DepthRecord{Price: o[0], Amount: o[1]}
	}

	return ret
}

func (bitmexWs *BitMexWs) parseMargin(msg []byte) []Margin {
	var data struct {
		Data []Margin
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	return data.Data
}

func (bitmexWs *BitMexWs) parseOrder(msg []byte) []FutureOrder {
	var data struct {
		Data []BitmexOrder
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	var ret []FutureOrder
	for i := range data.Data {
		if data.Data[i].Status == "" {
			continue
		}
		ret = append(ret, *data.Data[i].ToFutureOrder())
	}

	return ret
}

func (bitmexWs *BitMexWs) parseExecution(msg []byte) []FutureFill {
	var data struct {
		Data []Execution
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	var ret []FutureFill
	for i := range data.Data {
		if data.Data[i].TrdMatchId == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		ret = append(ret, *data.Data[i].ToFill())
	}

	return ret
}

func (bitmexWs *BitMexWs) GetDepthWithWs(pair CurrencyPair, handle func(*Depth)) error {
	bitmexWs.createWsConn()
	topic := fmt.Sprintf("orderBook10:%s", pair.ToSymbol(""))
	bitmexWs.wsDepthHandleMap[topic] = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) GetTradeWithWs(pair CurrencyPair, handle func(CurrencyPair, []Trade)) error {
	bitmexWs.createWsConn()
	topic := fmt.Sprintf("trade:%s", pair.ToSymbol(""))
	bitmexWs.wsTradeHandleMap[topic] = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) Authenticate() error {
	bitmexWs.createWsConn()
	expires := time.Now().Unix() + 30
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "authKeyExpires",
		"args": []interface{}{bitmexWs.apiKey, expires, BuildWsSignature(bitmexWs.apiSecretKey, "/realtime", expires)}})
}

func (bitmexWs *BitMexWs) GetMarginWithWs(handle func([]Margin)) error {
	bitmexWs.createWsConn()
	topic := "margin"
	bitmexWs.marginHandle = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) GetOrderWithWs(handle func([]FutureOrder)) error {
	bitmexWs.createWsConn()
	topic := "order"
	bitmexWs.orderHandle = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) GetFillWithWs(handle func([]FutureFill)) error {
	bitmexWs.createWsConn()
	topic := "execution"
	bitmexWs.fillHandle = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) CloseWs() {
	bitmexWs.ws.CloseWs()
}
