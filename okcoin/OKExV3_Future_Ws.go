package okcoin

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

func (okFuture *OKExV3) createWsConn() {
	if okFuture.ws == nil {
		//connect wsx
		okFuture.createWsLock.Lock()
		defer okFuture.createWsLock.Unlock()

		if okFuture.ws == nil {
			okFuture.wsDepthHandleMap = make(map[string]func(*Depth))
			okFuture.wsTradeHandleMap = make(map[string]func(string, []Trade))
			okFuture.wsIndexTickerHandleMap = make(map[string]func(string, []Ticker))
			okFuture.wsFundingRateHandleMap = make(map[string]func(SWAPFundingRate))
			okFuture.wsPositionHandleMap = make(map[string]func([]FuturePosition))
			okFuture.wsAccountHandleMap = make(map[string]func(bool, *FutureAccount))
			okFuture.wsOrderHandleMap = make(map[string]func([]FutureOrder))
			okFuture.depthManagers = make(map[string]*DepthManager)

			okFuture.ws = NewWsConn("wss://real.okex.com:10442/ws/v3")
			okFuture.ws.Heartbeat(func() interface{} { return "ping"}, 20*time.Second)
			okFuture.ws.SetErrorHandler(okFuture.errorHandle)
			okFuture.ws.ReConnect()
			okFuture.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				if isBin {
					msg, _ = GzipDecodeV3(msg)
				}
				//println(string(msg))
				if string(msg) == "pong" {
					okFuture.ws.UpdateActivedTime()
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
					if okFuture.wsLoginHandle != nil {
						okFuture.wsLoginHandle(err)
					}
					return
				}

				if len(data.Data) == 0 {
					return
				}

				switch data.Table  {
				case "swap/trade", "futures/trade":
				instrumentId, trades := okFuture.parseTrade(msg)
				if instrumentId != "" {
					topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
					okFuture.wsTradeHandleMap[topic](instrumentId, trades)
				}
				case "swap/depth", "futures/depth":
					depth := okFuture.parseDepth(msg)
					if depth != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, depth.InstrumentId)
						okFuture.wsDepthHandleMap[topic](depth)
					}
				case "index/ticker":
					instrumentId, tickers := okFuture.parseIndexTicker(msg)
					if len(tickers) > 0 {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okFuture.wsIndexTickerHandleMap[topic](instrumentId, tickers)
					}
				case "futures/position":
					instrumentId, positions := okFuture.parseFuturesPosition(msg)
					if positions != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okFuture.wsPositionHandleMap[topic](positions)
					}
				case "futures/account":
					account := okFuture.parseFuturesAccount(msg)
					if account != nil {
						okFuture.wsAccountHandleMap[data.Table](false, account)
					}
				case "futures/order":
					instrumentId, orders := okFuture.parseFuturesOrder(msg)
					if orders != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okFuture.wsOrderHandleMap[topic](orders)
					}
				case "swap/funding_rate":
					fundingRate := okFuture.parseFundingRate(msg)
					if fundingRate != nil {
						okFuture.wsFundingRateHandleMap[data.Table](*fundingRate)
					}
				case "swap/position":
					instrumentId, positions := okFuture.parseSwapPosition(msg)
					if positions != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okFuture.wsPositionHandleMap[topic](positions)
					}
				case "swap/account":
					account := okFuture.parseSwapAccount(msg)
					if account != nil {
						okFuture.wsAccountHandleMap[data.Table](false, account)
					}
				case "swap/order":
					instrumentId, orders := okFuture.parseSwapOrder(msg)
					if orders != nil {
						topic := fmt.Sprintf("%s:%s", data.Table, instrumentId)
						okFuture.wsOrderHandleMap[topic](orders)
					}
				}
			})
		}
	}
}

func (okFuture *OKExV3) isSwap(instrumentId string) bool {
	return strings.HasSuffix(instrumentId, "SWAP")
}

func (okFuture *OKExV3) getLoginData() interface{} {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET/users/self/verify"
	sign, _ := GetParamHmacSHA256Base64Sign(okFuture.apiSecretKey, message)

	return map[string]interface{}{
		"op":   "login",
		"args": []interface{}{okFuture.apiKey, okFuture.passphrase, timestamp, sign},
	}
}

func (okFuture *OKExV3) doLogin() error {
	ch := make(chan error)

	onDone := func(err error) {
		ch <- err
	}

	okFuture.wsLoginHandle = onDone

	data := okFuture.getLoginData()
	log.Printf("%+v", data)
	err := okFuture.ws.SendMessage(data)
	log.Printf("111111 %+v", err)
	if err != nil {
		return err
	}

	err = <- ch
	okFuture.wsLoginHandle = nil
	return err
}

func (okFuture *OKExV3) Login() error {
	okFuture.createWsConn()
	return okFuture.ws.Login(okFuture.doLogin)
}

