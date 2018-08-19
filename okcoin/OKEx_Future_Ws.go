package okcoin

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"strings"
	"time"
)

func (okFuture *OKEx) createWsConn() {
	if okFuture.ws == nil {
		//connect wsx
		okFuture.createWsLock.Lock()
		defer okFuture.createWsLock.Unlock()

		if okFuture.ws == nil {
			okFuture.wsTickerHandleMap = make(map[string]func(*Ticker))
			okFuture.wsDepthHandleMap = make(map[string]func(*Depth))
			okFuture.wsTradeHandleMap = make(map[string]func(CurrencyPair, string, []Trade))

			okFuture.ws = NewWsConn("wss://real.okex.com:10440/websocket/okexapi")
			okFuture.ws.Heartbeat(func() interface{} { return map[string]string{"event": "ping"} }, 30*time.Second)
			okFuture.ws.ReConnect()
			okFuture.ws.ReceiveMessage(func(msg []byte) {
				if string(msg) == "{\"event\":\"pong\"}" {
					okFuture.ws.UpdateActivedTime()
					return
				}

				var data []interface{}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if len(data) == 0 {
					return
				}

				datamap := data[0].(map[string]interface{})
				channel := datamap["channel"].(string)
				if channel == "addChannel" {
					return
				}

				pair := okFuture.getPairFromChannel(channel)
				contractType := okFuture.getContractFromChannel(channel)

				if strings.Contains(channel, "_trade") {
					data := datamap["data"].([]interface{})
					trades := okFuture.parseTrade(data)
					okFuture.wsTradeHandleMap[channel](pair, contractType, trades)
				} else {
					tickmap := datamap["data"].(map[string]interface{})

					if strings.Contains(channel, "_ticker") {
						ticker := okFuture.parseTicker(tickmap)
						ticker.Pair = pair
						ticker.ContractType = contractType
						okFuture.wsTickerHandleMap[channel](ticker)
					} else if strings.Contains(channel, "depth_") {
						dep := okFuture.parseDepth(tickmap)
						dep.Pair = pair
						dep.ContractType = contractType
						okFuture.wsDepthHandleMap[channel](dep)
					}
				}
			})
		}
	}
}

func (okFuture *OKEx) GetDepthWithWs(pair CurrencyPair, contractType string, n int, handle func(*Depth)) error {
	if n == 0 {
		n = 5
	}
	okFuture.createWsConn()
	channel := fmt.Sprintf("ok_sub_futureusd_%s_depth_%s_%d", strings.ToLower(pair.CurrencyA.Symbol), contractType, n)
	okFuture.wsDepthHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]string{
		"event":   "addChannel",
		"channel": channel})
}

func (okFuture *OKEx) GetTickerWithWs(pair CurrencyPair, contractType string, handle func(*Ticker)) error {
	okFuture.createWsConn()
	channel := fmt.Sprintf("ok_sub_futureusd_%s_ticker_%s", strings.ToLower(pair.CurrencyA.Symbol), contractType)
	okFuture.wsTickerHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]string{
		"event":   "addChannel",
		"channel": channel})
}

func (okFuture *OKEx) GetTradeWithWs(pair CurrencyPair, contractType string, handle func(CurrencyPair, string, []Trade)) error {
	okFuture.createWsConn()
	channel := fmt.Sprintf("ok_sub_futureusd_%s_trade_%s", strings.ToLower(pair.CurrencyA.Symbol), contractType)
	okFuture.wsTradeHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]string{
		"event":   "addChannel",
		"channel": channel})
}

func (okFuture *OKEx) parseTicker(tickmap map[string]interface{}) *Ticker {
	return &Ticker{
		Last: ToFloat64(tickmap["last"]),
		Low:  ToFloat64(tickmap["low"]),
		High: ToFloat64(tickmap["high"]),
		Vol:  ToFloat64(tickmap["vol"]),
		Sell: ToFloat64(tickmap["sell"]),
		Buy:  ToFloat64(tickmap["buy"]),
		Date: ToUint64(tickmap["timestamp"])}
}

func (okFuture *OKEx) parseTrade(data []interface{}) []Trade {
	ret := make([]Trade, len(data))
	for i := range data {
		r := data[i].([]interface{})
		tid := ToUint64(r[0])
		price := ToFloat64(r[1])
		amount := ToFloat64(r[2])
		t := r[3].(string)
		_type := r[4].(string)
		ret[i] = Trade{
			Tid: int64(tid),
			Price: price,
			Amount: amount,
			Time: t,
			Type: _type,
		}
	}
	return ret
}

func (okFuture *OKEx) parseDepth(tickmap map[string]interface{}) *Depth {
	asks := tickmap["asks"].([]interface{})
	bids := tickmap["bids"].([]interface{})

	var depth Depth

	timestamp := int64(ToUint64(tickmap["timestamp"]))
	depth.UTime = time.Unix(timestamp / 1000, timestamp % 1000 * int64(time.Millisecond))

	for _, v := range asks {
		var dr DepthRecord
		for i, vv := range v.([]interface{}) {
			switch i {
			case 0:
				dr.Price = ToFloat64(vv)
			case 1:
				dr.Amount = ToFloat64(vv)
			}
		}
		depth.AskList = append(depth.AskList, dr)
	}

	for _, v := range bids {
		var dr DepthRecord
		for i, vv := range v.([]interface{}) {
			switch i {
			case 0:
				dr.Price = ToFloat64(vv)
			case 1:
				dr.Amount = ToFloat64(vv)
			}
		}
		depth.BidList = append(depth.BidList, dr)
	}
	return &depth
}

func (okFuture *OKEx) getPairFromChannel(channel string) CurrencyPair {
	metas := strings.Split(channel, "_")
	return NewCurrencyPair2(metas[3] + "_usd")
}

func (okFuture *OKEx) getContractFromChannel(channel string) string {
	if strings.Contains(channel, THIS_WEEK_CONTRACT) {
		return THIS_WEEK_CONTRACT
	}

	if strings.Contains(channel, NEXT_WEEK_CONTRACT) {
		return NEXT_WEEK_CONTRACT
	}

	if strings.Contains(channel, QUARTER_CONTRACT) {
		return QUARTER_CONTRACT
	}
	return ""
}

func (okFuture *OKEx) CloseWs() {
	okFuture.ws.CloseWs()
}
