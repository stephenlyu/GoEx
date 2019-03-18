package okexv3spot

import (
	"encoding/json"
	"fmt"
	. "github.com/stephenlyu/GoEx"
	"log"
	"strings"
	"time"
	"compress/flate"
	"io/ioutil"
	"bytes"
	"sort"
	"github.com/shopspring/decimal"
	"strconv"
	"errors"
)

func GzipDecodeV3(in []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(in))
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (okSpot *OKExV3Spot) createWsConn() {
	if okSpot.ws == nil {
		//connect wsx
		okSpot.createWsLock.Lock()
		defer okSpot.createWsLock.Unlock()

		if okSpot.ws == nil {
			okSpot.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			okSpot.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))
			okSpot.wsAccountHandleMap = make(map[string]func(*SubAccountDecimal))
			okSpot.wsOrderHandleMap = make(map[string]func([]OrderDecimal))
			okSpot.depthManagers = make(map[string]*DepthManager)

			okSpot.ws = NewWsConn("wss://real.okex.com:10442/ws/v3")
			okSpot.ws.Heartbeat(func() interface{} { return "ping"}, 20*time.Second)
			okSpot.ws.ReConnect()
			okSpot.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				if isBin {
					msg, _ = GzipDecodeV3(msg)
				}
				//println(string(msg))
				if string(msg) == "pong" {
					okSpot.ws.UpdateActivedTime()
					return
				}

				var data struct {
					Event string
					ErrorCode int
					Message string
					Success bool
					Table string
					Action string
					Data []interface{}
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				if data.Event == "login" {
					var err error
					if !data.Success {
						err = errors.New("Login failure")
					}
					if okSpot.wsLoginHandle != nil {
						okSpot.wsLoginHandle(err)
					}
					return
				}

				if len(data.Data) == 0 {
					return
				}

				switch data.Table  {
				case "spot/trade":
				instrumentId, trades := okSpot.parseTrade(msg)
				if instrumentId != "" {
					topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
					okSpot.wsTradeHandleMap[topic](instrumentId, trades)
				}
				case "spot/depth":
					depth := okSpot.parseDepth(msg)
					if depth != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, depth.InstrumentId)
						okSpot.wsDepthHandleMap[topic](depth)
					}
				case "futures/account":
					account := okSpot.parseAccount(msg)
					if account != nil {
						okSpot.wsAccountHandleMap[data.Table](account)
					}
				case "futures/order":
					instrumentId, orders := okSpot.parseOrder(msg)
					if orders != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okSpot.wsOrderHandleMap[topic](orders)
					}
				}
			})
		}
	}
}

func (okSpot *OKExV3Spot) Login(handle func(error)) error {
	okSpot.createWsConn()
	okSpot.wsLoginHandle = handle

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET/users/self/verify"
	sign, _ := GetParamHmacSHA256Base64Sign(okSpot.apiSecretKey, message)

	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "login",
		"args": []interface{}{okSpot.apiKey, okSpot.passphrase, timestamp, sign},
	})
}

