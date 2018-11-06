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

type BitmexQuoter struct {
	api *bitmex.BitMexWs

	tickMap map[string]*entity.TickItem
	firstTrade bool

	prevDepth map[string]*goex.Depth

	callback quoter.QuoterCallback
}

func newBitmexQuoter() quoter.Quoter {
	return &BitmexQuoter{
		api: bitmex.NewBitMexWs("", ""),
		tickMap: make(map[string]*entity.TickItem),
		firstTrade: true,
		prevDepth: make(map[string]*goex.Depth),
	}
}

func (this *BitmexQuoter) Subscribe(security *entity.Security) {
	this.tickMap[security.String()] = &entity.TickItem{Code: security.String()}

	pair := FromSecurity(security)
	this.api.GetDepthWithWs(pair, this.onDepth)
	this.api.GetTradeWithWs(pair, this.onTrade)
}

func (this *BitmexQuoter) SetCallback(callback quoter.QuoterCallback) {
	this.callback = callback
}

func (this *BitmexQuoter) Destroy() {
	this.api.CloseWs()
}

func (this *BitmexQuoter) depthEquals(d1, d2 *goex.Depth) bool {
	if d1 == nil || d2 == nil {
		return false
	}

	if d1.Pair != d2.Pair {
		return false
	}

	if len(d1.AskList) != len(d2.AskList) || len(d1.BidList) != len(d2.BidList) {
		return false
	}

	for i := range d1.AskList {
		if math.Abs(d1.AskList[i].Price - d2.AskList[i].Price) > 0.01 {
			return false
		}
		if d1.AskList[i].Amount != d2.AskList[i].Amount {
			return false
		}
	}

	for i := range d1.BidList {
		if math.Abs(d1.BidList[i].Price - d2.BidList[i].Price) > 0.01 {
			return false
		}
		if d1.BidList[i].Amount != d2.BidList[i].Amount {
			return false
		}
	}
	return true
}

func (this *BitmexQuoter) onDepth(depth *goex.Depth) {
	prevDepth, _ := this.prevDepth[depth.Pair.String()]
	if this.depthEquals(prevDepth, depth) {
		return
	}

	security := ToSecurity(depth.Pair)

	thisTick := this.tickMap[security.String()]
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

	this.prevDepth[depth.Pair.String()] = depth
}

func (this *BitmexQuoter) onTrade(pair goex.CurrencyPair, trades []goex.Trade) {
	// 忽略第一次收到的Trade
	if this.firstTrade {
		this.firstTrade = false
		return
	}

	if len(trades) == 0 {
		return
	}

	security := ToSecurity(pair)
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
