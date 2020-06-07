package bibull

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/pborman/uuid"
	"github.com/shopspring/decimal"
	goex "github.com/stephenlyu/GoEx"
)

var (
	_DepthChPattern, _ = regexp.Compile("market_([a-zA-Z0-9_]+)_depth.step0")
	_TradeChPattern, _ = regexp.Compile("market_([a-zA-Z0-9]+)_trade_ticker")
)

func gzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (bibull *BiBull) createWsConn() {
	if bibull.ws == nil {
		//connect wsx
		bibull.createWsLock.Lock()
		defer bibull.createWsLock.Unlock()

		if bibull.ws == nil {
			bibull.wsDepthHandleMap = make(map[string]func(*goex.DepthDecimal))
			bibull.wsTradeHandleMap = make(map[string]func(string, []goex.TradeDecimal))
			bibull.wsSymbolMap = make(map[string]string)

			bibull.ws = goex.NewWsConn("wss://ws.bibull.co/kline-api/ws")
			//bibull.ws.Heartbeat(func() interface{} {
			//	return map[string]interface{} {"ping": time.Now().UnixNano()/1e6}
			//}, 20*time.Second)
			bibull.ws.SetErrorHandler(bibull.errorHandle)
			bibull.ws.ReConnect()
			bibull.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, err := gzipDecode(msg)
				if err != nil {
					fmt.Println(err)
					return
				}
				//println(string(msg))

				var data struct {
					EventRep string `json:"event_rep"`
					Channel  string
					Ts       int64
					Ping     int64
					Tick     interface{}
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					bibull.ws.WriteJSON(map[string]interface{}{"pong": data.Ping})
					bibull.ws.UpdateActivedTime()
					return
				}

				if data.EventRep != "" {
					return
				}

				if data.Tick == nil {
					return
				}

				switch {
				case _DepthChPattern.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					symbol = bibull.wsSymbolMap[symbol]
					depth := bibull.parseDepth(msg)
					bibull.wsDepthHandleMap[data.Channel](depth)
				case _TradeChPattern.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					symbol = bibull.wsSymbolMap[symbol]
					depth := bibull.parseTrade(msg)
					bibull.wsTradeHandleMap[data.Channel](symbol, depth)
				}
			})
		}
	}
}

// GetDepthWithWs Subscribe depth of symbol
func (bibull *BiBull) GetDepthWithWs(oSymbol string, handle func(*goex.DepthDecimal)) error {
	bibull.createWsConn()
	symbol := bibull.transSymbol(oSymbol)
	bibull.wsSymbolMap[symbol] = oSymbol

	channel := fmt.Sprintf("market_%s_depth_step0", symbol)

	bibull.wsDepthHandleMap[channel] = handle
	return bibull.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   uuid.New(),
			"asks":    150,
			"bids":    150,
		},
	})
}

// GetTradeWithWs Subscribe trade of symbol
func (bibull *BiBull) GetTradeWithWs(oSymbol string, handle func(string, []goex.TradeDecimal)) error {
	bibull.createWsConn()
	symbol := bibull.transSymbol(oSymbol)
	bibull.wsSymbolMap[symbol] = oSymbol

	channel := fmt.Sprintf("market_%s_trade_ticker", symbol)

	bibull.wsTradeHandleMap[channel] = handle
	return bibull.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   uuid.New(),
		},
	})
}

func (bibull *BiBull) parseTrade(msg []byte) []goex.TradeDecimal {
	var data *struct {
		Tick struct {
			ID   int64
			Ts   int64
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
	for i := range data.Tick.Data {
		o := &data.Tick.Data[i]
		ret[i] = goex.TradeDecimal{
			Tid:    o.ID,
			Type:   strings.ToLower(o.Side),
			Amount: o.Vol,
			Price:  o.Price,
			Date:   o.Ts,
		}
	}
	return ret
}

func (bibull *BiBull) parseDepth(msg []byte) *goex.DepthDecimal {
	var data *struct {
		Tick struct {
			Asks [][]decimal.Decimal `json:"asks"`
			Bids [][]decimal.Decimal `json:"buys"`
		}
	}

	json.Unmarshal(msg, &data)

	var d = new(goex.DepthDecimal)

	for _, o := range data.Tick.Asks {
		d.AskList = append(d.AskList, goex.DepthRecordDecimal{Price: o[0], Amount: o[1]})
	}

	for _, o := range data.Tick.Bids {
		d.BidList = append(d.BidList, goex.DepthRecordDecimal{Price: o[0], Amount: o[1]})
	}

	return d
}

// CloseWs Close websocket
func (bibull *BiBull) CloseWs() {
	bibull.ws.CloseWs()
}

// SetErrorHandler Set error handler
func (bibull *BiBull) SetErrorHandler(handle func(error)) {
	bibull.errorHandle = handle
}
