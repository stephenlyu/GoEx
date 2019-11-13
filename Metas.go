package goex

import (
	"net/http"
	"time"
	"github.com/shopspring/decimal"
)

type Order struct {
	Price,
	Amount,
	AvgPrice,
	DealAmount,
	Fee float64
	OrderID2  string
	OrderID   int
	OrderTime int
	Status    TradeStatus
	Currency  CurrencyPair
	Side      TradeSide
}

type OrderDecimal struct {
	Price,
	Amount,
	AvgPrice,
	DealAmount,
	Notinal,
	DealNotional,
	Fee decimal.Decimal
	FeeCurrency string
	OrderID2  string
	OrderID   int
	ClientOid string
	OrderTime int
	Timestamp int64
	Status    TradeStatus
	Currency  CurrencyPair
	Side      TradeSide
}

type Trade struct {
	Tid    int64   `json:"tid"`
	Type   string  `json:"type"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
	Date   int64   `json:"date_ms"`
	Time   string
}

type TradeDecimal struct {
	Tid    int64   `json:"tid"`
	Type   string  `json:"type"`
	Amount decimal.Decimal `json:"amount"`
	Price  decimal.Decimal `json:"price"`
	Date   int64   `json:"date_ms"`
	Time   string	`json:"-"`
}

type SubAccount struct {
	Currency Currency
	Amount,
	ForzenAmount,
	LoanAmount float64
}

type Account struct {
	Exchange    string
	Asset       float64 //总资产
	NetAsset    float64 //净资产
	SubAccounts map[Currency]SubAccount
}

type SubAccountDecimal struct {
	Currency Currency
	Amount,
	FrozenAmount,
	AvailableAmount,
	LoanAmount decimal.Decimal
}

type AccountDecimal struct {
	Exchange    string
	Asset       decimal.Decimal //总资产
	NetAsset    decimal.Decimal //净资产
	SubAccounts map[Currency]SubAccountDecimal
}

type Ticker struct {
	ContractType string       `json:"omitempty"`
	Pair         CurrencyPair `json:"omitempty"`
	Last         float64      `json:"last"`
	Buy          float64      `json:"buy"`
	Sell         float64      `json:"sell"`
	High         float64      `json:"high"`
	Low          float64      `json:"low"`
	Vol          float64      `json:"vol"`
	Date         uint64       `json:"date"` // 单位:秒(second)
	ContractId   int64        `json:"omitempty"`
}

type TickerDecimal struct {
	ContractType string       		`json:"omitempty"`
	Pair         CurrencyPair 		`json:"omitempty"`
	Last         decimal.Decimal    `json:"last"`
	Buy          decimal.Decimal    `json:"buy"`
	Sell         decimal.Decimal    `json:"sell"`
	Open         decimal.Decimal    `json:"open"`
	High         decimal.Decimal    `json:"high"`
	Low          decimal.Decimal    `json:"low"`
	Vol          decimal.Decimal    `json:"vol"`
	Date         uint64       		`json:"date"` // 单位:秒(second)
	ContractId   int64        		`json:"omitempty"`
}

type DepthRecord struct {
	Price,
	Amount float64
}

type DepthRecords []DepthRecord

func (dr DepthRecords) Len() int {
	return len(dr)
}

func (dr DepthRecords) Swap(i, j int) {
	dr[i], dr[j] = dr[j], dr[i]
}

func (dr DepthRecords) Less(i, j int) bool {
	return dr[i].Price < dr[j].Price
}

type Depth struct {
	ContractType string //for future
	InstrumentId string
	Symbol string
	Pair         CurrencyPair
	UTime        time.Time
	AskList,
	BidList DepthRecords
}

type DepthRecordDecimal struct {
	Price decimal.Decimal		`json:"price"`
	Amount decimal.Decimal		`json:"amount"`
}

type DepthRecordsDecimal []DepthRecordDecimal

func (dr DepthRecordsDecimal) Len() int {
	return len(dr)
}

func (dr DepthRecordsDecimal) Swap(i, j int) {
	dr[i], dr[j] = dr[j], dr[i]
}

func (dr DepthRecordsDecimal) Less(i, j int) bool {
	return dr[i].Price.LessThan(dr[j].Price)
}

type DepthDecimal struct {
	ContractType string //for future
	InstrumentId string
	Pair         CurrencyPair
	UTime        time.Time
	AskList,
	BidList DepthRecordsDecimal
}

type APIConfig struct {
	HttpClient *http.Client
	ApiUrl,
	AccessKey,
	SecretKey string
}

type Kline struct {
	Pair      CurrencyPair
	Timestamp int64
	Open,
	Close,
	High,
	Low,
	Vol float64
}

type FutureKline struct {
	*Kline
	Vol2 float64 //个数
}

type FutureSubAccount struct {
	Currency      Currency
	AccountRights float64 //账户权益
	KeepDeposit   float64 //保证金
	ProfitReal    float64 //已实现盈亏
	ProfitUnreal  float64
	RiskRate      float64 //保证金率
}

type FutureAccount struct {
	FutureSubAccounts map[Currency]FutureSubAccount
}

type FutureSubAccountDecimal struct {
	Currency      Currency
	AccountRights decimal.Decimal
	KeepDeposit   decimal.Decimal
	ProfitReal    decimal.Decimal
	ProfitUnreal  decimal.Decimal
	RiskRate      decimal.Decimal
}

type FutureAccountDecimal struct {
	FutureSubAccounts map[Currency]FutureSubAccountDecimal
}

type FutureOrder struct {
	Price        float64
	Amount       float64
	AvgPrice     float64
	DealAmount   float64
	OrderID      int64
	OrderID2  	 string
	ClientOrderID string
	OrderTime    int64
	Status       TradeStatus
	Currency     CurrencyPair
	OType        int     //1：开多 2：开空 3：平多 4： 平空
	Side      	 TradeSide
	LeverRate    int     //倍数
	Fee          float64 //手续费
	ContractName string
}

type FutureOrderDecimal struct {
	Price        decimal.Decimal
	Amount       decimal.Decimal
	AvgPrice     decimal.Decimal
	DealAmount   decimal.Decimal
	OrderID  	 string
	ClientOrderID string
	OrderTime    int64
	Status       TradeStatus
	OType        int     //1：开多 2：开空 3：平多 4： 平空
	Side      	 TradeSide
	LeverRate    int     //倍数
	Fee          decimal.Decimal //手续费
	ContractName string
}

type FuturePosition struct {
	BuyAmount      float64
	BuyAvailable   float64
	BuyPriceAvg    float64
	BuyPriceCost   float64
	BuyProfitReal  float64
	BuyProfitUnReal float64
	CreateDate     int64
	LeverRate      int
	SellAmount     float64
	SellAvailable  float64
	SellPriceAvg   float64
	SellPriceCost  float64
	SellProfitReal float64
	SellProfitUnReal float64
	Symbol         CurrencyPair //btc_usd:比特币,ltc_usd:莱特币
	ContractType   string
	ContractId     int64
	InstrumentId   string
	ForceLiquPrice float64 //预估爆仓价
}

type FutureFill struct {
	FillId string
	OrderId string
	ClientOrderId string
	Symbol string
	Side TradeSide
	LastQty int64
	LastPrice float64
	Price float64
	AvgPrice float64
	Commission int64
	TransactionTime int64
}

type FutureFillDecimal struct {
	FillId string
	OrderId string
	ContractName string
	Side TradeSide
	Qty decimal.Decimal
	Price decimal.Decimal
	Fee decimal.Decimal
	TransactionTime int64
	IsMaker bool
}
