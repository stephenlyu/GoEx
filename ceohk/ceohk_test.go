package ceohk

import (
	"testing"
	"encoding/json"
	"fmt"
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func output(v interface{}) {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bytes))
}

func TestCEOHK_GetTicker(t *testing.T) {
	api := NewCEOHK()
	ticker, err := api.GetTicker("sht_qc")
	chk(err)
	output(ticker)
}
