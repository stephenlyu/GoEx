package bitmex

import (
	"fmt"
	"github.com/stephenlyu/GoEx"
)

func BuildSignature(secret string, method string, path string, expires int64, data string) string {
	message := fmt.Sprintf("%s%s%d%s", method, path, expires, data)
	ret, _ := goex.GetParamHmacSHA256Sign(secret, message)
	return ret
}

func BuildWsSignature(secret string, path string, expires int64) string {
	message := fmt.Sprintf("GET%s%d", path, expires)
	ret, _ := goex.GetParamHmacSHA256Sign(secret, message)
	return ret
}
