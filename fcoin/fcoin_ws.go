package fcoin

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"time"
	"github.com/shopspring/decimal"
	"github.com/pborman/uuid"
	"strings"
)

func (this *FCoin) createWsConn() {
	if this.ws == nil {
		//connect wsx
		this.createWsLock.Lock()
		defer this.createWsLock.Unlock()

		if this.ws == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			this.wsSymbolMap = make(map[string]string)

			this.ws = NewWsConn("wss://api.fcoin.com/v2/ws")
			this.ws.SetErrorHandler(this.errorHandle)
			this.ws.Heartbeat(func() interface{} {
				ts := time.Now().UnixNano()/1000000
				args := make([]interface{}, 0)
				args = append(args, ts)
				return map[string]interface{}{
					"cmd":  "ping",
					"id":   uuid.New(),
					"args": args}
			}, 20*time.Second)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//println(string(msg))

				var data struct {
					Type string
					Ts int64
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					return
				}

				parts := strings.Split(data.Type, ".")

				switch parts[0] {
				case "ping", "hello":
					this.ws.UpdateActivedTime()
				case "depth":
					symbol := parts[2]
					depth := this.parseDepth(msg)
					pairSymbol := this.wsSymbolMap[symbol]
					depth.Pair = NewCurrencyPair2(pairSymbol)
					this.wsDepthHandleMap[data.Type](depth)
				case "trade":
					symbol := parts[1]
					trade := this.parseTrade(msg)
					pairSymbol := this.wsSymbolMap[symbol]
					this.wsTradeHandleMap[data.Type](pairSymbol, []TradeDecimal{*trade})
				}
			})
		}
	}
}

func (this *FCoin) newId() string {
	return uuid.New()
}

func (this *FCoin) transSymbol(inputSymbol string) string {
	pair := NewCurrencyPair2(inputSymbol)
	r, err := this.GetTradeSymbol(pair)
	if err != nil {
		panic(err)
	}
	return r.Name
}

func (this *FCoin) GetDepthWithWs(inputSymbol string, handle func(*DepthDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("depth.L20.%s", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsDepthHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"cmd":   "sub",
		"args": []string{channel},
	})
}

func (this *FCoin) GetTradeWithWs(inputSymbol string, handle func(string, []TradeDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("trade.%s", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsTradeHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"cmd":   "sub",
		"args": []string{channel},
	})
}

//{
//"type":"trade.ethbtc",
//"id":76000,
//"amount":1.000000000,
//"ts":1523419946174,
//"side":"sell",
//"price":4.000000000
//}
func (this *FCoin) parseTrade(msg []byte) *TradeDecimal {
	var data *struct {
	   Amount decimal.Decimal
	   Price decimal.Decimal
	   Id decimal.Decimal
	   Ts int64
	   Side string
	}

	json.Unmarshal(msg, &data)

	t := new(TradeDecimal)
	t.Amount = data.Amount
	t.Price = data.Price
	t.Type = data.Side
	t.Date = data.Ts
	t.Tid = data.Id.IntPart()

	return t
}

func (this *FCoin) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts int64
		Asks []decimal.Decimal
		Bids []decimal.Decimal
	}

	json.Unmarshal(msg, &data)

	timestamp := data.Ts

	r := data
	depth := new(DepthDecimal)

	depth.UTime = time.Unix(timestamp / 1000, timestamp % 1000)
	depth.AskList = make([]DepthRecordDecimal, len(r.Asks) / 2, len(r.Asks) / 2)
	for i := 0; i < len(r.Asks)/2; i++ {
		depth.AskList[i] = DepthRecordDecimal{Price: r.Asks[i*2], Amount: r.Asks[i*2+1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Bids)/2, len(r.Bids)/2)
	for i := 0; i < len(r.Bids)/2; i++ {
		depth.BidList[i] = DepthRecordDecimal{Price: r.Bids[i*2], Amount: r.Bids[i*2+1]}
	}

	return depth
}

func (this *FCoin) CloseWs() {
	this.ws.CloseWs()
}

func (this *FCoin) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
