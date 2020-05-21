package atop

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"github.com/shopspring/decimal"
	"strings"
	"io/ioutil"
	"compress/gzip"
	"bytes"
	"sort"
)

func GzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (atop *Atop) createWsConn() {
	if atop.ws == nil {
		//connect wsx
		atop.createWsLock.Lock()
		defer atop.createWsLock.Unlock()

		if atop.ws == nil {
			atop.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			atop.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			atop.wsSymbolMap = make(map[string]string)
			atop.depthManagers = make(map[string]*DepthManager)

			atop.ws = NewWsConn("wss://socket.a.top/websocket")
			atop.ws.SetErrorHandler(atop.errorHandle)
			atop.ws.ReConnect()
			atop.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				var err error
				//println(string(msg))
				//msg, err := GzipDecode(msg)
				//if err != nil {
				//	fmt.Println(err)
				//	return
				//}
				//println(string(msg))

				var data struct {
					Code decimal.Decimal
					Ping int64
					Data *struct {
						Channel string
						Market  string
					}
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Ping > 0 {
					atop.ws.WriteJSON(map[string]interface{}{"pong": data.Ping})
					atop.ws.UpdateActivedTime()
					return
				}

				if data.Code.IntPart() != 200 {
					return
				}

				if data.Data == nil {
					return
				}

				switch data.Data.Channel {
				case "ex_depth_data":
					depth := atop.parseDepth(msg)
					channel := fmt.Sprintf("ex_depth_data_%s", data.Data.Market)
					atop.wsDepthHandleMap[channel](depth)
				case "ex_last_trade":
					trades := atop.parseTrade(msg)
					channel := fmt.Sprintf("ex_last_trade_%s", data.Data.Market)
					atop.wsTradeHandleMap[channel](strings.ToUpper(data.Data.Market), trades)
				}
			})
		}
	}
}

func (atop *Atop) GetDepthWithWs(oSymbol string, handle func(*DepthDecimal)) error {
	atop.createWsConn()
	symbol := atop.transSymbol(oSymbol)

	channel := fmt.Sprintf("ex_depth_data_%s", symbol)

	atop.wsDepthHandleMap[channel] = handle
	atop.depthManagers[symbol] = NewDepthManager()
	return atop.ws.Subscribe(map[string]interface{}{
		"channel": "ex_depth_data",
		"market": symbol,
		"event": "addChannel",
	})
}

func (atop *Atop) GetTradeWithWs(oSymbol string, handle func(string, []TradeDecimal)) error {
	atop.createWsConn()
	symbol := atop.transSymbol(oSymbol)

	channel := fmt.Sprintf("ex_last_trade_%s", symbol)

	atop.wsTradeHandleMap[channel] = handle
	return atop.ws.Subscribe(map[string]interface{}{
		"channel": "ex_last_trade",
		"market": symbol,
		"event": "addChannel",
	})
}

func (atop *Atop) parseTrade(msg []byte) []TradeDecimal {
	var data *struct {
		Data struct {
				 Market  string
				 Records [][]interface{}
			 }
	}

	json.Unmarshal(msg, &data)

	ret := make([]TradeDecimal, len(data.Data.Records))
	for i, o := range data.Data.Records {
		t := &ret[i]

		t.Tid = int64(o[4].(float64))
		t.Amount = decimal.NewFromFloat(o[2].(float64))
		t.Price = decimal.NewFromFloat(o[1].(float64))
		t.Type = strings.ToLower(o[3].(string))
	}
	return ret
}

func (atop *Atop) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Data struct {
				 Market string
				 IsFull bool
				 Asks   [][]decimal.Decimal
				 Bids   [][]decimal.Decimal
			 }
	}

	json.Unmarshal(msg, &data)

	dm := atop.depthManagers[data.Data.Market]

	var d = new(DepthDecimal)

	d.AskList, d.BidList = dm.Update(data.Data.IsFull, data.Data.Asks, data.Data.Bids)

	return d
}

func (atop *Atop) CloseWs() {
	atop.ws.CloseWs()
}

func (this *Atop) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}

type DepthManager struct {
	buyMap  map[string]DepthRecordDecimal
	sellMap map[string]DepthRecordDecimal
}

func NewDepthManager() *DepthManager {
	return &DepthManager{
		buyMap: make(map[string]DepthRecordDecimal),
		sellMap: make(map[string]DepthRecordDecimal),
	}
}

func (this *DepthManager) Update(isFull bool, askList, bidList [][]decimal.Decimal) (DepthRecordsDecimal, DepthRecordsDecimal) {
	if isFull {
		this.buyMap = make(map[string]DepthRecordDecimal)
		this.sellMap = make(map[string]DepthRecordDecimal)
	}

	for _, o := range askList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.sellMap, key)
		} else {
			price := o[0]
			amount := o[1]
			this.sellMap[key] = DepthRecordDecimal{Price: price, Amount: amount}
		}
	}

	for _, o := range bidList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.buyMap, key)
		} else {
			price := o[0]
			amount := o[1]
			this.buyMap[key] = DepthRecordDecimal{Price: price, Amount: amount}
		}
	}

	bids := make(DepthRecordsDecimal, len(this.buyMap))
	i := 0
	for _, item := range this.buyMap {
		bids[i] = item
		i++
	}
	sort.SliceStable(bids, func(i, j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	asks := make(DepthRecordsDecimal, len(this.sellMap))
	i = 0
	for _, item := range this.sellMap {
		asks[i] = item
		i++
	}
	sort.SliceStable(asks, func(i, j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})
	return asks, bids
}
