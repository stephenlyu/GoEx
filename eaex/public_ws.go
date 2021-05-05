package eaex

import (
	"encoding/json"
	"log"
	"time"

	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
)

func (this *EAEX) createTradeWsConn() {
	if this.tradeWs == nil {
		//connect wsx
		this.createTradeWsLock.Lock()
		defer this.createTradeWsLock.Unlock()

		if this.tradeWs == nil {
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

			this.tradeWs = NewWsConn("ws://47.105.211.130:8081/openapi/quote/ws/v1")
			this.tradeWs.SetErrorHandler(this.errorHandle)
			this.tradeWs.ReConnect()
			this.tradeWs.ReceiveMessageEx(func(isBin bool, msg []byte) {
				// println(string(msg))

				var data struct {
					Ping   int64
					Topic  string
					Symbol string
					Event  string
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					this.tradeWs.UpdateActivedTime()
					this.tradeWs.SendMessage(map[string]interface{}{"pong": data.Ping})
					return
				}

				if data.Event == "sub" {
					return
				}

				switch data.Topic {
				case "trade":
					symbol := this.getPairByName(data.Symbol)
					trade := this.parseTrade(msg)
					this.wsTradeHandleMap[data.Symbol](symbol, trade)
				}
			})
		}
	}
}

func (this *EAEX) createDepthWsConn() {
	if this.depthWs == nil {
		//connect wsx
		this.createDepthWsLock.Lock()
		defer this.createDepthWsLock.Unlock()

		if this.depthWs == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))

			this.depthWs = NewWsConn("ws://47.105.211.130:8081/openapi/quote/ws/v1")
			this.depthWs.SetErrorHandler(this.errorHandle)
			this.depthWs.ReConnect()
			this.depthWs.ReceiveMessageEx(func(isBin bool, msg []byte) {
				// println(string(msg))

				var data struct {
					Ping   int64
					Topic  string
					Symbol string
					Event  string
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					this.depthWs.UpdateActivedTime()
					this.depthWs.SendMessage(map[string]interface{}{"pong": data.Ping})
					return
				}

				if data.Event == "sub" {
					return
				}

				switch data.Topic {
				case "depth":
					depth := this.parseDepth(msg)
					this.wsDepthHandleMap[data.Symbol](depth)
				}
			})
		}
	}
}

func (this *EAEX) GetDepthWithWs(symbol string,
	depthHandle func(*DepthDecimal)) error {

	symbol = this.transSymbol(symbol)

	this.createDepthWsConn()

	this.wsDepthHandleMap[symbol] = depthHandle

	event := map[string]interface{}{
		"symbol": symbol,
		"topic":  "depth",
		"event":  "sub",
		"params": map[string]interface{}{
			"binary": false,
		},
	}
	return this.depthWs.Subscribe(event)
}

func (this *EAEX) GetTradeWithWs(symbol string,
	tradesHandle func(string, []TradeDecimal)) error {
	this.createTradeWsConn()

	symbol = this.transSymbol(symbol)

	this.wsTradeHandleMap[symbol] = tradesHandle

	event := map[string]interface{}{
		"symbol": symbol,
		"topic":  "trade",
		"event":  "sub",
		"params": map[string]interface{}{
			"binary": false,
		},
	}
	return this.tradeWs.Subscribe(event)
}

func (this *EAEX) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Data []struct {
			Side         string
			Price        decimal.Decimal `json:"p"`
			Vol          decimal.Decimal `json:"q"`
			Ts           int64           `json:"t"`
			IsBuyerMaker bool            `json:"m"`
		}
	}

	json.Unmarshal(msg, &data)

	l := data.Data

	var ret = make([]TradeDecimal, len(l))
	for i, r := range l {
		ret[i] = TradeDecimal{
			Price:  r.Price,
			Amount: r.Vol,
			Date:   r.Ts,
		}
		if r.IsBuyerMaker {
			ret[i].Type = "sell"
		} else {
			ret[i].Type = "buy"
		}
	}

	return ret
}

func (this *EAEX) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts   int64
		Data []struct {
			Asks [][]decimal.Decimal `json:"a"`
			Buys [][]decimal.Decimal `json:"b"`
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Data[0]

	depth := new(DepthDecimal)

	depth.UTime = time.Unix(data.Ts/1000, data.Ts%1000*int64(time.Millisecond))

	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Buys), len(r.Buys))
	for i, o := range r.Buys {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth
}

func (this *EAEX) CloseWs() {
	closeTradeWs := func() {
		this.createTradeWsLock.Lock()
		defer this.createTradeWsLock.Unlock()
		if this.tradeWs != nil {
			this.tradeWs.Close()
			this.tradeWs = nil
		}
	}
	closeDepthWs := func() {
		this.createDepthWsLock.Lock()
		defer this.createDepthWsLock.Unlock()
		if this.depthWs != nil {
			this.depthWs.Close()
			this.depthWs = nil
		}
	}
	closeTradeWs()
	closeDepthWs()
}

func (this *EAEX) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
