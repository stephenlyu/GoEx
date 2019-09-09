package fameex

import (
	"encoding/json"
	. "github.com/stephenlyu/GoEx"
	"log"
	"time"
	"github.com/shopspring/decimal"
	"errors"
	"github.com/nntaoli-project/GoEx"
	"fmt"
	"strings"
)

func (this *Fameex) createWsConn() {
	if this.ws == nil {
		//connect wsx
		this.createWsLock.Lock()
		defer this.createWsLock.Unlock()

		if this.ws == nil {
			this.wsDepthHandleMap = make(map[string]func(*DepthDecimal))
			this.wsTradeHandleMap = make(map[string]func(string, []TradeDecimal))

			this.ws = NewWsConn("wss://test.fameex.com/push")
			this.ws.SetErrorHandler(this.errorHandle)
			this.ws.Heartbeat(func() interface{} {
				return map[string]string{
					"op": "heartBeat",
				}
			}, 20*time.Second)
			this.ws.ReConnect()
			this.ws.ReceiveMessageEx(func(isBin bool, msg []byte) {
				//println(string(msg))

				var data struct {
					Type string
					Code int
				}
				err := json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				switch data.Type {
				case "heartBeat":
					this.ws.UpdateActivedTime()
				case "login":
					var err error
					if data.Code != 200 {
						err = errors.New("Login failure")
					}
					if this.wsLoginHandle != nil {
						this.wsLoginHandle(err)
					}
				case "transDepth":
					depth := this.parseDepth(msg)
					if depth != nil {
						symbol := depth.Pair.ToSymbol("_")
						this.wsDepthHandleMap[symbol](depth)
					}
				case "lastTrade":
					symbol, trades := this.parseTrade(msg)
					this.wsTradeHandleMap[symbol](symbol, trades)
				case "myTransDepth":
					if this.wsOrderHandle != nil {
						this.parseOrder(msg)
					}
				}
			})
		}
	}
}

func (this *Fameex) getLoginData() interface{} {
	return map[string]interface{}{
		"op":   "login",
		"AccessKey": this.ApiKey,
		"sign": this.UserId,
	}
}

func (this *Fameex) doLogin() error {
	ch := make(chan error)

	onDone := func(err error) {
		ch <- err
	}

	this.wsLoginHandle = onDone

	data := this.getLoginData()
	err := this.ws.SendMessage(data)
	if err != nil {
		return err
	}

	err = <- ch
	this.wsLoginHandle = nil
	return err
}

func (this *Fameex) Login() error {
	this.createWsConn()
	return this.ws.Login(this.doLogin)
}

func (this *Fameex) GetDepthWithWs(symbol string,
	depthHandle func(*DepthDecimal),
	tradesHandle func(string, []TradeDecimal),
	orderHandle func([]OrderDecimal)) error {
	err, precision := this.getPrecision(symbol)
	if err != nil {
		return err
	}

	this.createWsConn()

	pair := goex.NewCurrencyPair2(symbol)

	this.wsDepthHandleMap[symbol] = depthHandle
	this.wsTradeHandleMap[symbol] = tradesHandle
	this.wsOrderHandle = orderHandle
	event := map[string]interface{}{
		"op":   "register",
		"type": "transDepth",
		"base": strings.ToLower(pair.CurrencyA.Symbol),
		"quote": strings.ToLower(pair.CurrencyB.Symbol),
		"percision": precision,
	}
	return this.ws.Subscribe(event)
}

func (this *Fameex) parseTrade(msg []byte) (string, []TradeDecimal) {
	var data *struct {
		Data struct {
				 Base    string
				 Quote   string
				 Count   decimal.Decimal
				 Price   decimal.Decimal
				 Time    int64
				 BuyType int
	   }
	}

	json.Unmarshal(msg, &data)

	r := &data.Data

	symbol := strings.ToUpper(fmt.Sprintf("%s_%s", r.Base, r.Quote))

	var side string
	if r.BuyType == SIDE_BUY {
		side = "buy"
	} else {
		side = "sell"
	}

	return symbol, []TradeDecimal {
		{
			Price: r.Price,
			Amount: r.Count,
			Type: side,
			Date: r.Time / 1000000,
		},
	}
}

func (this *Fameex) parseDepth(msg []byte) *DepthDecimal {
	type Item struct {
		Price decimal.Decimal
		Count decimal.Decimal
	}

	var data *struct {
		Data struct {
				 Base     string
				 Quote    string
				 SellList []Item
				 BuyList  []Item
		}
	}

	json.Unmarshal(msg, &data)

	r := &data.Data
	if r.Base == "" || r.Quote == "" {
		return nil
	}

	depth := new(DepthDecimal)
	
	depth.Pair = NewCurrencyPair2(fmt.Sprintf("%s_%s", r.Base, r.Quote))
	depth.AskList = make([]DepthRecordDecimal, len(r.SellList), len(r.SellList))
	for i, o := range r.SellList {
		depth.AskList[i] = DepthRecordDecimal{Price: o.Price, Amount: o.Count}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.BuyList), len(r.BuyList))
	for i, o := range r.BuyList {
		depth.BidList[i] = DepthRecordDecimal{Price: o.Price, Amount: o.Count}
	}

	return depth
}

func (this *Fameex) parseOrder(msg []byte) {
	if this.wsOrderHandle == nil {
		return
	}
	var data *struct {
		Data struct {
				Base string
				Quote string
				OrderId string
			 }
	}

	json.Unmarshal(msg, &data)

	if data.Data.OrderId == "" {
		return
	}
	symbol := strings.ToUpper(fmt.Sprintf("%s_%s", data.Data.Base, data.Data.Quote))
	go func(symbol, orderId string) {
		var err error
		var order *OrderDecimal
		for i := 0; i < 10; i++ {
			order, err = this.QueryOrder(symbol, orderId)
			if err == nil {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
		if err != nil {
			this.errorHandle(err)
		} else if order != nil {
			this.wsOrderHandle([]OrderDecimal{*order})
		}
	} (symbol, data.Data.OrderId)
}

func (this *Fameex) CloseWs() {
	this.ws.CloseWs()
}

func (this *Fameex) SetErrorHandler(handle func(error)) {
	this.errorHandle = handle
}
