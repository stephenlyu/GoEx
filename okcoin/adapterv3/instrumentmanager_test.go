package okexadapterv3

import (
	"testing"
	"github.com/stephenlyu/tds/util"
	"fmt"
	"github.com/stephenlyu/tds/date"
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
	mgr := NewInstrumentManager()
	for _, code := range []string {"EOSQFUT.OKEX", "EOSNFUT.OKEX", "EOSTFUT.OKEX"} {
		instrumentId, err := mgr.GetInstrumentId(code)
		if err != nil {
			panic(err)
		}
		fmt.Println(code, instrumentId)
	}
}
