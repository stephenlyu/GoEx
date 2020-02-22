package bitribe

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"time"
	"github.com/shopspring/decimal"
)


func (bitribe *Bitribe) createWsConn() {
	if bitribe.ws == nil {
		//connect wsx
		bitribe.createWsLock.Lock()
		defer bitribe.createWsLock.Unlock()

		if bitribe.ws == nil {
			bitribe.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			bitribe.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			bitribe.wsSymbolMap = make(map[string]string)

			bitribe.ws = NewWsConn("wss://wsapi.bitribe.com/openapi/quote/ws/v1")
			bitribe.ws.Heartbeat(func() interface{} {
				return map[string]interface{} {"ping": time.Now().UnixNano()/1e6}
			}, 20*time.Second)
			bitribe.ws.SetErrorHandler(bitribe.errorHandle)
			bitribe.ws.ReConnect()
			bitribe.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {

				var data struct {
					Symbol string
					Topic string
					Pong int64
					Data []interface{}
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Pong > 0 {
					bitribe.ws.UpdateActivedTime()
					return
				}

				if len(data.Data) == 0 {
					return
				}

				switch data.Topic  {
				case "trade":
				trades := bitribe.parseTrade(msg)
				if len(trades) > 0 {
					symbol := bitribe.wsSymbolMap[data.Symbol]
					topic := fmt.Sprintf("%s:%s", data.Topic, symbol)
					bitribe.wsTradeHandleMap[topic](symbol, trades)
				}
				case "depth":
					depth := bitribe.parseDepth(msg)
					if depth != nil {
						symbol := bitribe.wsSymbolMap[data.Symbol]
						topic := fmt.Sprintf("%s:%s", data.Topic, symbol)
						depth.InstrumentId = symbol
						bitribe.wsDepthHandleMap[topic](depth)
					}
				}
			})
		}
	}
}

func (bitribe *Bitribe) GetDepthWithWs(oSymbol string, handle func(*DepthDecimal)) error {
	bitribe.createWsConn()
	symbol := bitribe.transSymbol(oSymbol)
	bitribe.wsSymbolMap[symbol] = oSymbol

	topic := fmt.Sprintf("depth:%s", oSymbol)

	bitribe.wsDepthHandleMap[topic] = handle
	return bitribe.ws.Subscribe(map[string]interface{}{
		"symbol": symbol,
		"topic": "depth",
		"event": "sub",
		"params": map[string]interface{} {
			"binary": false,
		},
	})
}

func (bitribe *Bitribe) GetTradeWithWs(oSymbol string, handle func(string, []TradeDecimal)) error {
	bitribe.createWsConn()
	symbol := bitribe.transSymbol(oSymbol)
	bitribe.wsSymbolMap[symbol] = oSymbol

	topic := fmt.Sprintf("trade:%s", oSymbol)

	bitribe.wsTradeHandleMap[topic] = handle
	return bitribe.ws.Subscribe(map[string]interface{}{
		"symbol": symbol,
		"topic": "trade",
		"event": "sub",
		"params": map[string]interface{} {
			"binary": false,
		},
	})
}

func (bitribe *Bitribe) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Data   []struct {
			Timestamp int64				`json:"t"`
			Price decimal.Decimal		`json:"p"`
			IsSellerMaker bool 			`json:"m"`
			Qty decimal.Decimal			`json:"q"`
		}
	}

	json.Unmarshal(msg, &data)

	ret := make([]TradeDecimal, len(data.Data))
	for i := range data.Data {
		o := &data.Data[i]
		var side string
		if o.IsSellerMaker {
			side = "buy"
		} else {
			side = "sell"
		}

		ret[i] = TradeDecimal {
			Tid: o.Timestamp,
			Type: side,
			Amount: o.Qty,
			Price: o.Price,
			Date: o.Timestamp,
		}
	}

	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}
	return ret
}

func (bitribe *Bitribe) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Data []struct {
			Asks [][]decimal.Decimal	`json:"a"`
			Bids [][]decimal.Decimal	`json:"b"`
		}
	}

	json.Unmarshal(msg, &data)

	var d = new(DepthDecimal)

	for _, o := range data.Data[0].Asks {
		d.AskList = append(d.AskList, DepthRecordDecimal{Price:o[0], Amount:o[1]})
	}

	for _, o := range data.Data[0].Bids {
		d.BidList = append(d.BidList, DepthRecordDecimal{Price:o[0], Amount:o[1]})
	}

	return d
}

func (bitribe *Bitribe) CloseWs() {
	bitribe.ws.CloseWs()
}

func (this *Bitribe) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