func (okFuture *OKExV3) GetDepthWithWs(instrumentId string, handle func(*Depth)) error {
	okFuture.createWsConn()

	var channel string
	if okFuture.isSwap(instrumentId) {
		channel = fmt.Sprintf("swap/depth:%s", instrumentId)
	} else {
		channel = fmt.Sprintf("futures/depth:%s", instrumentId)
	}

	okFuture.wsDepthHandleMap[channel] = handle
	okFuture.depthManagers[instrumentId] = NewDepthManager()
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetTradeWithWs(instrumentId string, handle func(string, []Trade)) error {
	okFuture.createWsConn()

	var channel string
	if okFuture.isSwap(instrumentId) {
		channel = fmt.Sprintf("swap/trade:%s", instrumentId)
	} else {
		channel = fmt.Sprintf("futures/trade:%s", instrumentId)
	}

	okFuture.wsTradeHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetIndexTickerWithWs(instrumentId string, handle func(string, []Ticker)) error {
	okFuture.createWsConn()

	var channel string
	channel = fmt.Sprintf("index/ticker:%s", instrumentId)

	okFuture.wsIndexTickerHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetFundingRateWithWs(instrumentId string, handle func(SWAPFundingRate)) error {
	okFuture.createWsConn()

	var channel string
	channel = fmt.Sprintf("swap/funding_rate:%s", instrumentId)

	okFuture.wsFundingRateHandleMap["swap/funding_rate"] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetPositionWithWs(instrumentId string, handle func([]FuturePosition)) error {
	okFuture.createWsConn()

	var channel string
	if okFuture.isSwap(instrumentId) {
		channel = fmt.Sprintf("swap/position:%s", instrumentId)
	} else {
		channel = fmt.Sprintf("futures/position:%s", instrumentId)
	}

	okFuture.wsPositionHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetAccountWithWs(currency Currency, isSwap bool, handle func(bool, *FutureAccount)) error {
	okFuture.createWsConn()

	var channel string
	var key string
	if isSwap {
		channel = fmt.Sprintf("swap/account:%s-USD-SWAP", currency.Symbol)
		key = fmt.Sprintf("swap/account")
	} else {
		channel = fmt.Sprintf("futures/account:%s", currency.Symbol)
		key = fmt.Sprintf("futures/account")
	}

	okFuture.wsAccountHandleMap[key] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) GetOrderWithWs(instrumentId string, handle func([]FutureOrder)) error {
	okFuture.createWsConn()

	var channel string
	if okFuture.isSwap(instrumentId) {
		channel = fmt.Sprintf("swap/order:%s", instrumentId)
	} else {
		channel = fmt.Sprintf("futures/order:%s", instrumentId)
	}

	okFuture.wsOrderHandleMap[channel] = handle
	return okFuture.ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []interface{}{channel}})
}

func (okFuture *OKExV3) parseTrade(msg []byte) (string, []Trade) {
	var data *struct {
		Table  string
		Action string
		Data   []struct {
			InstrumentId string			`json:"instrument_id"`
			Price decimal.Decimal
			Side string
			Size decimal.Decimal
			Qty decimal.Decimal
			Timestamp string
			TradeId decimal.Decimal 	`json:"trade_id"`
		}
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]Trade, len(data.Data))
	for i := range data.Data {
		o := &data.Data[i]
		price, _ := o.Price.Float64()
		var amount float64
		if okFuture.isSwap(instrumentId) {
			amount, _ = o.Size.Float64()
		} else {
			amount, _ = o.Qty.Float64()
		}
		timestamp := V3ParseDate(o.Timestamp)
		ret[i] = Trade {
			Tid: o.TradeId.IntPart(),
			Type: o.Side,
			Amount: amount,
			Price: price,
			Date: timestamp,
		}
	}

	return instrumentId, ret
}

func (okFuture *OKExV3) parseDepth(msg []byte) *Depth {
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
	depthManager, _ := okFuture.depthManagers[instrumentId]
	if depthManager == nil {
		panic("Illegal state error")
	}

	asks, bids := depthManager.Update(data.Action, data.Data[0].Asks, data.Data[0].Bids)
	return &Depth{
		InstrumentId: instrumentId,
		Pair: CurrencyPair{Currency{Symbol: parts[0]}, Currency{Symbol: parts[1]}},
		UTime: time.Unix(timestamp / 1000, timestamp % 1000 * int64(time.Millisecond)),
		AskList: asks,
		BidList: bids,
	}
}

func (okFuture *OKExV3) parseIndexTicker(msg []byte) (string, []Ticker) {
	var data *struct{
		Data []struct {
			Last decimal.Decimal
			High24h decimal.Decimal	`json:"high_24h"`
			Low24h decimal.Decimal 	`json:"low_24h"`
			InstrumentId string 	`json:"instrument_id"`
			Open24h decimal.Decimal	`json:"open_24h"`
			Timestamp string
		}
	}
	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	var tickers []Ticker
	for _, r := range data.Data {
		last, _ := r.Last.Float64()
		high, _ := r.High24h.Float64()
		low, _ := r.Low24h.Float64()
		timestamp := V3ParseDate(r.Timestamp)
		tickers = append(tickers, Ticker{
			Last: last,
			High: high,
			Low: low,
			Date: uint64(timestamp),
		})
	}
	return instrumentId, tickers
}

func (okFuture *OKExV3) parseFuturesPosition(msg []byte) (string, []FuturePosition) {
	var data *struct {
		Table  string
		Action string
		Data   []V3Position
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]FuturePosition, len(data.Data))
	for i := range data.Data {
		ret[i] = *data.Data[i].ToFuturePosition()
	}

	return instrumentId, ret
}

func (okFuture *OKExV3) parseFuturesAccount(msg []byte) *FutureAccount {
	var data *struct {
		Table  string
		Action string
		Data   []map[string]V3CurrencyInfo
	}

	json.Unmarshal(msg, &data)

	account := new(FutureAccount)
	account.FutureSubAccounts = make(map[Currency]FutureSubAccount)

	for symbol, info := range data.Data[0] {
		currency := Currency{Symbol: symbol}
		account.FutureSubAccounts[currency] = *info.ToFutureSubAccount(currency)
	}

	return account
}

func (okFuture *OKExV3) parseFuturesOrder(msg []byte) (string, []FutureOrder) {
	var data *struct {
		Table  string
		Action string
		Data   []V3OrderInfo
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]FutureOrder, len(data.Data))
	for i := range data.Data {
		ret[i] = *data.Data[i].ToFutureOrder()
	}

	return instrumentId, ret
}

