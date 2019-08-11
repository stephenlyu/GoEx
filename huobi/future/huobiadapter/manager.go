package huobiadapter

import (
	"net/http"
	"github.com/stephenlyu/tds/util"
	"fmt"
	"strings"
	"sync"
	"time"
	"github.com/stephenlyu/GoEx/huobi/future"
)

//
// 负责将Security映射为对应的ContractCode
//



const WEEK_MILLS = 7 * 24 * 3600 * 1000

type Manager struct {
	api                   *huobifuture.HuobiFuture

	lock                  sync.RWMutex
	nextSyncTimestamp     uint64
	symbolContractCodeMap map[string]string
	contractCodeSymbolMap map[string]string
}

func NewManager() *Manager {
	return &Manager{
		api: huobifuture.NewHuobiFuture(http.DefaultClient, "", ""),
		symbolContractCodeMap: make(map[string]string),
	}
}

func (this *Manager) getNextSyncTimestamp() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.nextSyncTimestamp
}

func (this *Manager) ensureMap() error {
	now := util.Tick()
	if now < this.getNextSyncTimestamp() {
		return nil
	}

	var err error
	var contracts []huobifuture.ContractInfo

	for i := 0; i < 5; i++ {
		contracts, err = this.api.GetContractInfo()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}

	// 建立Currency -> InstrumentIds映射
	currencyContracts := make(map[string][]huobifuture.ContractInfo)
	for _, ins := range contracts {
		currency := ins.Symbol
		currencyContracts[currency] = append(currencyContracts[currency], ins)
	}

	m := make(map[string]string)

	var missingCode bool
	for currency, contracts := range currencyContracts {
		if len(contracts) != 3 {
			missingCode = true
		}

		for _, ins := range contracts {
			switch ins.ContractType {
			case "quarter":
				m[currency + "_CQ"] = ins.ContractCode
			case "this_week":
				m[currency + "_CW"] = ins.ContractCode
			case "next_week":
				m[currency + "_NW"] = ins.ContractCode
			}
		}
	}

	this.lock.Lock()
	var changed = false
	for k, v := range m {
		if this.symbolContractCodeMap[k] != v {
			this.symbolContractCodeMap[k] = v
			changed = true
			break
		}
	}
	if changed {
		this.symbolContractCodeMap = m
		this.contractCodeSymbolMap = make(map[string]string)
		for c, i := range this.symbolContractCodeMap {
			this.contractCodeSymbolMap[i] = c
		}
		// 周合约交割后，会出现一个只有两个合约的阶段，这个阶段，需要持续更新，直到新的合约产生
		if !missingCode {
			this.nextSyncTimestamp = NextSyncTimestamp(util.Tick())
		}
	}
	this.lock.Unlock()
	return nil
}

func (this *Manager) GetContractCode(symbol string) (string, error) {
	err := this.ensureMap()
	if err != nil {
		return "", err
	}

	contractCode, ok := this.symbolContractCodeMap[symbol]
	if !ok {
		parts := strings.Split(symbol, "_")
		if parts[1] == "CQ" {
			contractCode, ok = this.symbolContractCodeMap[fmt.Sprintf("%s_NW", parts[0])]
		} else if parts[1] == "NW" {
			contractCode, ok = this.symbolContractCodeMap[fmt.Sprintf("%s_TW", parts[0])]
		}
	}
	if !ok {
		return "", fmt.Errorf("No contract code for %s", symbol)
	}
	return contractCode, nil
}

func (this *Manager) GetSymbol(contractCode string) (string, error) {
	err := this.ensureMap()
	if err != nil {
		return "", err
	}

	code, ok := this.contractCodeSymbolMap[contractCode]
	if !ok {
		return "", fmt.Errorf("No code for %s", contractCode)
	}
	return code, nil
}

func NextSyncTimestamp(now uint64) uint64 {
	const (
		HOUR_MILLIS = 3600 * 1000
		DAY_MILLIS = 24 * HOUR_MILLIS
	)
	now += 8 * HOUR_MILLIS		// 东八区+8

	passedDayMillis := now % DAY_MILLIS
	dayStartTs := now / DAY_MILLIS * DAY_MILLIS
	if passedDayMillis > 16 * HOUR_MILLIS {
		return dayStartTs + (24 + 16 - 8) * HOUR_MILLIS
	}
	return dayStartTs + (16 - 8) * HOUR_MILLIS + 5 * 1000	// 延迟5秒
}

var DEFAULT_INSTRUMENT_MANAGER = NewManager()
