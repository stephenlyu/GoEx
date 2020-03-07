package bicc

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"github.com/shopspring/decimal"
	"regexp"
	"strings"
	"github.com/pborman/uuid"
	"io/ioutil"
	"compress/gzip"
	"bytes"
)

var (
	_DEPTH_CH_PATTERN, _ = regexp.Compile("market_([a-zA-Z0-9_]+)_depth.step0")
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

func (bicc *Bicc) createWsConn() {
	if bicc.ws == nil {
		//connect wsx
		bicc.createWsLock.Lock()
		defer bicc.createWsLock.Unlock()

		if bicc.ws == nil {
			bicc.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			bicc.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			bicc.wsSymbolMap = make(map[string]string)

			bicc.ws = NewWsConn("wss://ws.bi.cc/kline-api/ws")
			//bicc.ws.Heartbeat(func() interface{} {
			//	return map[string]interface{} {"ping": time.Now().UnixNano()/1e6}
			//}, 20*time.Second)
			bicc.ws.SetErrorHandler(bicc.errorHandle)
			bicc.ws.ReConnect()
			bicc.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, err := GzipDecode(msg)
				if err != nil {
					fmt.Println(err)
					return
				}
				//println(string(msg))

				var data struct {
					EventRep string    `json:"event_rep"`
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
					bicc.ws.WriteJSON(map[string]interface{} {"pong": data.Ping})
					bicc.ws.UpdateActivedTime()
					return
				}

				if data.EventRep != "" {
					return
				}

				if data.Tick == nil {
					return
				}

				switch {
				case _DEPTH_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					symbol = bicc.wsSymbolMap[symbol]
					depth := bicc.parseDepth(msg)
					bicc.wsDepthHandleMap[data.Channel](depth)
				case _TRADE_CH_PATTERN.Match([]byte(data.Channel)):
					symbol := strings.Split(data.Channel, "_")[1]
					symbol = bicc.wsSymbolMap[symbol]
					depth := bicc.parseTrade(msg)
					bicc.wsTradeHandleMap[data.Channel](symbol, depth)
				}
			})
		}
	}
}

func (bicc *Bicc) GetDepthWithWs(oSymbol string, handle func(*DepthDecimal)) error {
	bicc.createWsConn()
	symbol := bicc.transSymbol(oSymbol)
	bicc.wsSymbolMap[symbol] = oSymbol

	channel := fmt.Sprintf("market_%s_depth_step0", symbol)

	bicc.wsDepthHandleMap[channel] = handle
	return bicc.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id": uuid.New(),
			"asks": 150,
			"bids": 150,
		},
	})
}

func (bicc *Bicc) GetTradeWithWs(oSymbol string, handle func(string, []TradeDecimal)) error {
	bicc.createWsConn()
	symbol := bicc.transSymbol(oSymbol)
	bicc.wsSymbolMap[symbol] = oSymbol

	channel := fmt.Sprintf("market_%s_trade_ticker", symbol)

	bicc.wsTradeHandleMap[channel] = handle
	return bicc.ws.Subscribe(map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id": uuid.New(),
		},
	})
}

func (bicc *Bicc) parseTrade(msg []byte) []TradeDecimal {
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

	ret := make([]TradeDecimal, len(data.Tick.Data))
	for i := range data.Tick.Data {
		o := &data.Tick.Data[i]
		ret[i] = TradeDecimal{
			Tid: o.ID,
			Type: strings.ToLower(o.Side),
			Amount: o.Vol,
			Price: o.Price,
			Date: o.Ts,
		}
	}
	return ret
}

func (bicc *Bicc) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Tick struct {
			Asks [][]decimal.Decimal    `json:"asks"`
			Bids [][]decimal.Decimal    `json:"buys"`
		}
	}

	json.Unmarshal(msg, &data)

	var d = new(DepthDecimal)

	for _, o := range data.Tick.Asks {
		d.AskList = append(d.AskList, DepthRecordDecimal{Price:o[0], Amount:o[1]})
	}

	for _, o := range data.Tick.Bids {
		d.BidList = append(d.BidList, DepthRecordDecimal{Price:o[0], Amount:o[1]})
	}

	return d
}

func (bicc *Bicc) CloseWs() {
	bicc.ws.CloseWs()
}

func (this *Bicc) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
