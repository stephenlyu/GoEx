package cointiger

import (
	"encoding/json"
	. "github.com/stephenlyu/GoEx"
	"log"
	"github.com/shopspring/decimal"
	"fmt"
	"strings"
	"github.com/pborman/uuid"
	"regexp"
	"time"
	"io/ioutil"
	"compress/gzip"
	"bytes"
)
var (
	_DEPTH_CH_PATTERN, _ = regexp.Compile("market_([a-zA-Z0-9]+)_depth_step0")
	_TRADE_CH_PATTERN, _ = regexp.Compile("market_([a-zA-Z0-9]+)_trade_ticker")
)

func GzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (this *CoinTiger) createPublicWsConn() {
	if this.publicWs == nil {
		//connect wsx
		this.createPublicWsLock.Lock()
		defer this.createPublicWsLock.Unlock()

		if this.publicWs == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

			this.publicWs = NewWsConn("wss://" + Host + "/exchange-market/ws")
			this.publicWs.SetErrorHandler(this.errorHandle)
			this.publicWs.ReConnect()
			this.publicWs.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, err := GzipDecode(msg)
				if err != nil {
					fmt.Println(err)
					return
				}
				//println(string(msg))

				var data struct {
					Ping int64
					EventRep string `json:"event_rep"`
					Channel string
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					this.publicWs.UpdateActivedTime()
					this.publicWs.SendMessage(map[string]interface{}{"pong": data.Ping})
					return
				}

				if data.EventRep == "subed" {
					return
				}

				switch {
				case _DEPTH_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := this.parseDepth(msg)
					this.wsDepthHandleMap[symbol](depth)
				case _TRADE_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := this.parseTrade(msg)
					this.wsTradeHandleMap[symbol](symbol, depth)
				}
			})
		}
	}
}

func (this *CoinTiger) GetDepthWithWs(symbol string,
	depthHandle func(*DepthDecimal)) error {

	symbol = this.transSymbol(symbol)

	this.createPublicWsConn()

	this.wsDepthHandleMap[symbol] = depthHandle
	channel := fmt.Sprintf("market_%s_depth_step0", symbol)

	event := map[string]interface{}{
		"event":   "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id": uuid.New(),
			"asks": 150,
			"bids": 150,
		},
	}
	return this.publicWs.Subscribe(event)
}

func (this *CoinTiger) GetTradeWithWs(symbol string,
	tradesHandle func(string, []TradeDecimal)) error {
	this.createPublicWsConn()

	symbol = this.transSymbol(symbol)

	this.wsTradeHandleMap[symbol] = tradesHandle
	channel := fmt.Sprintf("market_%s_trade_ticker", symbol)

	event := map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id": uuid.New(),
		},
	}
	return this.publicWs.Subscribe(event)
}

func (this *CoinTiger) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Tick struct {
			Data []struct {
				Id int64
				Side string
				Price decimal.Decimal
			 	Vol decimal.Decimal
			 	Ts int64
			}
		}
	}

	json.Unmarshal(msg, &data)

	l := data.Tick.Data

	var ret = make([]TradeDecimal, len(l))
	for i, r := range l {
		ret[i] = TradeDecimal{
			Tid: r.Id,
			Type: r.Side,
			Price: r.Price,
			Amount: r.Vol,
			Date: r.Ts,
		}
	}

	return ret
}

func (this *CoinTiger) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Ts int64
		Tick struct {
		   Asks [][]decimal.Decimal
		   Buys [][]decimal.Decimal
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Tick

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

func (this *CoinTiger) CloseWs() {
	this.publicWs.CloseWs()
}

func (this *CoinTiger) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
