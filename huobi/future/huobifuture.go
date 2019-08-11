package huobifuture

import (
	"net/http"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"github.com/shopspring/decimal"
	. "github.com/stephenlyu/GoEx"
	"sort"
	"net/url"
	"sync"
	"log"
	"errors"
)

const (
	SIDE_BUY = 0
	SIDE_SELL = 1
)

const (
	HOST = "api.hbdm.com"
	API_BASE_URL = "https://" + HOST
	CONTRACT_INFO = "/api/v1/contract_contract_info"
	TICKER = "/market/detail/merged"
	DEPTH = "/market/depth"
	TRADE = "/market/trade"
	ACCOUNTS = "/api/v1/contract_account_info"
	POSITIONS = "/api/v1/contract_position_info"
	PLACE_ORDER = "/api/v1/contract_order"
	BATCH_PLACE_ORDERS = "/api/v1/contract_batchorder"
	BATCH_CANCEL = "/api/v1/contract_cancel"
	CANCEL_ALL = "/api/v1/contract_cancelall"
	OPEN_ORDERS = "/api/v1/contract_openorders"
	HIS_ORDERS = "/api/v1/contract_hisorders"
	QUERY_ORDER = "/api/v1/contract_order_info"
)

type HuobiFuture struct {
	ApiKey             string
	SecretKey          string
	client             *http.Client

	symbols            map[string]*ContractInfo

	publicWs           *WsConn
	createPublicWsLock sync.Mutex
	wsDepthHandleMap   map[string]func(*DepthDecimal)
	wsTradeHandleMap   map[string]func(string, []TradeDecimal)
	errorHandle        func(error)

	privateWs           *WsConn
	createPrivateWsLock sync.Mutex
	wsLoginHandle      func(err error)
	wsOrderHandle      func([]FutureOrderDecimal)
	privateErrorHandle func(error)

	lock               sync.Mutex
}

func NewHuobiFuture(client *http.Client, ApiKey, SecretKey string) *HuobiFuture {
	this := new(HuobiFuture)
	this.ApiKey = ApiKey
	this.SecretKey = SecretKey
	this.client = client

	return this
}

func (this *HuobiFuture) signData(data string) string {
	sign, _ := GetParamHmacSHA256Base64Sign(this.SecretKey, data)

	return sign
}

func (this *HuobiFuture) getTimestamp() string {
	const (
		DATE_FORMAT = "2006-01-02T15:04:05"
	)
	return time.Now().In(time.UTC).Format(DATE_FORMAT)
}

func (this *HuobiFuture) sign(method, reqUrl string, param map[string]string) string {
	param["AccessKeyId"] = this.ApiKey
	param["SignatureMethod"] = "HmacSHA256"
	param["SignatureVersion"] = "2"
	param["Timestamp"] = this.getTimestamp()
	var keys []string
	for k := range param {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i,j int) bool {
		return keys[i] < keys[j]
	})

	var parts []string
	for _, k := range keys {
		parts = append(parts, k + "=" + url.QueryEscape(param[k]))
	}
	data := strings.Join(parts, "&")

	lines := []string {
		method,
		HOST,
		reqUrl,
		data,
	}

	message := strings.Join(lines, "\n")
	sign := this.signData(message)
	return data + "&Signature=" + url.QueryEscape(sign)
}

func (this *HuobiFuture) buildQueryString(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k + "=" + url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

func (this *HuobiFuture) GetContractInfo() ([]ContractInfo, error) {
	url := API_BASE_URL + CONTRACT_INFO
	var resp struct {
		Status string
		Msg string
		Data []ContractInfo
	}

	err := HttpGet4(this.client, url, nil, &resp)

	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		log.Printf("HuobiFuture.GetContractInfo error status: %s\n", resp.Status)
		return nil, fmt.Errorf("error_code: %s", resp.Status)
	}

	return resp.Data, nil
}

func (this *HuobiFuture) GetTicker(symbol string) (*TickerDecimal, error) {
	params := map[string]string {
		"symbol": symbol,
	}
	url := API_BASE_URL + TICKER + "?" + this.buildQueryString(params)
	var resp struct {
		Status string
		Msg string
		Tick struct {
			Vol decimal.Decimal
			Ask []decimal.Decimal
			Bid []decimal.Decimal
			Close decimal.Decimal
			Count decimal.Decimal
			High decimal.Decimal
			Low decimal.Decimal
			Open decimal.Decimal
			Amount decimal.Decimal
			 }
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		log.Printf("HuobiFuture.GetTicker error code: %s\n", resp.Status)
		return nil, fmt.Errorf("error_code: %s", resp.Status)
	}

	r := &resp.Tick

	ticker := new(TickerDecimal)
	ticker.Date = uint64(time.Now().UnixNano()/1000000)
	ticker.Open = r.Open
	ticker.Last = r.Close
	ticker.High = r.High
	ticker.Low = r.Low
	ticker.Vol = r.Vol
	ticker.Buy = r.Bid[0]
	ticker.Sell = r.Ask[0]

	return ticker, nil
}

