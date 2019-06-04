package ceohk

import (
	. "github.com/stephenlyu/GoEx"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
)

const (
	API_BASE_URL    = "https://ceohk.bi"
	TICKER 	  		= "/api/market/ticker?market=%s"
)

type CEOHK struct {
	client            *http.Client
}

func NewCEOHK() *CEOHK {
	this := new(CEOHK)
	this.client = http.DefaultClient
	return this
}

func (ok *CEOHK) GetTicker(market string) (*TickerDecimal, error) {
	url := API_BASE_URL + TICKER
	resp, err := ok.client.Get(fmt.Sprintf(url, market))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Code decimal.Decimal
		Data struct {
			Buy decimal.Decimal
			Sell decimal.Decimal
			Last decimal.Decimal
			Vol decimal.Decimal
			High decimal.Decimal
			Low decimal.Decimal
			Time decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.Code.IntPart() != 1000 {
		return nil, fmt.Errorf("error code: %s", data.Code.String())
	}

	ticker := new(TickerDecimal)
	ticker.Date = uint64(data.Data.Time.IntPart()) * 1000
	ticker.Buy = data.Data.Buy
	ticker.Sell = data.Data.Sell
	ticker.Last = data.Data.Last
	ticker.High = data.Data.High
	ticker.Low = data.Data.Low
	ticker.Vol = data.Data.Vol

	return ticker, nil
}
