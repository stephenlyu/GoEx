package deerdex

import (
	"encoding/json"
	. "github.com/stephenlyu/GoEx"
	"log"
	"github.com/shopspring/decimal"
	"time"
)

func (this *DeerDex) createPublicWsConn() {
	if this.publicWs == nil {
		//connect wsx
		this.createPublicWsLock.Lock()
		defer this.createPublicWsLock.Unlock()

		if this.publicWs == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

			this.publicWs = NewWsConn("wss://wsapi.deerdex.com/openapi/quote/ws/v1")
			this.publicWs.SetErrorHandler(this.errorHandle)
			this.publicWs.ReConnect()
			this.publicWs.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//println(string(msg))

				var data struct {
					Ping int64
					Topic string
					Symbol string
					Event string
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					this.publicWs.UpdateActivedTime()
					this.publicWs.SendMessage(map[string]interface{}{"pong": data.Ping})
					return
				}

				if data.Event == "sub" {
					return
				}

				switch data.Topic {
				case "depth":
					depth := this.parseDepth(msg)
					this.wsDepthHandleMap[data.Symbol](depth)
				case "trade":
					symbol := this.getPairByName(data.Symbol)
					depth := this.parseTrade(msg)
					this.wsTradeHandleMap[data.Symbol](symbol, depth)
				}
			})
		}
	}
}

func (this *DeerDex) GetDepthWithWs(symbol string,
	depthHandle func(*DepthDecimal)) error {

	symbol = this.transSymbol(symbol)

	this.createPublicWsConn()

	this.wsDepthHandleMap[symbol] = depthHandle

	event := map[string]interface{}{
		"symbol": symbol,
		"topic": "depth",
		"event": "sub",
		"params": map[string]interface{} {
			"binary": false,
		},
	}
	return this.publicWs.Subscribe(event)
}

func (this *DeerDex) GetTradeWithWs(symbol string,
	tradesHandle func(string, []TradeDecimal)) error {
	this.createPublicWsConn()

	symbol = this.transSymbol(symbol)

	this.wsTradeHandleMap[symbol] = tradesHandle

	event := map[string]interface{}{
		"symbol": symbol,
		"topic": "trade",
		"event": "sub",
		"params": map[string]interface{} {
			"binary": false,
		},
	}
	return this.publicWs.Subscribe(event)
}

func (this *DeerDex) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Data []struct {
			Side string
			Price decimal.Decimal	`json:"p"`
			Vol decimal.Decimal		`json:"q"`
			Ts int64				`json:"t"`
			IsBuyerMaker bool 		`json:"m"`
		}
	}

	json.Unmarshal(msg, &data)

	l := data.Data

	var ret = make([]TradeDecimal, len(l))
	for i, r := range l {
		ret[i] = TradeDecimal{
			Price: r.Price,
			Amount: r.Vol,
			Date: r.Ts,
		}
		if r.IsBuyerMaker {
			ret[i].Type = "sell"
		} else {
			ret[i].Type = "buy"
		}
	}

	return ret
}

func (this *DeerDex) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts int64
		Data []struct {
		   Asks [][]decimal.Decimal		`json:"a"`
		   Buys [][]decimal.Decimal		`json:"b"`
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Data[0]

	depth := new(DepthDecimal)

	depth.UTime = time.Unix(data.Ts/1000, data.Ts%1000 * int64(time.Millisecond))
	
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

func (this *DeerDex) CloseWs() {
	this.publicWs.CloseWs()
}

func (this *DeerDex) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
