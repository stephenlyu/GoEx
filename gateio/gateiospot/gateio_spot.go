package gateiospot

import (
	"sync"
	"net/http"
	. "github.com/stephenlyu/GoEx"
)

type GateIOSpot struct {
	apiKey,
	apiSecretKey string
	client            *http.Client

	ws                *WsConn
	createWsLock      sync.Mutex
	wsLoginHandle func(err error)
	wsDepthHandleMap  map[string]func(*DepthDecimal)
	wsTradeHandleMap map[string]func(CurrencyPair, []TradeDecimal)
	wsAccountHandleMap  map[string]func(*SubAccountDecimal)
	wsOrderHandleMap  map[string]func([]OrderDecimal)
	depthManagers	 map[string]*DepthManager
}

func NewGateIOSpot(	apiKey, apiSecretKey string) *GateIOSpot {
	return &GateIOSpot{
		apiKey: apiKey,
		apiSecretKey: apiSecretKey,
		client: http.DefaultClient,
	}
}
