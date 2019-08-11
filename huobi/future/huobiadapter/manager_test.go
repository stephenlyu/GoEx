package huobiadapter

import (
	"testing"
	"github.com/stephenlyu/tds/util"
	"fmt"
	"github.com/stephenlyu/tds/date"
	"time"
	"github.com/stephenlyu/tds/entity"
)

func TestNextSyncTimestamp(t *testing.T) {
	var ts uint64
	ts = NextSyncTimestamp(util.Tick())
	fmt.Println(date.Timestamp2SecondString(ts))

	ts, _ = date.SecondString2Timestamp("20181206 14:00:00")
	ts = NextSyncTimestamp(ts)
	fmt.Println(date.Timestamp2SecondString(ts))

	ts, _ = date.SecondString2Timestamp("20181206 16:00:00")
	ts = NextSyncTimestamp(ts)
	fmt.Println(date.Timestamp2SecondString(ts))

	ts, _ = date.SecondString2Timestamp("20181206 16:00:01")
	ts = NextSyncTimestamp(ts)
	fmt.Println(date.Timestamp2SecondString(ts))
}

func TestInstrumentManager_GetInstrumentId(t *testing.T) {
	mgr := NewManager()
	for _, code := range []string {"EOS_CQ", "EOS_CW", "EOS_NW"} {
		instrumentId, err := mgr.GetContractCode(code)
		if err != nil {
			panic(err)
		}
		code1, _ := mgr.GetSymbol(instrumentId)
		fmt.Println(code, instrumentId, code1)
	}
}

func TestNewInstrumentManager(t *testing.T) {
	mgr := NewManager()
	for {
		instrumentId, err := mgr.GetContractCode("EOS_NW")
		if err != nil {
			panic(err)
		}
		fmt.Println(instrumentId)
		time.Sleep(time.Minute)
	}
}

func TestContractCodeFromSecurity(t *testing.T) {
	for _, code := range []string {"EOSQFUT.HUOBI", "EOSTFUT.HUOBI", "EOSNFUT.HUOBI", "EOS190927.HUOBI", "EOS190816.HUOBI", "EOS190823.HUOBI"} {
		security := entity.ParseSecurityUnsafe(code)
		contractCode := ContractCodeFromSecurity(security)
		fmt.Println(code, contractCode, SymbolFromSecurity(security), ContractCodeToSecurity(contractCode).String())
	}
}