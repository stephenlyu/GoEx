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
	apiSecretKey     string
	ws               *WsConn
	createWsLock     sync.Mutex
	wsDepthHandleMap map[string]func(*Depth)
	wsTradeHandleMap map[string]func(string, []Trade)
	orderHandle      func([]FutureOrder)
	fillHandle       func([]FutureFill)
	accountHandle    func(*FutureAccount)
	positionHandle   func([]FuturePosition)
	errorHandle      func(error)
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
			bitmexWs.wsTradeHandleMap = make(map[string]func(string, []Trade))

			bitmexWs.ws = NewWsConn("wss://www.bitmex.com/realtime")
			bitmexWs.ws.SetErrorHandler(bitmexWs.errorHandle)
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
						topic := fmt.Sprintf("trade:%s", symbol)
						bitmexWs.wsTradeHandleMap[topic](symbol, trades)
					}
				case "orderBook10":
					depth := bitmexWs.parseDepth(msg)
					if depth != nil {
						topic := fmt.Sprintf("orderBook10:%s", depth.Symbol)
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
					account := bitmexWs.parseMargin(msg)
					if account != nil && bitmexWs.accountHandle != nil {
						bitmexWs.accountHandle(account)
					}
				case "position":
					positions := bitmexWs.parsePosition(msg)
					if len(positions) > 0 && bitmexWs.positionHandle != nil {
						bitmexWs.positionHandle(positions)
					}
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
	ret.Symbol = r.Symbol
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

func (bitmexWs *BitMexWs) parseMargin(msg []byte) *FutureAccount {
	log.Println("BitMexWs.parseMargin", string(msg))
	var data struct {
		Data []Margin
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	if len(data.Data) == 0 {
		return nil
	}

	return data.Data[len(data.Data) - 1].ToFutureAccount()
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

func (bitmexWs *BitMexWs) parsePosition(msg []byte) []FuturePosition {
	fmt.Println(string(msg))
	var data struct {
		Data []BitmexPosition
	}
	err := json.Unmarshal(msg, &data)
	if err != nil {
		return nil
	}

	var ret []FuturePosition
	for i := range data.Data {
		ret = append(ret, *data.Data[i].ToFuturePosition())
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

func (bitmexWs *BitMexWs) GetDepthWithWs(symbol string, handle func(*Depth)) error {
	bitmexWs.createWsConn()
	topic := fmt.Sprintf("orderBook10:%s", symbol)
	bitmexWs.wsDepthHandleMap[topic] = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) GetTradeWithWs(symbol string, handle func(string, []Trade)) error {
	bitmexWs.createWsConn()
	topic := fmt.Sprintf("trade:%s", symbol)
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

func (bitmexWs *BitMexWs) GetAccountWithWs(handle func(*FutureAccount)) error {
	bitmexWs.createWsConn()
	topic := "margin"
	bitmexWs.accountHandle = handle
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

func (bitmexWs *BitMexWs) GetPositionWithWs(handle func([]FuturePosition)) error {
	bitmexWs.createWsConn()
	topic := "position"
	bitmexWs.positionHandle = handle
	return bitmexWs.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{topic}})
}

func (bitmexWs *BitMexWs) SetErrorHandler(handle func(error)) {
	bitmexWs.errorHandle = handle
}

func (bitmexWs *BitMexWs) CloseWs() {
	bitmexWs.ws.CloseWs()
}
