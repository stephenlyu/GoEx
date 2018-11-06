package bitmexadapter

import (
	"github.com/stephenlyu/tds/entity"
	"github.com/stephenlyu/GoEx"
	"fmt"
)

func FromSecurity(security *entity.Security) (pair goex.CurrencyPair) {
	currency := goex.Currency{security.Category, ""}
	if currency == goex.BTC {
		currency = goex.XBT
	}
	pair = goex.CurrencyPair{
		currency,
		goex.USD,
	}

	return
}

func ToSecurity(pair goex.CurrencyPair) *entity.Security {
	cat := pair.CurrencyA.Symbol
	if cat == "XBT" {
		cat = "BTC"
	}
	code := fmt.Sprintf("%sFUT.BITMEX", cat)
	return entity.ParseSecurityUnsafe(code)
}
