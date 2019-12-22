package okexadapterv3

import (
	"net/http"
	"github.com/stephenlyu/tds/entity"
	"github.com/stephenlyu/tds/quoter"
	"github.com/stephenlyu/GoEx/okcoin"
	"github.com/stephenlyu/GoEx"
	"time"
	"math"
	"fmt"
)

type OKExQuoter struct {
	okex *okcoin.OKExV3

	tickMap map[string]*entity.TickItem
	firstTrade bool

	callback quoter.QuoterCallback

	instrumentIdSecurityMap map[string]*entity.Security
}

func newOKExQuoter() quoter.Quoter {
	this := &OKExQuoter{
		okex: okcoin.NewOKExV3(http.DefaultClient, "", "", ""),
		tickMap: make(map[string]*entity.TickItem),
		firstTrade: true,
		instrumentIdSecurityMap: make(map[string]*entity.Security),
	}
	this.okex.SetErrorHandler(func (err error) {
		if err != nil {
			if this.callback != nil {
				this.callback.OnError(err)
			}
		}
	})

	return this
}

func (this *OKExQuoter) Subscribe(security *entity.Security) {
	this.tickMap[security.String()] = &entity.TickItem{Code: security.String()}

	instrumentId := FromSecurity(security)
	this.instrumentIdSecurityMap[instrumentId] = security
	if security.IsIndex() {
		this.okex.GetIndexTickerWithWs(instrumentId, this.onTicker)
	} else {
		this.okex.GetDepthWithWs(instrumentId, this.onDepth)
		this.okex.GetTradeWithWs(instrumentId, this.onTrade)
	}
}

func (this *OKExQuoter) SetCallback(callback quoter.QuoterCallback) {
	this.callback = callback
}

func (this *OKExQuoter) Destroy() {
	this.okex.CloseWs()
}

func (this *OKExQuoter) checkInstrumentCodeChanged(instrumentId string) {
	security, _ := this.instrumentIdSecurityMap[instrumentId]
	newInstrumentId := FromSecurity(security)
	if instrumentId != newInstrumentId {
		panic("Quit for delivery")
	}
}

func (this *OKExQuoter) onDepth(depth *goex.Depth) {
	this.checkInstrumentCodeChanged(depth.InstrumentId)
	security, _ := this.instrumentIdSecurityMap[depth.InstrumentId]

	thisTick := this.tickMap[security.String()]
	thisTick.Timestamp = uint64(depth.UTime.UnixNano() / int64(time.Millisecond))

	askLen, bidLen := 20, 20
	if askLen > len(depth.AskList) {
		askLen = len(depth.AskList)
	}
	if bidLen > len(depth.BidList) {
		bidLen = len(depth.BidList)
	}

	thisTick.AskVolumes = make([]float64, askLen)
	thisTick.AskPrices = make([]float64, askLen)
	thisTick.BidVolumes = make([]float64, bidLen)
	thisTick.BidPrices = make([]float64, bidLen)

	for i := 0; i < askLen; i++ {
		r := &depth.AskList[i]
		thisTick.AskPrices[i] = r.Price
		thisTick.AskVolumes[i] = r.Amount
	}

	for i := 0; i < bidLen; i++ {
		r := &depth.BidList[i]
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
}

func (this *OKExQuoter) onTrade(instrumentId string, trades []goex.Trade) {
	// 忽略第一次收到的Trade
	if this.firstTrade {
		this.firstTrade = false
		return
	}

	if len(trades) == 0 {
		return
	}
	this.checkInstrumentCodeChanged(instrumentId)

	security, _ := this.instrumentIdSecurityMap[instrumentId]
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
			if security.Category == "BTC" {
				amount += t.Amount * 100 / t.Price
			} else {
				amount += t.Amount * 10 / t.Price
			}
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

func (this *OKExQuoter) onTicker(instrumentId string, tickers []goex.Ticker) {
	security, ok := this.instrumentIdSecurityMap[instrumentId]
	if !ok {
		return
	}

	if len(tickers) == 0 {
		return
	}
	this.checkInstrumentCodeChanged(instrumentId)

	ticker := tickers[0]

	tick := &entity.TickItem{
		Code: security.String(),
		Timestamp: uint64(ticker.Date),
		Price: ticker.Last,
		High: ticker.Last,
		Low: ticker.Last,
		Volume: 1,
	}
	if this.callback != nil {
		this.callback.OnTickItem(tick)
	}
}