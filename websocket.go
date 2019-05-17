package goex

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
	"io/ioutil"
	"sync/atomic"
)

type WsConn struct {
	*websocket.Conn
	url                      string
	heartbeatIntervalTime    time.Duration
	checkConnectIntervalTime time.Duration
	actived                  time.Time
	close                    chan int
	isClose                  bool
	subs                     []interface{}
	loginFunc                func() error

	errorCh                  chan error
	errorHandler             func(error)

	reconnecting 			 int32
}

const (
	SUB_TICKER      = 1 + iota
	SUB_ORDERBOOK
	SUB_KLINE_1M
	SUB_KLINE_15M
	SUB_KLINE_30M
	SUB_KLINE_1D
	UNSUB_TICKER
	UNSUB_ORDERBOOK
)

func NewWsConn(wsurl string) *WsConn {
	wsConn, resp, err := websocket.DefaultDialer.Dial(wsurl, nil)
	if err != nil {
		if resp != nil {
			log.Println(resp.Header)
			log.Println(resp.Status)
			bytes, _ := ioutil.ReadAll(resp.Body)
			log.Println(string(bytes))
		}
		panic(err)
	}
	return &WsConn{Conn: wsConn, url: wsurl, actived: time.Now(), checkConnectIntervalTime: 30 * time.Second, close: make(chan int, 1), errorCh: make(chan error)}
}

func (ws *WsConn) ReConnect() {

	tryReconnect := func() error {
		ws.Close()
		log.Println("start reconnect websocket:", ws.url)
		wsConn, _, err := websocket.DefaultDialer.Dial(ws.url, nil)
		if err != nil {
			log.Println("reconnect fail ???")
			return err
		} else {
			ws.Conn = wsConn
			ws.actived = time.Now()

			if ws.loginFunc != nil {
				err := ws.doLogin()
				if err != nil {
					log.Printf("login fail, error: %+v", err)
					return err
				}
			}

			//re subscribe
			for _, sub := range ws.subs {
				log.Println("subscribe:", sub)
				err := ws.WriteJSON(sub)
				if err != nil {
					log.Printf("subscribe %s fail, error: %+v", sub, err)
					return err
				}
			}
		}
		return nil
	}

	doReconnect := func() {
		if !atomic.CompareAndSwapInt32(&ws.reconnecting, 0, 1) {
			return
		}

		for {
			err := tryReconnect()
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		atomic.StoreInt32(&ws.reconnecting, 0)
	}

	var errTimes int

	timer := time.NewTimer(ws.checkConnectIntervalTime)
	go func() {
		for {
			select {
			case <-timer.C:
				if time.Now().Sub(ws.actived) >= ws.checkConnectIntervalTime + 5 * time.Second {
					go doReconnect()
					errTimes = 0
				}
				timer.Reset(ws.checkConnectIntervalTime)
			case err := <- ws.errorCh:
				if err == nil {
					errTimes = 0
				} else {
					if  ws.errorHandler != nil {
						ws.errorHandler(err)
					}
					errTimes++
					if errTimes > 10 {
						go doReconnect()
						errTimes = 0
					}
				}
				timer.Reset(ws.checkConnectIntervalTime)
			case <-ws.close:
				timer.Stop()
				log.Println("close websocket connect, exiting reconnect goroutine.")
				return
			}
		}
	}()
}

func (ws *WsConn) Heartbeat(heartbeat func() interface{}, interval time.Duration) {
	ws.heartbeatIntervalTime = interval
	ws.checkConnectIntervalTime = ws.heartbeatIntervalTime

	timer := time.NewTimer(interval)
	go func() {
		for {
			select {
			case <-timer.C:
				data := heartbeat()
				var err error
				if _, ok := data.(string); ok {
					err = ws.WriteMessage(websocket.TextMessage, []byte(data.(string)))
				} else {
					err = ws.WriteJSON(data)
				}
				ws.errorCh <- err
				if err != nil {
					log.Println("heartbeat error , ", err)
					time.Sleep(time.Second)
				}
				timer.Reset(interval)
			case <-ws.close:
				timer.Stop()
				log.Println("close websocket connect , exiting heartbeat goroutine.")
				return
			}
		}
	}()
}

func (ws *WsConn) Subscribe(subEvent interface{}) error {
	err := ws.WriteJSON(subEvent)
	if err != nil {
		return err
	}
	ws.subs = append(ws.subs, subEvent)
	return nil
}

func (ws *WsConn) SendMessage(data interface{}) error {
	return ws.WriteJSON(data)
}

func (ws *WsConn) doLogin() error {
	return ws.loginFunc()
}

func (ws *WsConn) Login(f func () error) error {
	ws.loginFunc = f
	return ws.doLogin()
}

func (ws *WsConn) ReceiveMessage(handle func(msg []byte)) {
	go func() {
		for {
			t, msg, err := ws.ReadMessage()
			if !ws.isReconnecting() {
				ws.errorCh <- err
			} else if err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}
			if err != nil {
				log.Println(err)
				if ws.isClose {
					log.Println("exiting receive message goroutine.")
					break
				}
				time.Sleep(time.Second)
				continue
			}
			switch t {
			case websocket.TextMessage, websocket.BinaryMessage:
				handle(msg)
			case websocket.PongMessage:
				ws.actived = time.Now()
			case websocket.CloseMessage:
				ws.CloseWs()
				return
			default:
				log.Println("error websocket message type , content is :\n", string(msg))
			}
		}
	}()
}

func (ws *WsConn) ReceiveMessageEx(handle func(isBin bool, msg []byte)) {
	go func() {
		for {
			t, msg, err := ws.ReadMessage()
			if !ws.isReconnecting() {
				ws.errorCh <- err
			} else if err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}
			if err != nil {
				log.Println(err)
				if ws.isClose {
					log.Println("exiting receive message goroutine.")
					break
				}
				time.Sleep(time.Second)
				continue
			}
			switch t {
			case websocket.TextMessage:
				handle(false, msg)
			case websocket.BinaryMessage:
				handle(true, msg)
			case websocket.PongMessage:
				ws.actived = time.Now()
			case websocket.CloseMessage:
				ws.CloseWs()
				return
			default:
				log.Println("error websocket message type , content is :\n", string(msg))
			}
		}
	}()
}

func (ws *WsConn) UpdateActivedTime() {
	ws.actived = time.Now()
}

func (ws *WsConn) CloseWs() {
	ws.close <- 1 //exit reconnect goroutine
	if ws.heartbeatIntervalTime > 0 {
		ws.close <- 1 //exit heartbeat goroutine
	}

	err := ws.Close()
	if err != nil {
		log.Println("close websocket connect error , ", err)
	}

	ws.isClose = true
}

func (ws *WsConn) SetErrorHandler(handler func (error)) {
	ws.errorHandler = handler
}

func (ws *WsConn) isReconnecting() bool {
	return atomic.LoadInt32(&ws.reconnecting) == 1
}