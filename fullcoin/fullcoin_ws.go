package fullcoin

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"strings"
	"time"
	"io/ioutil"
	"bytes"
	"github.com/shopspring/decimal"
	"regexp"
	"github.com/pborman/uuid"
	"compress/gzip"
)

var (
	_DEPTH_CH_PATTERN, _ = regexp.Compile("market_([a-z0-9]+)_depth_step0")
	_TRADE_CH_PATTERN, _ = regexp.Compile("market_([a-z0-9]+)_trade_ticker")
)

func GzipDecodeV3(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (this *FullCoin) createWsConn() {
	if this.ws == nil {
		//connect wsx
		this.createWsLock.Lock()
		defer this.createWsLock.Unlock()

		if this.ws == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			this.wsSymbolMap = make(map[string]string)

			this.ws = NewWsConn("wss://ws.fullcoin.com/kline-api/ws")
			this.ws.SetErrorHandler(this.errorHandle)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, _ = GzipDecodeV3(msg)
				//println(string(msg))

				var data struct {
					Ping    int64
					Channel string
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					this.ws.UpdateActivedTime()
					this.ws.SendMessage(map[string]interface{}{"pong": data.Ping})
					return
				}

				switch {
				case _DEPTH_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := this.parseDepth(msg)
					pairSymbol := this.wsSymbolMap[symbol]
					depth.Pair = NewCurrencyPair2(pairSymbol)
					this.wsDepthHandleMap[data.Channel](depth)
				case _TRADE_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := this.parseTrade(msg)
					pairSymbol := this.wsSymbolMap[symbol]
					this.wsTradeHandleMap[data.Channel](pairSymbol, depth)
				}
			})
		}
	}
}

func (this *FullCoin) newId() string {
	return uuid.New()
}

func (this *FullCoin) GetDepthWithWs(inputSymbol string, handle func(*DepthDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("market_%s_depth_step0", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsDepthHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params":   map[string]interface{} {
			"channel": channel,
			"cb_id": this.newId(),
			"asks": 150,
			"bids": 150,
		},
	})
}

func (this *FullCoin) GetTradeWithWs(inputSymbol string, handle func(string, []TradeDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("market_%s_trade_ticker", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsTradeHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params":   map[string]interface{} {
			"channel": channel,
			"cb_id": this.newId(),
		},
	})
}

func (this *FullCoin) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Ts     int64
		Tick struct {
		   Data [] struct {
			   Id decimal.Decimal
			   Side string
			   Price decimal.Decimal
			   Amount decimal.Decimal
			   Vol decimal.Decimal
			   Ts int64
		   }
	   }
	}

	json.Unmarshal(msg, &data)

	var trades = make([]TradeDecimal, len(data.Tick.Data))

	for i, o := range data.Tick.Data {
		t := &trades[i]
		t.Tid = o.Id.IntPart()
		t.Amount = o.Vol
		t.Price = o.Price
		t.Type = strings.ToLower(o.Side)
		t.Date = o.Ts
	}

	return trades
}

func (this *FullCoin) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts int64
		Tick struct {
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal		`json:"buys"`
		}
	}

	json.Unmarshal(msg, &data)

	timestamp := data.Ts

	r := &data.Tick
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

	return depth
}

func (this *FullCoin) CloseWs() {
	this.ws.CloseWs()
}

func (this *FullCoin) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
