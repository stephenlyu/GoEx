package okexadapter

import (
	"net/http"
	"github.com/stephenlyu/tds/entity"
	"github.com/stephenlyu/tds/quoter"
	"github.com/stephenlyu/GoEx/okcoin"
	"github.com/stephenlyu/GoEx"
	"time"
	"math"
)

type OKExQuoter struct {
	okex *okcoin.OKEx

	tickMap map[string]*entity.TickItem
	firstTrade bool

	callback quoter.QuoterCallback
}

func newOKExQuoter() quoter.Quoter {
	return &OKExQuoter{
		okex: okcoin.NewOKEx(http.DefaultClient, "", ""),
		tickMap: make(map[string]*entity.TickItem),
		firstTrade: true,
	}
}

func (this *OKExQuoter) Subscribe(security *entity.Security) {
	this.tickMap[security.String()] = &entity.TickItem{Code: security.String()}

	pair, contractType := FromSecurity(security)
	this.okex.GetDepthWithWs(pair, contractType, 20, this.onDepth)
	this.okex.GetTradeWithWs(pair, contractType, this.onTrade)
}

func (this *OKExQuoter) SetCallback(callback quoter.QuoterCallback) {
	this.callback = callback
}

func (this *OKExQuoter) Destroy() {
	this.okex.CloseWs()
}

func (this *OKExQuoter) onDepth(depth *goex.Depth) {
	security := ToSecurity(depth.Pair, depth.ContractType)

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

	thisTick.High = 0
	thisTick.Low = 0
	thisTick.Amount = 0
	thisTick.Volume = 0
	thisTick.Side = entity.TICK_SIDE_UNKNOWN
}

func (this *OKExQuoter) onTrade(pair goex.CurrencyPair, contractType string, trades []goex.Trade) {
	// 忽略第一次收到的Trade
	if this.firstTrade {
		this.firstTrade = false
		return
	}

	if len(trades) == 0 {
		return
	}

	security := ToSecurity(pair, contractType)
	thisTick := this.tickMap[security.String()]

	high, low, amount, volume, side := thisTick.High, thisTick.Low, thisTick.Amount, thisTick.Volume, thisTick.Side
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

		volume += t.Amount
		if t.Price != 0 {
			if pair.CurrencyA.Symbol == "BTC" {
				amount += t.Amount * 100 / t.Price
			} else {
				amount += t.Amount * 10 / t.Price
			}
		}

		if t.Type == "bid" {
			side = entity.TICK_SIDE_BUY
		} else if t.Type == "ask" {
			side = entity.TICK_SIDE_SELL
		} else {
			side = entity.TICK_SIDE_UNKNOWN
		}

		price = t.Price
	}

	thisTick.Price, thisTick.High, thisTick.Low, thisTick.Amount, thisTick.Volume, thisTick.Side = price, high, low, amount, volume, side
}
