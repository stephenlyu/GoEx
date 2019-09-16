package binancefuture

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	"github.com/pborman/uuid"
	"github.com/z-ray/log"
	"github.com/gorilla/websocket"
	"sync"
	"sort"
)


func (this *Binance) createDataWsConn(symbols []string) {
	this.wsLock.Lock()
	defer this.wsLock.Unlock()
	if this.wsData != nil {
		return
	}
	this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
	this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
	this.depthManagers = make(map[string]*DepthManager)

	var streams []string
	var symbolMap = make(map[string]string)
	for _, rawSymbol := range symbols {
		pair := NewCurrencyPair2(rawSymbol)
		symbol := this.transSymbol(rawSymbol)
		streamSymbol := strings.ToLower(symbol)
		symbolMap[streamSymbol] = rawSymbol
		streams = append(streams, streamSymbol + "@depth")
		streams = append(streams, streamSymbol + "@aggTrade")

		dm := NewDepthManager(this, pair)
		dm.Start()
		this.depthManagers[pair.ToSymbol("")] = dm
	}

	url := fmt.Sprintf("wss://fstream.binance.com/stream?streams=%s", strings.Join(streams, "/"))
	ws := NewWsConn(url)
	ws.SetErrorHandler(this.errorHandle)
	ws.HeartbeatEx(func() (int, string) {return websocket.PongMessage, "pong"}, 20*time.Second)
	ws.ReConnect()
	ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
		//println(string(msg))

		// 只要收到消息，就说明连接还是活的
		ws.UpdateActivedTime()
		var data struct {
			Stream string
		}
		err := json.Unmarshal(msg, &data)
		if err != nil {
			log.Print(err)
			return
		}

		switch {
		case strings.HasSuffix(data.Stream, "@depth"):
			du := this.parseDepth(msg)
			dm := this.depthManagers[du.Symbol]
			depth := dm.Feed(du)
			pairSymbol := symbolMap[strings.ToLower(du.Symbol)]
			if depth != nil {
				this.wsDepthHandleMap[pairSymbol](depth)
			}
		case strings.HasSuffix(data.Stream, "@aggTrade"):
			symbol, trades := this.parseTrade(msg)
			pairSymbol := symbolMap[strings.ToLower(symbol)]
			this.wsTradeHandleMap[pairSymbol](pairSymbol, trades)
		}
	})
	this.wsData = ws
}

func (this *Binance) newId() string {
	return uuid.New()
}

func (this *Binance) transSymbol(symbol string) string {
	return strings.ToLower(strings.Replace(symbol, "_", "", -1))
}

func (this *Binance) GetDepthTradeWithWs(symbols []string, depthCB func(*DepthDecimal), tradeCB func(string, []TradeDecimal)) error {
	this.createDataWsConn(symbols)
	for _, symbol := range symbols {
		this.wsDepthHandleMap[symbol] = depthCB
		this.wsTradeHandleMap[symbol] = tradeCB
	}
	return nil
}

func (this *Binance) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data *struct {
		Data map[string]interface{}
	}
	json.Unmarshal(msg, &data)
	r := data.Data

	var side string
	if r["m"].(bool) {
		side = "sell"
	} else {
		side = "buy"
	}

	symbol := r["s"].(string)
	amount, _ := decimal.NewFromString(r["q"].(string))
	price, _ := decimal.NewFromString(r["p"].(string))
	timestamp := r["T"].(float64)

	return symbol, []TradeDecimal {
		{
			Type: side,
			Amount: amount,
			Price: price,
			Date: int64(timestamp),
		},
	}
}

func (this *Binance) parseDepth(msg []byte) *DepthUpdate {
	var data *struct {
		Data DepthUpdate
	}

	json.Unmarshal(msg, &data)

	return &data.Data
}

func (this *Binance) CloseWs() {
	this.wsData.Close()
}

func (this *Binance) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}

const (
	DmStateInit = iota
	DmStatePull			// Restful拉取数据
	DmStateWaitValidData		// 等待合并后第一个合法数据
	DmStateNormal				// 进入正常模式
)

