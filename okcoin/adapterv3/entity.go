package okexadapterv3

import (
	"github.com/stephenlyu/tds/entity"
	"fmt"
	"strings"
	"github.com/z-ray/log"
)

func FromSecurity(security *entity.Security) string {
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
		panic("Unknown contract type")
	}
	return ""
}

func ToSecurity(instrumentId string) *entity.Security {
	var code string
	if strings.HasSuffix(instrumentId, "SWAP") {
		parts := strings.Split(instrumentId, "-")
		code = fmt.Sprintf("%sFUT.OKEX", parts[0])
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
