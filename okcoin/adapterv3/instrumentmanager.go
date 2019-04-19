package okexadapterv3

import (
	"github.com/stephenlyu/GoEx/okcoin"
	"net/http"
	"github.com/stephenlyu/tds/util"
	"fmt"
	"strings"
	"sort"
	"sync"
	"github.com/stephenlyu/tds/date"
)

//
// 负责将Security映射为对应的InstumentId
//

const WEEK_MILLS = 7 * 24 * 3600 * 1000

type InstrumentManager struct {
	api *okcoin.OKExV3

	lock sync.RWMutex
	nextSyncTimestamp uint64
	tryTimes int
	codeInstrumentIdMap map[string]string
	instrumentIdCodeMap map[string]string
}

func NewInstrumentManager() *InstrumentManager {
	return &InstrumentManager{
		api: okcoin.NewOKExV3(http.DefaultClient, "", "", ""),
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

	instruments, err := this.api.GetInstruments()
	if err != nil {
		return err
	}

	// 建立Currency -> InstrumentIds映射
	currencyInstruments := make(map[string][]string)
	for _, ins := range instruments {
		parts := strings.Split(ins.InstrumentId, "-")
		currency := parts[0]
		currencyInstruments[currency] = append(currencyInstruments[currency], ins.InstrumentId)
	}

	m := make(map[string]string)
	getDate := func(instrumentId string) string {
		return strings.Split(instrumentId, "-")[2]
	}
	var missingCode bool
	for currency, ids := range currencyInstruments {
		sort.SliceStable(ids, func(i,j int) bool {
			return ids[i] < ids[j]
		})
		if len(ids) == 3 {
			m[currency + "TFUT.OKEX"] = ids[0]
			m[currency + "NFUT.OKEX"] = ids[1]
			m[currency + "QFUT.OKEX"] = ids[2]
		} else if len(ids) == 2 {
			now := date.GetNowString()
			yearPrefix := now[:2]

			d1, _ := date.DayString2Timestamp(yearPrefix + getDate(ids[0]))
			d2, _ := date.DayString2Timestamp(yearPrefix + getDate(ids[1]))
			if d2 - d1 == WEEK_MILLS {
				m[currency + "TFUT.OKEX"] = ids[0]
				m[currency + "NFUT.OKEX"] = ids[1]
				m[currency + "QFUT.OKEX"] = ""
			} else {
				m[currency + "TFUT.OKEX"] = ids[0]
				m[currency + "NFUT.OKEX"] = ""
				m[currency + "QFUT.OKEX"] = ids[1]
			}
			missingCode = true
		}
	}

	rm := make(map[string]string)
	for c, i := range m {
		rm[i] = c
	}

	this.lock.Lock()
	var changed = false
	for k, v := range m {
		if this.codeInstrumentIdMap[k] != v {
			changed = true
			break
		}
	}
	this.tryTimes++
	if changed || this.tryTimes > 60 {
		this.codeInstrumentIdMap = m
		this.instrumentIdCodeMap = rm
		// 周合约交割后，会出现一个只有两个合约的阶段，这个阶段，需要持续更新，直到新的合约产生
		if !missingCode {
			this.nextSyncTimestamp = NextSyncTimestamp(util.Tick())
			this.tryTimes = 0
		}
	} else {
		this.tryTimes++
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
