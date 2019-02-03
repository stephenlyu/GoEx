package okexadapter

import (
	"github.com/stephenlyu/tds/entity"
	"github.com/stephenlyu/GoEx"
	"fmt"
)

func FromSecurity(security *entity.Security) (pair goex.CurrencyPair, contractType string) {
	pair = goex.CurrencyPair{
		goex.Currency{security.Category, ""},
		goex.USD,
	}

	switch security.Code {
	case "QFUT":
		contractType = "quarter"
	case "TFUT":
		contractType = "this_week"
	case "NFUT":
		contractType = "next_week"
	case "FUT":
		contractType = "swap"
	default:
		panic("Unknown contract type")
	}
	return
}

func ToSecurity(pair goex.CurrencyPair, contractType string) *entity.Security {
	cat := pair.CurrencyA.Symbol
	var code string
	switch contractType {
	case "quarter":
		code = fmt.Sprintf("%sQFUT.OKEX", cat)
	case "this_week":
		code = fmt.Sprintf("%sTFUT.OKEX", cat)
	case "next_week":
		code = fmt.Sprintf("%sNFUT.OKEX", cat)
	case "swap":
		code = fmt.Sprintf("%sFUT.OKEX", cat)
	default:
		panic("Unknow contract type")
	}

	return entity.ParseSecurityUnsafe(code)
}
