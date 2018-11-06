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

const UTC_FORMAT = "2006-01-02T15:04:05.999Z"

type BitMexWs struct {
	apiKey,
	apiSecretKey string
	ws                *WsConn
	createWsLock      sync.Mutex
	wsDepthHandleMap  map[string]func(*Depth)
	wsTradeHandleMap map[string]func(CurrencyPair, []Trade)
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
				case "execution":
				case "margin":
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

func (bitmexWs *BitMexWs) CloseWs() {
	bitmexWs.ws.CloseWs()
}

func ParseTimestamp(ts string) (error, int64) {
	t, err := time.Parse(UTC_FORMAT, ts)
	if err != nil {
		return err, 0
	}
	return nil, t.UnixNano() / int64(time.Millisecond)
}

func FormatTimestamp(ts int64) string {
	t := time.Unix(ts / 1000, ts % 1000 * int64(time.Millisecond)).In(time.UTC)
	return t.Format(UTC_FORMAT)
}

func ParseSymbol(symbol string) CurrencyPair {
	if symbol != "XBTUSD" {
		log.Fatalf("symbol %s not supported", symbol)
	}

	return CurrencyPair{XBT, USD}
}