func (okFuture *OKExV3) parseSwapPosition(msg []byte) (string, []FuturePosition) {
	var data *struct {
		Table  string
		Action string
		Data   []V3_SWAPPosition
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]FuturePosition, len(data.Data))
	for i := range data.Data {
		ret[i] = *data.Data[i].ToFuturePosition()
	}

	return instrumentId, ret
}

func (okFuture *OKExV3) parseFundingRate(msg []byte) *SWAPFundingRate {
	var data *struct{
		Data []SWAPFundingRate
	}
	json.Unmarshal(msg, &data)
	return &data.Data[0]
}

func (okFuture *OKExV3) parseSwapAccount(msg []byte) *FutureAccount {
	var data *struct {
		Table  string
		Action string
		Data   []V3_SWAPCurrencyInfo
	}

	json.Unmarshal(msg, &data)

	account := new(FutureAccount)
	account.FutureSubAccounts = make(map[Currency]FutureSubAccount)

	for _, info := range data.Data {
		currency := V3SWAPInstrumentId2Currency(info.InstrumentId)
		account.FutureSubAccounts[currency] = *info.ToFutureSubAccount(currency)
	}

	return account
}

func (okFuture *OKExV3) parseSwapOrder(msg []byte) (string, []FutureOrder) {
	var data *struct {
		Table  string
		Action string
		Data   []V3_SWAPOrderInfo
	}

	json.Unmarshal(msg, &data)

	instrumentId := data.Data[0].InstrumentId

	ret := make([]FutureOrder, len(data.Data))
	for i := range data.Data {
		ret[i] = *data.Data[i].ToFutureOrder()
	}

	return instrumentId, ret
}

func (okFuture *OKExV3) CloseWs() {
	okFuture.ws.CloseWs()
}

func (this *OKExV3) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}

type DepthManager struct {
	buyMap map[string]DepthRecord
	sellMap map[string]DepthRecord
}

func NewDepthManager() *DepthManager {
	return &DepthManager{
		buyMap: make(map[string]DepthRecord),
		sellMap: make(map[string]DepthRecord),
	}
}

func (this *DepthManager) Update(action string, askList, bidList [][]decimal.Decimal) (DepthRecords, DepthRecords) {
	if action == "partial" {
		this.buyMap = make(map[string]DepthRecord)
		this.sellMap = make(map[string]DepthRecord)
	}

	for _, o := range askList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.sellMap, key)
		} else {
			price, _ := o[0].Float64()
			amount, _ := o[1].Float64()
			this.sellMap[key] = DepthRecord{Price: price, Amount: amount}
		}
	}

	for _, o := range bidList {
		key := o[0].String()
		if o[1].Equal(decimal.Zero) {
			delete(this.buyMap, key)
		} else {
			price, _ := o[0].Float64()
			amount, _ := o[1].Float64()
			this.buyMap[key] = DepthRecord{Price: price, Amount: amount}
		}
	}

	bids := make(DepthRecords, len(this.buyMap))
	i := 0
	for _, item := range this.buyMap {
		bids[i] = item
		i++
	}
	sort.SliceStable(bids, func(i,j int) bool {
		return bids[i].Price > bids[j].Price
	})

	asks := make(DepthRecords, len(this.sellMap))
	i = 0
	for _, item := range this.sellMap {
		asks[i] = item
		i++
	}
	sort.SliceStable(asks, func(i,j int) bool {
		return asks[i].Price < asks[j].Price
	})
	return asks, bids
}
