package appex

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
	_DEPTH_CH_PATTERN, _ = regexp.Compile("market\\.([a-z0-9]+)\\.depth\\.step0")
	_TRADE_CH_PATTERN, _ = regexp.Compile("market\\.([a-z0-9]+)\\.trade\\.detail")
)

func GzipDecodeV3(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (this *Appex) createWsConn() {
	if this.ws == nil {
		//connect wsx
		this.createWsLock.Lock()
		defer this.createWsLock.Unlock()

		if this.ws == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			this.wsSymbolMap = make(map[string]string)

			this.ws = NewWsConn("wss://www.appex.pro/api/ws/v3")
			this.ws.SetErrorHandler(this.errorHandle)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, _ = GzipDecodeV3(msg)

				var data struct {
					Ping int64
					Ch string
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
				case _DEPTH_CH_PATTERN.Match([]byte(data.Ch)):
					symbol := strings.Split(data.Ch, ".")[1]
					depth := this.parseDepth(msg)
					pairSymbol := this.getPairByName(symbol)
					depth.Pair = NewCurrencyPair2(pairSymbol)
					this.wsDepthHandleMap[data.Ch](depth)
				case _TRADE_CH_PATTERN.Match([]byte(data.Ch)):
					symbol := strings.Split(data.Ch, ".")[1]
					depth := this.parseTrade(msg)
					pairSymbol := this.getPairByName(symbol)
					this.wsTradeHandleMap[data.Ch](pairSymbol, depth)
				}
			})
		}
	}
}

func (this *Appex) newId() string {
	return uuid.New()
}

func (this *Appex) GetDepthWithWs(inputSymbol string, handle func(*DepthDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("market.%s.depth.step0", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsDepthHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"sub":   channel,
		"id": this.newId(),
	})
}

func (this *Appex) GetTradeWithWs(inputSymbol string, handle func(string, []TradeDecimal)) error {
	this.createWsConn()

	symbol := this.transSymbol(inputSymbol)
	channel := fmt.Sprintf("market.%s.trade.detail", symbol)

	this.wsSymbolMap[symbol] = inputSymbol
	this.wsTradeHandleMap[channel] = handle
	return this.ws.Subscribe(map[string]interface{}{
		"sub":   channel,
		"id": this.newId()})
}

func (this *Appex) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Status string
		Ch     string
		Ts     int64
		Tick struct {
		   Data [] struct {
			   Amount decimal.Decimal
			   Price decimal.Decimal
			   Id decimal.Decimal
			   Ts int64
			   Direction string
		   }
		   Ts int64
	   }
	}

	json.Unmarshal(msg, &data)

	var trades = make([]TradeDecimal, len(data.Tick.Data))

	for i, o := range data.Tick.Data {
		t := &trades[i]
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Direction
		t.Date = o.Ts
	}

	return trades
}

func (this *Appex) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts int64
		Tick struct {
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal
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

func (this *Appex) CloseWs() {
	this.ws.CloseWs()
}

func (this *Appex) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
