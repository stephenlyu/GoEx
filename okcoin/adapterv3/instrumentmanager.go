package okexadapterv3

import (
	"github.com/stephenlyu/GoEx/okcoin"
	"net/http"
	"github.com/stephenlyu/tds/util"
	"fmt"
	"strings"
	"sync"
	"time"
	"github.com/stephenlyu/tds/entity"
)

//
// 负责将Security映射为对应的InstumentId
//

const WEEK_MILLS = 7 * 24 * 3600 * 1000

type InstrumentManager struct {
	api *okcoin.OKExV3

	lock sync.RWMutex
	nextSyncTimestamp uint64
	codeInstrumentIdMap map[string]string
	instrumentIdCodeMap map[string]string
}

func NewInstrumentManager() *InstrumentManager {
	return &InstrumentManager{
		api: okcoin.NewOKExV3(http.DefaultClient, "", "", ""),
		codeInstrumentIdMap: make(map[string]string),
	}
}

func (this *InstrumentManager) getNextSyncTimestamp() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.nextSyncTimestamp
}

func (this *InstrumentManager) ensureMap() error {
	now := util.Tick()
	if now < this.getNextSyncTimestamp() {
		return nil
	}

	var err error
	var instruments []okcoin.V3Instrument

	for i := 0; i < 5; i++ {
		instruments, err = this.api.GetInstruments()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}

	// 建立Currency -> InstrumentIds映射
	currencyInstruments := make(map[string][]okcoin.V3Instrument)
	for _, ins := range instruments {
		parts := strings.Split(ins.InstrumentId, "-")
		currency := parts[0]
		currencyInstruments[currency] = append(currencyInstruments[currency], ins)
	}

	m := make(map[string]string)

	var missingCode bool
	for currency, instruments := range currencyInstruments {
		if len(instruments) != 3 && len(instruments) != 6 {
			missingCode = true
		}

		var code string
		for _, ins := range instruments {
			var suffix string
			if strings.Contains(ins.InstrumentId, "USDT") {
				suffix = "USDT"
			}
			switch ins.Alias {
			case "quarter":
				code = fmt.Sprintf("%sQFUT%s.OKEX", currency, suffix)
			case "this_week":
				code = fmt.Sprintf("%sTFUT%s.OKEX", currency, suffix)
			case "next_week":
				code = fmt.Sprintf("%sNFUT%s.OKEX", currency, suffix)
			}
			m[code] = ins.InstrumentId
		}
	}
	this.lock.Lock()
	var changed = false
	for k, v := range m {
		if this.codeInstrumentIdMap[k] != v {
			this.codeInstrumentIdMap[k] = v
			changed = true
			break
		}
	}
	if changed {
		this.codeInstrumentIdMap = m
		this.instrumentIdCodeMap = make(map[string]string)
		for c, i := range this.codeInstrumentIdMap {
			this.instrumentIdCodeMap[i] = c
		}
		// 周合约交割后，会出现一个只有两个合约的阶段，这个阶段，需要持续更新，直到新的合约产生
		if !missingCode {
			this.nextSyncTimestamp = NextSyncTimestamp(util.Tick())
		}
	}
	this.lock.Unlock()
	return nil
}

func (this *InstrumentManager) GetInstrumentId(code string) (string, error) {
	err := this.ensureMap()
	if err != nil {
		return "", err
	}

	instrumentId, ok := this.codeInstrumentIdMap[code]
	if !ok {
		security := entity.ParseSecurityUnsafe(code)
		code := security.GetCode()
		var suffix string
		if len(code) > 4 {
			suffix = code[4:]
		}
		if strings.HasPrefix(code, "QFUT") {
			instrumentId, ok = this.codeInstrumentIdMap[fmt.Sprintf("%sNFUT%s.OKEX", security.GetCategory(), suffix)]
		} else if strings.HasPrefix(code, "NFUT") {
			instrumentId, ok = this.codeInstrumentIdMap[fmt.Sprintf("%sTFUT%s.OKEX", security.GetCategory(), suffix)]
		}
	}
	if !ok {
		return "", fmt.Errorf("No instrument id for %s", code)
	}
	return instrumentId, nil
}

func (this *InstrumentManager) GetCode(instrumentId string) (string, error) {
	err := this.ensureMap()
	if err != nil {
		return "", err
	}

	code, ok := this.instrumentIdCodeMap[instrumentId]
	if !ok {
		return "", fmt.Errorf("No code for %s", instrumentId)
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

var DEFAULT_INSTRUMENT_MANAGER = NewInstrumentManager()
