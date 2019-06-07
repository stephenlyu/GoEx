package zbg

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
	"time"
	"strings"
	"strconv"
	"sort"
)

const (
	ORDER_TYPE_SELL = iota
	ORDER_TYPE_BUY
)

const (
	API_BASE_URL    = "https://www.zbg.com"
	KLINE_API_BASE_URL    = "https://kline.zbg.com"
	MARKET_LIST 	= "/exchange/config/controller/website/marketcontroller/getByWebId"
	CURRENCY_LIST = "/exchange/config/controller/website/currencycontroller/getCurrencyList"
	ACCOUNT = "/exchange/fund/controller/website/fundcontroller/findbypage"
	ADD_ENTRUST = "/exchange/entrust/controller/website/EntrustController/addEntrust"
	CANCEL_ENTRUST = "/exchange/entrust/controller/website/EntrustController/cancelEntrust"
	QUERY_PENDING_ORDERS = "/exchange/entrust/controller/website/EntrustController/getUserEntrustRecordFromCache?marketId=%s"
	QUERY_ORDER = "/exchange/entrust/controller/website/EntrustController/getEntrustById?marketId=%s&entrustId=%s"
	TICKER = "/api/data/v1/ticker?marketName=%s"
	DEPTH = "/api/data/v1/entrusts?marketName=%s&dataSize=%d"
	TRADES = "/api/data/v1/trades?marketName=%s&dataSize=%d"
)

type ZBG struct {
	ApiId string
	SecretKey string
	client *http.Client

	currencyInfoMap map[string]CurrencyInfo
	marketMap map[string]Market

}

func NewZBG(ApiId string, SecretKey string) *ZBG {
	this := new(ZBG)
	this.ApiId = ApiId
	this.SecretKey = SecretKey
	this.client = http.DefaultClient
	this.currencyInfoMap = make(map[string]CurrencyInfo)
	this.marketMap = make(map[string]Market)
	return this
}

func (this *ZBG) getCurrencyNameById(id string) string {
	c, ok := this.currencyInfoMap[id]
	if ok {
		return c.Name
	}

	var err error
	var l []CurrencyInfo
	for i := 0; i < 5; i++ {
		l, err = this.GetCurrencyList()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}

	for _, o := range l {
		this.currencyInfoMap[o.CurrencyId] = o
	}
	c, ok = this.currencyInfoMap[id]
	if !ok {
		return ""
	}
	return c.Name
}

func (this *ZBG) getMarketIdByName(name string) string {
	name = strings.ToUpper(name)
	c, ok := this.marketMap[name]
	if ok {
		return c.MarketId
	}

	var err error
	var l []Market
	for i := 0; i < 5; i++ {
		l, err = this.GetMarketList()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}

	for _, o := range l {
		this.marketMap[strings.ToUpper(o.Name)] = o
	}
	c, ok = this.marketMap[name]
	if !ok {
		return ""
	}
	return c.MarketId
}

func (ok *ZBG) GetMarketList() ([]Market, error) {
	url := API_BASE_URL + MARKET_LIST
	resp, err := ok.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Datas []Market
		ResMsg struct {
			Message string
			Code decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", data.ResMsg.Code.String())
	}

	return data.Datas, nil
}

func (ok *ZBG) GetCurrencyList() ([]CurrencyInfo, error) {
	url := API_BASE_URL + CURRENCY_LIST
	resp, err := ok.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		Datas []CurrencyInfo
		ResMsg struct {
		    Message string
		    Code decimal.Decimal
 	    }
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", data.ResMsg.Code.String())
	}

	return data.Datas, nil
}

func (ok *ZBG) GetTicker(market string) (*TickerDecimal, error) {
	url := KLINE_API_BASE_URL + TICKER
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
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
	    }
		Datas []string
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", data.ResMsg.Code.String())
	}

	r := data.Datas

	ticker := new(TickerDecimal)
	ticker.Date = uint64(time.Now().UnixNano() / int64(time.Millisecond))
	ticker.Buy, _ = decimal.NewFromString(r[7])
	ticker.Sell, _ = decimal.NewFromString(r[8])
	ticker.Last, _ = decimal.NewFromString(r[1])
	ticker.High, _ = decimal.NewFromString(r[2])
	ticker.Low, _ = decimal.NewFromString(r[3])
	ticker.Vol, _ = decimal.NewFromString(r[4])

	return ticker, nil
}

func (ok *ZBG) GetDepth(market string, dataSize int) (*DepthDecimal, error) {
	if dataSize == 0 {
		dataSize = 5
	}
	market = strings.ToUpper(market)
	url := fmt.Sprintf(KLINE_API_BASE_URL + DEPTH, market, dataSize)
	resp, err := ok.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
	    }
		Datas struct {
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal
		}
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", data.ResMsg.Code.String())
	}

	r := data.Datas

	depth := new(DepthDecimal)
	depth.Pair = NewCurrencyPair2(market)

	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	return depth, nil
}

func (ok *ZBG) GetTrades(market string, dataSize int) ([]TradeDecimal, error) {
	if dataSize == 0 {
		dataSize = 1
	}
	market = strings.ToUpper(market)
	url := fmt.Sprintf(KLINE_API_BASE_URL + TRADES, market, dataSize)
	resp, err := ok.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var data struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
  	    }
		Datas [][]string
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if data.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", data.ResMsg.Code.String())
	}

	var trades = make([]TradeDecimal, len(data.Datas))

	for i, o := range data.Datas {
		t := &trades[i]
		t.Tid, _ = strconv.ParseInt(o[2], 10, 64)
		t.Amount, _ = decimal.NewFromString(o[6])
		t.Price, _ = decimal.NewFromString(o[5])
		if o[4] == "bid" {
			t.Type = "buy"
		} else {
			t.Type = "sell"
		}
		t.Date = t.Tid * 1000
	}

	return trades, nil
}