func (okSpot *OKExV3Spot) GetDepthWithWs(instrumentId string, handle func(*DepthDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/depth:%s", instrumentId)
	okSpot.wsDepthHandleMap[channel] = handle
	okSpot.depthManagers[instrumentId] = NewDepthManager()
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okSpot *OKExV3Spot) GetTradeWithWs(instrumentId string, handle func(string, []TradeDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/trade:%s", instrumentId)
	okSpot.wsTradeHandleMap[channel] = handle
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okSpot *OKExV3Spot) GetAccountWithWs(currency Currency, handle func(*SubAccountDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/account:%s", currency.Symbol)
	okSpot.wsAccountHandleMap[channel] = handle
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okSpot *OKExV3Spot) GetOrderWithWs(instrumentId string, handle func([]OrderDecimal)) error {
	okSpot.createWsConn()

	channel := fmt.Sprintf("spot/order:%s", instrumentId)
	okSpot.wsOrderHandleMap[channel] = handle
	return okSpot.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okSpot *OKExV3Spot) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data *struct {
		Table  string
		Action string
		Data   []struct {
			InstrumentId string			`json:"instrument_id"`
			Price decimal.Decimal
			Side string
			Size decimal.Decimal
			Timestamp string
			TradeId decimal.Decimal 	`json:"trade_id"`
		}
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]TradeDecimal, len(data.Data))
	for i := range data.Data {
		o := &data.Data[i]
		timestamp := V3ParseDate(o.Timestamp)
		ret[i] = TradeDecimal {
			Tid: o.TradeId.IntPart(),
			Type: o.Side,
			Amount: o.Size,
			Price: o.Price,
			Date: timestamp,
		}
	}

	return instrumentId, ret
}

func (okSpot *OKExV3Spot) parseDepth(msg []byte) *DepthDecimal {
	var data *struct {
		Table string
		Action string
		Data []struct {
			InstrumentId string			`json:"instrument_id"`
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal
			Timestamp string
			Checksum int
		}
	}

	json.Unmarshal(msg, &data)

	timestamp := V3ParseDate(data.Data[0].Timestamp)
	instrumentId := data.Data[0].InstrumentId
	parts := strings.Split(instrumentId, "-")
	depthManager, _ := okSpot.depthManagers[instrumentId]
	if depthManager == nil {
		panic("Illegal state error")
	}

	asks, bids := depthManager.Update(data.Action, data.Data[0].Asks, data.Data[0].Bids)
	return &DepthDecimal{
		InstrumentId: instrumentId,
		Pair: CurrencyPair{Currency{Symbol: parts[0]}, Currency{Symbol: parts[1]}},
		UTime: time.Unix(timestamp / 1000, timestamp % 1000 * int64(time.Millisecond)),
		AskList: asks,
		BidList: bids,
	}
}

func (okSpot *OKExV3Spot) parseAccount(msg []byte) *SubAccountDecimal {
	var data *struct {
		Table  string
		Action string
		Data   []struct {
			Balance decimal.Decimal
			Available decimal.Decimal
			Hold decimal.Decimal
			Id string
			Currency string
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Data[0]
	currency := Currency{Symbol: r.Currency}
	return &SubAccountDecimal{
		Currency: currency,
		Amount: r.Balance,
		AvailableAmount: r.Available,
		FrozenAmount: r.Hold,
	}
}

func (okSpot *OKExV3Spot) parseOrder(msg []byte) (string, []OrderDecimal) {
	// TODO:
	var data *struct {
		Table  string
		Action string
		Data   []V3OrderInfo
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]OrderDecimal, len(data.Data))
	//for i := range data.Data {
	//	ret[i] = *data.Data[i].ToFutureOrder()
	//}

	return instrumentId, ret
}

func (okSpot *OKExV3Spot) CloseWs() {
	okSpot.ws.CloseWs()
}

type DepthManager struct {
	buyMap map[string][]decimal.Decimal
	sellMap map[string][]decimal.Decimal
}

func NewDepthManager() *DepthManager {
	return &DepthManager{
		buyMap: make(map[string][]decimal.Decimal),
		sellMap: make(map[string][]decimal.Decimal),
	}
}

func (this *DepthManager) Update(action string, askList, bidList [][]decimal.Decimal) (DepthRecordsDecimal, DepthRecordsDecimal) {
	if action == "partial" {
		this.buyMap = make(map[string][]decimal.Decimal)
		this.sellMap = make(map[string][]decimal.Decimal)
	}

	for _, o := range askList {
		price := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.sellMap, price)
		} else {
			this.sellMap[price] = o
		}
	}

	for _, o := range bidList {
		price := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.buyMap, price)
		} else {
			this.buyMap[price] = o
		}
	}

	bids := make(DepthRecordsDecimal, len(this.buyMap))
	i := 0
	for _, item := range this.buyMap {
		bids[i] = DepthRecordDecimal{Price: item[0], Amount: item[1]}
		i++
	}
	sort.SliceStable(bids, func(i,j int) bool {
		return bids[i].Price.GreaterThan(bids[j].Price)
	})

	asks := make(DepthRecordsDecimal, len(this.sellMap))
	i = 0
	for _, item := range this.sellMap {
		asks[i] = DepthRecordDecimal{Price: item[0], Amount: item[1]}
		i++
	}
	sort.SliceStable(asks, func(i,j int) bool {
		return asks[i].Price.LessThan(asks[j].Price)
	})
	return asks, bids
}
