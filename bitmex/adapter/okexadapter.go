package bitmexadapter

import (
	"github.com/stephenlyu/tds/entity"
	"github.com/stephenlyu/tds/quoter"
	"github.com/stephenlyu/GoEx"
	"time"
	"math"
	"fmt"
	"github.com/stephenlyu/GoEx/bitmex"
)

const DEPTH_INTERVAL = 1000

type BitmexQuoter struct {
	api *bitmex.BitMexWs

	tickMap map[string]*entity.TickItem
	firstTrade bool

	lastTimestamps map[string]int64

	callback quoter.QuoterCallback
}

func newBitmexQuoter() quoter.Quoter {
	return &BitmexQuoter{
		api: bitmex.NewBitMexWs("", ""),
		tickMap: make(map[string]*entity.TickItem),
		firstTrade: true,
		lastTimestamps: make(map[string]int64),
	}
}

func (this *BitmexQuoter) Subscribe(security *entity.Security) {
	this.tickMap[security.String()] = &entity.TickItem{Code: security.String()}

	symbol := FromSecurity(security)
	this.api.GetDepthWithWs(symbol, this.onDepth)
	this.api.GetTradeWithWs(symbol, this.onTrade)
}

func (this *BitmexQuoter) SetCallback(callback quoter.QuoterCallback) {
	this.callback = callback
}

func (this *BitmexQuoter) Destroy() {
	this.api.CloseWs()
}

func (this *BitmexQuoter) onDepth(depth *goex.Depth) {
	lastTs, _ := this.lastTimestamps[depth.Pair.String()]
	ts := depth.UTime.UnixNano() / 1000000

	security := ToSecurity(depth.Symbol)
	thisTick := this.tickMap[security.String()]
	if thisTick.Volume == 0 && ts - lastTs < DEPTH_INTERVAL {
		return
	}

	thisTick.Timestamp = uint64(depth.UTime.UnixNano() / int64(time.Millisecond))

	thisTick.AskVolumes = make([]float64, len(depth.AskList))
	thisTick.AskPrices = make([]float64, len(depth.AskList))
	thisTick.BidVolumes = make([]float64, len(depth.BidList))
	thisTick.BidPrices = make([]float64, len(depth.BidList))

	for i, r := range depth.AskList {
		thisTick.AskPrices[i] = r.Price
		thisTick.AskVolumes[i] = r.Amount
	}

	for i, r := range depth.BidList {
		thisTick.BidPrices[i] = r.Price
		thisTick.BidVolumes[i] = r.Amount
	}

	tick := *thisTick

	if this.callback != nil {
		this.callback.OnTickItem(&tick)
	}

	thisTick.Open = 0
	thisTick.High = 0
	thisTick.Low = 0
	thisTick.Amount = 0
	thisTick.Volume = 0
	thisTick.Side = entity.TICK_SIDE_UNKNOWN
	thisTick.BuyVolume = 0
	thisTick.SellVolume = 0

	this.lastTimestamps[depth.Pair.String()] = ts
}

func (this *BitmexQuoter) onTrade(symbol string, trades []goex.Trade) {
	// 忽略第一次收到的Trade
	if this.firstTrade {
		this.firstTrade = false
		return
	}

	if len(trades) == 0 {
		return
	}

	security := ToSecurity(symbol)
	thisTick := this.tickMap[security.String()]

	open, high, low, amount, volume, side, buyVolume, sellVolume := thisTick.Open, thisTick.High, thisTick.Low, thisTick.Amount, thisTick.Volume, thisTick.Side, thisTick.BuyVolume, thisTick.SellVolume
	var price float64

	for i := range trades {
		t := &trades[i]
		if high == 0 {
			high = t.Price
		} else {
			high = math.Max(high, t.Price)
		}

		if low == 0 {
			low = t.Price
		} else {
			low = math.Min(low, t.Price)
		}

		if open == 0 {
			open = t.Price
		}

		volume += t.Amount
		if t.Price != 0 {
			amount += t.Amount / t.Price
		}

		if t.Type == "buy" {
			side = entity.TICK_SIDE_BUY
			buyVolume += t.Amount
		} else if t.Type == "sell" {
			side = entity.TICK_SIDE_SELL
			sellVolume += t.Amount
		} else {
			side = entity.TICK_SIDE_UNKNOWN
			fmt.Println("unknown", t.Type)
		}

		price = t.Price
	}

	thisTick.Open, thisTick.Price, thisTick.High, thisTick.Low, thisTick.Amount, thisTick.Volume, thisTick.Side, thisTick.BuyVolume, thisTick.SellVolume = open, price, high, low, amount, volume, side, buyVolume, sellVolume
}
