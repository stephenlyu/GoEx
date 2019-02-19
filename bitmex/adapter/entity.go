package bitmexadapter

import (
	"github.com/stephenlyu/tds/entity"
	"fmt"
)

func FromSecurity(security *entity.Security) string {
	cat := security.GetCategory()

	if cat == "BTC" {
		cat = "XBT"
	}

	var code string

	if security.GetCode() == "FUT" {
		code = "USD"
	} else {
		code = security.GetCode()
	}

	return cat + code
}

func ToSecurity(symbol string) *entity.Security {
	cat := symbol[:3]
	code := symbol[3:]

	if code == "USD" {
		code = "FUT"
	}

	if cat == "XBT" {
		cat = "BTC"
	}
	return entity.ParseSecurityUnsafe(fmt.Sprintf("%s%s.BITMEX", cat, code))
}
