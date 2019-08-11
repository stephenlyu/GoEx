package huobifuture

import (
	"encoding/json"
	. "github.com/stephenlyu/GoEx"
	"log"
	"fmt"
	"strings"
	"github.com/pborman/uuid"
	"sort"
	"net/url"
	"github.com/shopspring/decimal"
)

func (this *HuobiFuture) createPrivateWsConn() {
	if this.privateWs == nil {
		//connect wsx
		this.createPrivateWsLock.Lock()
		defer this.createPrivateWsLock.Unlock()

		if this.privateWs == nil {
			this.privateWs = NewWsConn("wss://api.hbdm.com/notification")
			this.privateWs.SetErrorHandler(this.privateErrorHandle)
			this.privateWs.ReConnect()
			this.privateWs.ReceiveMessageEx(func(isBin bool, msg []byte) {
				msg, err := GzipDecode(msg)
				if err != nil {
					fmt.Println(err)
					return
				}
				//println(string(msg))

				var data struct {
					Op string
					Ts decimal.Decimal
					ErrCode int 			`json:"err_code"`
					Topic string
				}
				err = json.Unmarshal(msg, &data)
				if err != nil {
					log.Print(err)
					return
				}

				switch data.Op {
				case "ping":
					this.privateWs.UpdateActivedTime()
					this.privateWs.SendMessage(map[string]interface{}{"op": "pong", "ts": data.Ts})
				case "auth":
					var err error
					if data.ErrCode != 0 {
						err = fmt.Errorf("error_code: %d", data.ErrCode)
					}
					this.wsLoginHandle(err)
				case "notify":
					switch {
					case strings.HasPrefix(data.Topic, "orders."):
						order := this.parseOrder(msg)
						if order != nil {
							this.wsOrderHandle([]FutureOrderDecimal{*order})
						}
					}
				}
			})
		}
	}
}

func (this *HuobiFuture) loginSign() (param map[string]string) {
	param = make(map[string]string)
	param["AccessKeyId"] = this.ApiKey
	param["SignatureMethod"] = "HmacSHA256"
	param["SignatureVersion"] = "2"
	param["Timestamp"] = this.getTimestamp()
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + "=" + url.QueryEscape(param[k]))
	}
	data := strings.Join(parts, "&")

	lines := []string {
		"GET",
		HOST,
		"/notification",
		data,
	}

	message := strings.Join(lines, "\n")
	param["Signature"] = this.signData(message)
	return
}

func (this *HuobiFuture) getLoginData() interface{} {
	param := this.loginSign()
	param["op"] = "auth"
	param["type"] = "api"
	param["cid"] = uuid.New()
	return param
}

func (this *HuobiFuture) doLogin() error {
	ch := make(chan error)

	onDone := func(err error) {
		ch <- err
	}

	this.wsLoginHandle = onDone

	data := this.getLoginData()
	err := this.privateWs.SendMessage(data)
	if err != nil {
		return err
	}

	err = <- ch
	this.wsLoginHandle = nil
	return err
}

func (this *HuobiFuture) Login() error {
	this.createPrivateWsConn()
	return this.privateWs.Login(this.doLogin)
}

func (this *HuobiFuture) GetOrderWithWs(symbol string,
	orderHandle func([]FutureOrderDecimal)) error {
	symbol = strings.ToLower(symbol)

	this.createPrivateWsConn()

	this.wsOrderHandle = orderHandle
	topic := fmt.Sprintf("orders.%s", symbol)

	event := map[string]interface{}{
		"op":   "sub",
		"cid": uuid.New(),
		"topic": topic,
	}
	fmt.Printf("%+v\n", event)
	return this.privateWs.Subscribe(event)
}

func (this *HuobiFuture) parseOrder(msg []byte) *FutureOrderDecimal {
	var order *OrderInfo
	err := json.Unmarshal(msg, &order)
	if err != nil || order == nil {
		return nil
	}
	return order.ToOrderDecimal()
}

func (this *HuobiFuture) ClosePrivateWs() {
	this.privateWs.CloseWs()
}

func (this *HuobiFuture) SetPrivateErrorHandler(handle func(error)) {
	this.privateErrorHandle = handle
}