type DepthManager struct {
	ba *Binance
	pair CurrencyPair

	state int

	depthUpdates []*DepthUpdate
	lastUpdateId int64

	lastU int64

	askMap map[string]DepthRecordDecimal
	bidMap map[string]DepthRecordDecimal

	lock sync.RWMutex
}

func NewDepthManager(ba *Binance, pair CurrencyPair) *DepthManager {
	dm := new(DepthManager)
	dm.state = DmStateInit
	dm.ba = ba
	dm.pair = pair
	return dm
}

func (this *DepthManager) SetState(state int) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.state = state
}

func (this *DepthManager) Start() {
	this.askMap = make(map[string]DepthRecordDecimal)
	this.bidMap = make(map[string]DepthRecordDecimal)

	go func() {
		this.SetState(DmStatePull)

		var d *DepthData
		var err error

		for i := 0; i < 3; i++ {
			d, err = this.ba.GetDepthInternal(500, this.pair)
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			if this.ba.errorHandle != nil {
				this.ba.errorHandle(err)
			}
			return
		}

		this.lock.Lock()
		defer this.lock.Unlock()

		for _, r := range d.Asks {
			price := r[0]
			qty := r[1]
			this.askMap[price.String()] = DepthRecordDecimal{
				Price: price,
				Amount: qty,
			}
		}
		for _, r := range d.Bids {
			price := r[0]
			qty := r[1]
			this.bidMap[price.String()] = DepthRecordDecimal{
				Price: price,
				Amount: qty,
			}
		}

		for _, du := range this.depthUpdates {
			if du.ULast < d.LastUpdateId {
				continue
			}
			this.applyDu(du)
		}

		this.depthUpdates = nil
		this.lastUpdateId = d.LastUpdateId
		this.state = DmStateWaitValidData
	} ()
}

func (this *DepthManager) Feed(du *DepthUpdate) *DepthDecimal {
	var ret *DepthDecimal
	switch this.state {
	case DmStateInit, DmStatePull:
		this.lock.Lock()
		this.depthUpdates = append(this.depthUpdates, du)
		this.lock.Unlock()
	case DmStateWaitValidData:
		if du.ULast < this.lastUpdateId {
			break
		}
		this.SetState(DmStateNormal)
		fallthrough
	case DmStateNormal:
		if this.lastU > 0 && this.lastU != du.PrevU {
			println("missing packets")
			this.lastU = 0
			this.lastUpdateId = 0
			this.SetState(DmStateInit)
			this.Start()
			break
		}

		this.applyDu(du)

		ret = new(DepthDecimal)
		timestamp := du.EventTs
		ret.UTime = time.Unix(timestamp / 1000, timestamp % 1000)

		ret.AskList = make(DepthRecordsDecimal, len(this.askMap))
		ret.BidList = make(DepthRecordsDecimal, len(this.bidMap))

		var i int
		for _, r := range this.askMap {
			ret.AskList[i] = r
			i++
		}
		sort.Slice(ret.AskList, func(i,j int) bool {
			return ret.AskList[i].Price.LessThan(ret.AskList[j].Price)
		})

		i = 0
		for _, r := range this.bidMap {
			ret.BidList[i] = r
			i++
		}
		sort.Slice(ret.BidList, func(i,j int) bool {
			return ret.BidList[i].Price.GreaterThan(ret.BidList[j].Price)
		})
		this.lastU = du.ULast
	}
	return ret
}

func (this *DepthManager) applyDu(du *DepthUpdate) {
	for _, r := range du.Asks {
		price := r[0]
		qty := r[1]
		if qty.IsZero() {
			delete(this.askMap, price.String())
		} else {
			this.askMap[price.String()] = DepthRecordDecimal{
				Price: price,
				Amount: qty,
			}
		}
	}
	for _, r := range du.Bids {
		price := r[0]
		qty := r[1]
		if qty.IsZero() {
			delete(this.bidMap, price.String())
		} else {
			this.bidMap[price.String()] = DepthRecordDecimal{
				Price: price,
				Amount: qty,
			}
		}
	}
}