func (this *ZBG) signData(data string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond), 10)
	message := this.ApiId + timestamp + data + this.SecretKey
	sign, _ := GetParamMD5Sign(this.SecretKey, message)

	return map[string]string {
		"Apiid": this.ApiId,
		"Timestamp": timestamp,
		"sign": sign,
	}
}

func (this *ZBG) signGet(param map[string]string) map[string]string {
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + param[k])
	}
	data := strings.Join(parts, "")

	return this.signData(data)
}

func (this *ZBG) GetAccount(page int, pageSize int) ([]SubAccountDecimal, error) {
	params := map[string]string {}
	if page > 0 {
		params["pageNum"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		params["pageSize"] = strconv.Itoa(pageSize)
	}

	bytes, _ := json.Marshal(params)
	data := string(bytes)

	header := this.signData(data)

	url := API_BASE_URL + ACCOUNT
	body, err := HttpPostJson(this.client, url, data, header)

	if err != nil {
		return nil, err
	}

	var resp struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
	    }
		Datas struct {
			List []struct {
				CurrencyTypeId decimal.Decimal
				Amount decimal.Decimal
				Freeze decimal.Decimal
			}
		}
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", resp.ResMsg.Code.String())
	}

	var ret []SubAccountDecimal
	for _, o := range resp.Datas.List {
		currency := this.getCurrencyNameById(o.CurrencyTypeId.String())
		if currency == "" {
			continue
		}
		ret = append(ret, SubAccountDecimal{
			Currency: Currency{Symbol: strings.ToUpper(currency)},
			AvailableAmount: o.Amount,
			FrozenAmount: o.Freeze,
			Amount: o.Amount.Add(o.Freeze),
		})
	}

	return ret, nil
}

func (this *ZBG) PlaceOrder(amount decimal.Decimal, _type int, marketName string, price decimal.Decimal) (string, error) {
	marketId := this.getMarketIdByName(marketName)
	params := map[string]interface{} {
		"amount": amount,
		"rangeType": 0,
		"type": _type,
		"marketId": marketId,
		"price": price,

	}
	bytes, _ := json.Marshal(params)
	data := string(bytes)

	header := this.signData(data)

	url := API_BASE_URL + ADD_ENTRUST
	body, err := HttpPostJson(this.client, url, data, header)

	if err != nil {
		return "", err
	}

	var resp struct {
		ResMsg struct {
		    Message string
		    Code decimal.Decimal
 	    }
		Datas struct {
		   EntrustId string
	   }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}

	if resp.ResMsg.Code.IntPart() != 1 {
		return "", fmt.Errorf("error code: %s", resp.ResMsg.Code.String())
	}

	return resp.Datas.EntrustId, nil
}

func (this *ZBG) CancelOrder(marketName string, entrustId string) error {
	marketId := this.getMarketIdByName(marketName)
	params := map[string]interface{} {
		"marketId": marketId,
		"entrustId": entrustId,

	}
	bytes, _ := json.Marshal(params)
	data := string(bytes)

	header := this.signData(data)

	url := API_BASE_URL + CANCEL_ENTRUST
	body, err := HttpPostJson(this.client, url, data, header)

	if err != nil {
		return err
	}

	var resp struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
 	    }
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.ResMsg.Code.IntPart() != 1 {
		return fmt.Errorf("error code: %s", resp.ResMsg.Code.String())
	}

	return nil
}

func (this *ZBG) QueryPendingOrders(marketName string) ([]OrderDecimal, error) {
	marketName = strings.ToUpper(marketName)
	marketId := this.getMarketIdByName(marketName)
	if marketId == "" {
		return nil, fmt.Errorf("unknown market %s", marketName)
	}
	header := this.signGet(map[string]string{"marketId": marketId})

	url := fmt.Sprintf(API_BASE_URL + QUERY_PENDING_ORDERS, marketId)

	var resp struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
	    }
		Datas []OrderInfo
	}

	err := HttpGet4(this.client, url, header, &resp)
	if err != nil {
		return nil, err
	}

	if resp.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", resp.ResMsg.Code.String())
	}

	var ret = make([]OrderDecimal, len(resp.Datas))
	for i := range resp.Datas {
		ret[i] = *resp.Datas[i].ToOrderDecimal(marketName)
	}

	return ret, nil
}

func (this *ZBG) QueryOrder(marketName string, entrustId string) (*OrderDecimal, error) {
	marketName = strings.ToUpper(marketName)
	marketId := this.getMarketIdByName(marketName)
	if marketId == "" {
		return nil, fmt.Errorf("unknown market %s", marketName)
	}
	header := this.signGet(map[string]string{
		"marketId": marketId,
		"entrustId": entrustId,
	})

	url := fmt.Sprintf(API_BASE_URL + QUERY_ORDER, marketId, entrustId)

	var resp struct {
		ResMsg struct {
		   Message string
		   Code decimal.Decimal
	   }
		Datas *OrderInfo
	}

	err := HttpGet4(this.client, url, header, &resp)
	if err != nil {
		return nil, err
	}

	if resp.ResMsg.Code.IntPart() != 1 {
		return nil, fmt.Errorf("error code: %s", resp.ResMsg.Code.String())
	}

	if resp.Datas == nil {
		return nil, nil
	}

	return resp.Datas.ToOrderDecimal(marketName), nil
}
