package okexadapterv3

import (
	"github.com/stephenlyu/tds/entity"
	"fmt"
	"strings"
	"github.com/z-ray/log"
	"regexp"
	"github.com/stephenlyu/GoEx"
)

var CODE_PATTERN, _ = regexp.Compile("[0-9]+")
var SUFFIX_PATTERN, _ = regexp.Compile("[QTN]?FUT[A-Z]+")

func FromSecurity(security *entity.Security) string {
	if security.IsIndex() {
		return fmt.Sprintf("%s-USD", security.GetCategory())
	}

	switch security.Code {
	case "QFUT", "TFUT", "NFUT":
		instrumentId, err := DEFAULT_INSTRUMENT_MANAGER.GetInstrumentId(security.String())
		if err != nil {
			panic(err)
		}
		return instrumentId
	case "FUT":
		return fmt.Sprintf("%s-USD-SWAP", security.Category)
	default:
		if SUFFIX_PATTERN.Match([]byte(security.Code)) {
			instrumentId, err := DEFAULT_INSTRUMENT_MANAGER.GetInstrumentId(security.String())
			if err != nil {
				panic(err)
			}
			return instrumentId
		}

		if CODE_PATTERN.Match([]byte(security.Code)) {
			return fmt.Sprintf("%s-USD-%s", security.Category, security.GetCode())
		}

		panic("Unknown contract type")
	}
	return ""
}

func ToSecurity(instrumentId string) *entity.Security {
	var code string
	if strings.HasSuffix(instrumentId, "SWAP") {
		parts := strings.Split(instrumentId, "-")
		code = fmt.Sprintf("%sFUT.OKEX", parts[0])
	} else if strings.HasSuffix(instrumentId, "USD") {
		parts := strings.Split(instrumentId, "-")
		code = fmt.Sprintf("%sINDEX.OKEX", parts[0])
	} else {
		var err error
		code, err = DEFAULT_INSTRUMENT_MANAGER.GetCode(instrumentId)
		if err != nil {
			log.Printf("error: %+v", err)
			panic(err)
		}
	}
	return entity.ParseSecurityUnsafe(code)
}

func InstrumentId2CurrencyPair(instrumentId string) goex.CurrencyPair {
	parts := strings.Split(instrumentId, "-")
	return goex.CurrencyPair{
		goex.Currency{Symbol: parts[0]},
		goex.Currency{Symbol: parts[1]},
	}
}