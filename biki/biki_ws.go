package biki

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

var (
	_depthChPattern, _ = regexp.Compile("market_([a-zA-Z0-9_]+)_depth_step0")
	_tradeChPattern, _ = regexp.Compile("market_([a-zA-Z0-9_]+)_trade_ticker")
)

func gzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (biki *Biki) createWsConn() {
	if biki.ws == nil {
		//connect wsx
		biki.createWsLock.Lock()
		defer biki.createWsLock.Unlock()

		if biki.ws == nil {
			biki.wsDepthHandleMap = make(map[string]func(*goex.DepthDecimal))
			biki.wsTradeHandleMap = make(map[string]func(string, []goex.TradeDecimal))
			biki.wsSymbolMap = make(map[string]string)

			biki.ws = goex.NewWsConn("wss://ws.biki.com/kline-api/ws")
			biki.ws.SetErrorHandler(biki.errorHandle)
			biki.ws.ReConnect()
			biki.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				var err error
				//println(string(msg))
				msg, err = gzipDecode(msg)
				if err != nil {
					fmt.Println(err)
					return
				}
				// println(string(msg))

				var data struct {
					Ping    int64
					Channel string
					Tick    interface{}
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					biki.ws.WriteJSON(map[string]interface{}{"pong": data.Ping})
					biki.ws.UpdateActivedTime()
					return
				}

				if data.Tick == nil {
					return
				}

				switch {
				case _depthChPattern.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := biki.parseDepth(msg)
					biki.wsDepthHandleMap[symbol](depth)
				case _tradeChPattern.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					depth := biki.parseTrade(msg)
					biki.wsTradeHandleMap[symbol](symbol, depth)
				}
			})
		}
	}
}

// GetDepthWithWs Subscribe depth
func (biki *Biki) GetDepthWithWs(oSymbol string, handle func(*goex.DepthDecimal)) error {
	biki.createWsConn()
	symbol := biki.transSymbol(oSymbol)

	channel := fmt.Sprintf("market_%s_depth_step0", symbol)

	biki.wsDepthHandleMap[symbol] = handle
	return biki.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   uuid.New(),
			"asks":    150,
			"bids":    150,
		},
	})
}

// GetTradeWithWs Subscribe trades
func (biki *Biki) GetTradeWithWs(oSymbol string, handle func(string, []goex.TradeDecimal)) error {
	biki.createWsConn()
	symbol := biki.transSymbol(oSymbol)

	channel := fmt.Sprintf("market_%s_trade_ticker", symbol)

	biki.wsTradeHandleMap[symbol] = handle
	return biki.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   uuid.New(),
		},
	})
}

func (biki *Biki) parseTrade(msg []byte) []goex.TradeDecimal {
	var data *struct {
		Tick struct {
			Data []struct {
				ID     int64
				Side   string
				Price  decimal.Decimal
				Vol    decimal.Decimal
				Amount decimal.Decimal
				Ts     int64
			}
		}
	}

	json.Unmarshal(msg, &data)

	ret := make([]goex.TradeDecimal, len(data.Tick.Data))
	for i, o := range data.Tick.Data {
		t := &ret[i]

		t.Tid = o.ID
		t.Amount = o.Vol
		t.Price = o.Price
		t.Type = strings.ToLower(o.Side)
	}
	return ret
}

func (biki *Biki) parseDepth(msg []byte) *goex.DepthDecimal {
	var data *struct {
		Ts   int64
		Tick struct {
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal `json:"buys"`
		}
	}

	json.Unmarshal(msg, &data)

	var d = new(goex.DepthDecimal)

	d.UTime = time.Unix(data.Ts/1000, data.Ts%1000)

	r := &data.Tick

	d.AskList = make([]goex.DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		d.AskList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	d.BidList = make([]goex.DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		d.BidList[i] = goex.DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}
	return d
}

// CloseWs Close websocket
func (biki *Biki) CloseWs() {
	biki.ws.CloseWs()
}

// SetErrorHandler Set error handler
func (biki *Biki) SetErrorHandler(handle func(error)) {
	biki.errorHandle = handle
}
