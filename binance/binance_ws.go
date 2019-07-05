package binance

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	"github.com/pborman/uuid"
	"github.com/z-ray/log"
	"github.com/gorilla/websocket"
)


func (this *Binance) createDataWsConn(symbols []string) {
	this.wsLock.Lock()
	defer this.wsLock.Unlock()
	if this.wsData != nil {
		return
	}
	this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
	this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

	var streams []string
	var symbolMap = make(map[string]string)
	for _, symbol := range symbols {
		streamSymbol := strings.ToLower(this.transSymbol(symbol))
		symbolMap[streamSymbol] = symbol
		streams = append(streams, streamSymbol + "@depth")
		streams = append(streams, streamSymbol + "@trade")
	}

	url := fmt.Sprintf("wss://stream.binance.com:9443/stream?streams=%s", strings.Join(streams, "/"))
	ws := NewWsConn(url)
	ws.SetErrorHandler(this.errorHandle)
	ws.HeartbeatEx(func() (int, string) {return websocket.PongMessage, "pong"}, 20*time.Second)
	ws.ReConnect()
	ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
		//println(string(msg))

		// 只要收到消息，就说明连接还是活的
		ws.UpdateActivedTime()
		var data struct {
			Stream string
		}
		err := json.Unmarshal(msg, &data)
		if err != nil {
			log.Print(err)
			return
		}

		switch {
		case strings.HasSuffix(data.Stream, "@depth"):
			symbol, depth := this.parseDepth(msg)
			pairSymbol := symbolMap[strings.ToLower(symbol)]
			depth.Pair = NewCurrencyPair2(pairSymbol)
			this.wsDepthHandleMap[pairSymbol](depth)
		case strings.HasSuffix(data.Stream, "@trade"):
			symbol, trades := this.parseTrade(msg)
			pairSymbol := symbolMap[strings.ToLower(symbol)]
			this.wsTradeHandleMap[pairSymbol](pairSymbol, trades)
		}
	})
	this.wsData = ws
}

func (this *Binance) newId() string {
	return uuid.New()
}

func (this *Binance) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Binance) GetDepthTradeWithWs(symbols []string, depthCB func(*DepthDecimal), tradeCB func(string, []TradeDecimal)) error {
	this.createDataWsConn(symbols)
	for _, symbol := range symbols {
		this.wsDepthHandleMap[symbol] = depthCB
		this.wsTradeHandleMap[symbol] = tradeCB
	}
	return nil
}

func (this *Binance) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data *struct {
		Data map[string]interface{}
	}
	json.Unmarshal(msg, &data)
	r := data.Data

	var side string
	if r["m"].(bool) {
		side = "sell"
	} else {
		side = "buy"
	}

	symbol := r["s"].(string)
	tid := r["t"].(float64)
	amount, _ := decimal.NewFromString(r["q"].(string))
	price, _ := decimal.NewFromString(r["p"].(string))
	timestamp := r["T"].(float64)

	return symbol, []TradeDecimal {
		{
			Tid: int64(tid),
			Type: side,
			Amount: amount,
			Price: price,
			Date: int64(timestamp),
		},
	}
}

func (this *Binance) parseDepth(msg []byte) (string, *DepthDecimal) {
	var data *struct {
		Data struct {
				 Event string 					`json:"e"`
				 EventTs int64 					`json:"E"`
				 Symbol string 					`json:"s"`
				 Bids [][]decimal.Decimal		`json:"b"`
				 Asks [][]decimal.Decimal		`json:"a"`
			 }
	}

	json.Unmarshal(msg, &data)

	r := &data.Data
	timestamp := r.EventTs

	depth := new(DepthDecimal)

	depth.UTime = time.Unix(timestamp / 1000, timestamp % 1000)
	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return r.Symbol, depth
}

func (this *Binance) CloseWs() {
	this.wsData.Close()
}

func (this *Binance) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
