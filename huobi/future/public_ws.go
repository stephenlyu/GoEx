package huobifuture

import (
	"encoding/json"
	. "github.com/stephenlyu/GoEx"
	"log"
	"github.com/shopspring/decimal"
	"fmt"
	"strings"
	"github.com/pborman/uuid"
	"io/ioutil"
	"bytes"
	"compress/gzip"
	"regexp"
)
var (
	_DEPTH_CH_PATTERN, _ = regexp.Compile("market\\.([a-zA-Z0-9_]+)\\.depth\\.step0")
	_TRADE_CH_PATTERN, _ = regexp.Compile("market\\.([a-zA-Z0-9_]+)\\.trade\\.detail")
)

func GzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (this *HuobiFuture) createPublicWsConn() {
	if this.publicWs == nil {
		//connect wsx
		this.createPublicWsLock.Lock()
		defer this.createPublicWsLock.Unlock()

		if this.publicWs == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

			this.publicWs = NewWsConn("wss://www.hbdm.com/ws")
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
					Ch string
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

				switch {
				case _DEPTH_CH_PATTERN.Match([]byte(data.Ch)):
					symbol := strings.Split(data.Ch, ".")[1]
					depth := this.parseDepth(msg)
					this.wsDepthHandleMap[symbol](depth)
				case _TRADE_CH_PATTERN.Match([]byte(data.Ch)):
					symbol := strings.Split(data.Ch, ".")[1]
					depth := this.parseTrade(msg)
					this.wsTradeHandleMap[symbol](symbol, depth)
				}
			})
		}
	}
}

func (this *HuobiFuture) GetDepthWithWs(symbol string,
	depthHandle func(*DepthDecimal)) error {

	this.createPublicWsConn()

	this.wsDepthHandleMap[symbol] = depthHandle
	channel := fmt.Sprintf("market.%s.depth.step0", symbol)

	event := map[string]interface{}{
		"sub":   channel,
		"id": uuid.New(),
	}
	return this.publicWs.Subscribe(event)
}


func (this *HuobiFuture) GetTradeWithWs(symbol string,
	tradesHandle func(string, []TradeDecimal)) error {
	this.createPublicWsConn()

	this.wsTradeHandleMap[symbol] = tradesHandle
	channel := fmt.Sprintf("market.%s.trade.detail", symbol)

	event := map[string]interface{}{
		"sub":   channel,
		"id": uuid.New(),
	}
	return this.publicWs.Subscribe(event)
}

func (this *HuobiFuture) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Tick struct {
				 Data []struct {
					 Amount decimal.Decimal
					 Price decimal.Decimal
					 Ts int64
					 Direction string
					 Id int64
				 }
			 }
	}

	json.Unmarshal(msg, &data)

	l := data.Tick.Data

	var ret = make([]TradeDecimal, len(l))
	for i, r := range l {
		ret[i] = TradeDecimal{
			Tid: r.Id,
			Type: r.Direction,
			Price: r.Price,
			Amount: r.Amount,
			Date: r.Ts,
		}
	}

	return ret
}

func (this *HuobiFuture) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Tick struct {
				 Asks [][]decimal.Decimal
				 Bids [][]decimal.Decimal
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Tick

	depth := new(DepthDecimal)
	
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

func (this *HuobiFuture) CloseWs() {
	this.publicWs.CloseWs()
}

func (this *HuobiFuture) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
