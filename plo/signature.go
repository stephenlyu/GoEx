package plo

import (
	"fmt"
	"github.com/stephenlyu/GoEx"
)

func BuildSignature(apiKey, secret string, ts uint64, data string) (string, string) {
	message := fmt.Sprintf("accessKey=%s&data=%s&ts=%d", apiKey, data, ts)
	ret, _ := goex.GetParamHmacSHA256Sign(secret, message)
	return message, ret
}

func BuildWsSignature(apiKey, secret string, ts uint64) string {
	message := fmt.Sprintf("accessKey=%s&ts=%d", apiKey, ts)
	ret, _ := goex.GetParamHmacSHA256Sign(secret, message)
	return ret
}
