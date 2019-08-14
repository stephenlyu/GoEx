package huobiadapter

import (
	"github.com/stephenlyu/tds/entity"
	"fmt"
	"strings"
	"github.com/z-ray/log"
	"regexp"
)

var CODE_PATTERN, _ = regexp.Compile("[0-9]+")

// 获取交易代码
func ContractCodeFromSecurity(security *entity.Security) string {
	switch security.Code {
	case "QFUT", "TFUT", "NFUT":
		contractCode, err := DEFAULT_INSTRUMENT_MANAGER.GetContractCode(SymbolFromSecurity(security))
		if err != nil {
			panic(err)
		}
		return contractCode
	default:
		if CODE_PATTERN.Match([]byte(security.Code)) {
			return fmt.Sprintf("%s%s", security.Category, security.GetCode())
		}

		panic("Unknown contract type")
	}
	return ""
}

func SymbolFromSecurity(security *entity.Security) string {
	switch security.Code {
	case "QFUT":
		return fmt.Sprintf("%s_CQ", security.GetCategory())
	case "TFUT":
		return fmt.Sprintf("%s_CW", security.GetCategory())
	case "NFUT":
		return fmt.Sprintf("%s_NW", security.GetCategory())
	default:
		symbol, err := DEFAULT_INSTRUMENT_MANAGER.GetSymbol(security.GetCategory() + security.GetCode())
		if err != nil {
			log.Printf("error: %+v", err)
			panic(err)
		}
		if symbol != "" {
			return symbol
		}

		panic("Unknown security")
	}
	return ""
}

func ContractCodeToSecurity(contractCode string) *entity.Security {
	symbol, err := DEFAULT_INSTRUMENT_MANAGER.GetSymbol(contractCode)
	if err != nil {
		log.Printf("error: %+v", err)
		panic(err)
	}
	parts := strings.Split(symbol, "_")
	var code string
	switch parts[1] {
	case "CQ":
		code = fmt.Sprintf("%sQFUT.HUOBI", parts[0])
	case "CW":
		code = fmt.Sprintf("%sTFUT.HUOBI", parts[0])
	case "NW":
		code = fmt.Sprintf("%sNFUT.HUOBI", parts[0])
	}

	return entity.ParseSecurityUnsafe(code)
}