func (this *HuobiFuture) GetDepth(symbol string) (*DepthDecimal, error) {
	params := map[string]string {
		"symbol": symbol,
		"type": "step0",
	}
	url := API_BASE_URL + DEPTH + "?" + this.buildQueryString(params)
	var resp struct {
		Status string
		Tick struct {
			Ts int64
			Asks [][]decimal.Decimal
			Bids [][]decimal.Decimal
		}
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		log.Printf("HuobiFuture.GetDepth error code: %s\n", resp.Status)
		return nil, fmt.Errorf("error_code: %s", resp.Status)
	}

	r := resp.Tick

	depth := new(DepthDecimal)

	depth.AskList = make([]DepthRecordDecimal, len(r.Asks), len(r.Asks))
	for i, o := range r.Asks {
		depth.AskList[i] = DepthRecordDecimal{Price: o[0], Amount: o[1]}
	}

	depth.BidList = make([]DepthRecordDecimal, len(r.Bids), len(r.Bids))
	for i, o := range r.Bids {
		depth.BidList[i] = DepthRecordDecimal{Price: o[0], Amount: o[0]}
	}

	return depth, nil
}

func (this *HuobiFuture) GetTrades(symbol string) ([]TradeDecimal, error) {
	params := map[string]string {
		"symbol": symbol,
	}
	url := API_BASE_URL + TRADE + "?" + this.buildQueryString(params)
	var resp struct {
		Status string
		Tick struct {
				 Data []struct {
					 Amount decimal.Decimal
					 Price decimal.Decimal
					 Direction string
					 Id int64
					 Ts int64
				 }
			 }
	}

	err := HttpGet4(this.client, url, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "ok" {
		log.Printf("HuobiFuture.GetTrade error code: %s\n", resp.Status)
		return nil, fmt.Errorf("error_code: %s", resp.Status)
	}

	var trades = make([]TradeDecimal, len(resp.Tick.Data))

	for i, o := range resp.Tick.Data {
		t := &trades[i]
		t.Amount = o.Amount
		t.Price = o.Price
		t.Type = o.Direction
		t.Date = o.Ts
		t.Tid = o.Id
	}

	return trades, nil
}

func (this *HuobiFuture) GetAccounts() (*FutureAccountDecimal, error) {
	params := map[string]string {}
	queryString := this.sign("POST", ACCOUNTS, params)

	reqUrl := API_BASE_URL + ACCOUNTS + "?" + queryString
	postData := map[string]interface{} {}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data []struct {
			Symbol string
			MarginBalance decimal.Decimal		`json:"margin_balance"`
			MarginFrozen decimal.Decimal		`json:"margin_frozen"`
			MarginAvailable decimal.Decimal 	`json:"margin_available"`
			ProfitReal decimal.Decimal			`json:"profit_real"`
			ProfitUnreal decimal.Decimal		`json:"profit_unreal"`
			RiskRate decimal.Decimal			`json:"risk_rate"`
		}
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.GetAccounts error code: %d\n", data.ErrCode)
		return nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	var ret *FutureAccountDecimal
	ret = new(FutureAccountDecimal)
	ret.FutureSubAccounts = make(map[Currency]FutureSubAccountDecimal)
	for _, r := range data.Data {
		curency := NewCurrency(r.Symbol, "")
		ret.FutureSubAccounts[curency] = FutureSubAccountDecimal {
			Currency: curency,
			AccountRights: r.MarginBalance,
			ProfitReal: r.ProfitReal,
			ProfitUnreal: r.ProfitUnreal,
			RiskRate: r.RiskRate,
		}
	}

	return ret, nil
}

func (this *HuobiFuture) GetPosition(symbol string) ([]PositionInfo, error) {
	params := map[string]string {}
	queryString := this.sign("POST", POSITIONS, params)

	reqUrl := API_BASE_URL + POSITIONS + "?" + queryString
	postData := map[string]interface{} {
		"symbol": strings.ToUpper(symbol),
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data []PositionInfo
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.GetPosition error code: %d\n", data.ErrCode)
		return nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	return data.Data, nil
}

func (this *HuobiFuture) PlaceOrder(req OrderReq) (string, error) {
	params := map[string]string {}
	queryString := this.sign("POST", PLACE_ORDER, params)

	reqUrl := API_BASE_URL + PLACE_ORDER + "?" + queryString
	bytes, err := HttpPostForm4(this.client, reqUrl, req, nil)
	if err != nil {
		return "", err
	}
	println(string(bytes))
	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data struct {
				 OrderId decimal.Decimal 		`json:"order_id"`
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return "", err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.PlaceOrder error code: %d\n", data.ErrCode)
		return "", fmt.Errorf("error_code: %d", data.ErrCode)
	}

	return data.Data.OrderId.String(), nil
}

func (this *HuobiFuture) PlaceOrders(reqList []OrderReq) ([]string, []error, error) {
	params := map[string]string {}
	queryString := this.sign("POST", BATCH_PLACE_ORDERS, params)

	reqUrl := API_BASE_URL + BATCH_PLACE_ORDERS + "?" + queryString
	postData := map[string]interface{} {
		"orders_data": reqList,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, nil, err
	}
	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data struct {
				 Errors []struct {
					 Index int
					 ErrCode int 		`json:"err_code"`
					 ErrMsg string 		`json:"err_msg"`
				 }
				 Success []struct {
					 Index int
					 OrderId decimal.Decimal 	`json:"order_id"`
				 }
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.PlaceOrders error code: %d\n", data.ErrCode)
		return nil, nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	var orderIds = make([]string, len(reqList))
	var errorList = make([]error, len(reqList))
	for _, r := range data.Data.Errors {
		log.Printf("HuobiFuture.PlaceOrders error code: %d\n", r.ErrCode)
		errorList[r.Index-1] = fmt.Errorf("error_code: %d", r.ErrCode)
	}

	for _, r := range data.Data.Success {
		orderIds[r.Index-1] = r.OrderId.String()
	}

	return orderIds, errorList, nil
}

func (this *HuobiFuture) BatchCancelOrders(symbol string, orderIds []string) (error, []error) {
	var errorList =  make([]error, len(orderIds))

	params := map[string]string {}
	queryString := this.sign("POST", BATCH_CANCEL, params)

	reqUrl := API_BASE_URL + BATCH_CANCEL + "?" + queryString
	postData := map[string]interface{} {
		"order_id": strings.Join(orderIds, ","),
		"symbol": symbol,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return err, errorList
	}

	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data struct {
				 Errors []struct {
					 OrderId   string
					 ErrCode int			`json:"err_code"`
				 }
				 Successes string
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err, errorList
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.BatchCancelOrders error code: %d\n", data.ErrCode)
		return fmt.Errorf("error_code: %d", data.ErrCode), errorList
	}

	orderIdMap := make(map[string]int)
	for i, orderId := range orderIds {
		orderIdMap[orderId] = i
	}

	for _, r := range data.Data.Errors {
		if r.ErrCode == 1071 {
			continue
		}
		log.Printf("HuobiFuture.BatchCancelOrders error code: %d\n", r.ErrCode)
		errorList[orderIdMap[r.OrderId]] = fmt.Errorf("error_code: %d", r.ErrCode)
	}

	return nil, errorList
}

func (this *HuobiFuture) QueryPendingOrders(symbol string, page, pageSize int) ([]FutureOrderDecimal, error) {
	if pageSize == 0 {
		pageSize = 100
	}

	params := map[string]string {}
	queryString := this.sign("POST", OPEN_ORDERS, params)

	reqUrl := API_BASE_URL + OPEN_ORDERS+ "?" + queryString
	postData := map[string]interface{} {
		"symbol": symbol,
		"page_index": page,
		"page_size": pageSize,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data struct {
				 Orders []OrderInfo
			 }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.QueryPendingOrders error code: %d", data.ErrCode)
		return nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	var ret = make([]FutureOrderDecimal, len(data.Data.Orders))
	for i := range data.Data.Orders {
		ret[i] = *data.Data.Orders[i].ToOrderDecimal()
	}

	return ret, nil
}

func (this *HuobiFuture) QueryHisOrders(symbol string, page, pageSize int) ([]FutureOrderDecimal, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 50
	}

	params := map[string]string {}
	queryString := this.sign("POST", HIS_ORDERS, params)

	reqUrl := API_BASE_URL + HIS_ORDERS + "?" + queryString
	postData := map[string]interface{} {
		"symbol": symbol,
		"trade_type": 0,
		"type": 2,
		"status": 0,
		"create_date": 90,
		"page_index": page,
		"page_size": pageSize,
	}
	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data struct {
				   Orders []OrderInfo
			   }
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.QueryHisOrders error code: %d\n", data.ErrCode)
		return nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	var ret = make([]FutureOrderDecimal, len(data.Data.Orders))
	for i := range data.Data.Orders {
		ret[i] = *data.Data.Orders[i].ToOrderDecimal()
	}

	return ret, nil
}

func (this *HuobiFuture) QueryOrder(symbol string, orderId, clientOid string) (*FutureOrderDecimal, error) {
	params := map[string]string {}
	queryString := this.sign("POST", QUERY_ORDER, params)

	reqUrl := API_BASE_URL + QUERY_ORDER + "?" + queryString
	postData := map[string]interface{} {
		"symbol": symbol,
	}
	if orderId != "" {
		postData["order_id"] = orderId
	} else if clientOid != "" {
		postData["client_order_id"] = clientOid
	} else {
		return nil, errors.New("bad parameter")
	}

	bytes, err := HttpPostForm4(this.client, reqUrl, postData, nil)
	if err != nil {
		return nil, err
	}
	var data struct {
		Status string
		ErrCode int 			`json:"err_code"`
		Data []OrderInfo
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	if data.Status != "ok" {
		log.Printf("HuobiFuture.QueryOrder error code: %d\n", data.ErrCode)
		return nil, fmt.Errorf("error_code: %d", data.ErrCode)
	}

	if len(data.Data) == 0 {
		return nil, nil
	}

	return data.Data[0].ToOrderDecimal(), nil
}
