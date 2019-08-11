package huobifuture

import (
	"github.com/shopspring/decimal"
	"github.com/stephenlyu/GoEx"
)

type ContractInfo struct {
	Symbol string
	ContractCode string 			`json:"contract_code"`
	ContractType string 			`json:"contract_type"`
	ContractSize decimal.Decimal	`json:"contract_size"`
	PriceTick decimal.Decimal		`json:"price_tick"`
	DeliveryDate string 			`json:"delivery_date"`
	CreateDate string 				`json:"create_date"`
	ContractStatus int 				`json:"contract_status"`
}

const (
	DirectionBuy = "buy"
	DirectionSell = "sell"

	OffsetOpen = "open"
	OffsetClose = "close"

	PriceTypeLimit = "limit"
	PriceTypeOpponent = "opponent"
	PriceTypePostOnly = "post_only"
)

type OrderReq struct {
	ContractCode string 		`json:"contract_code"`
	ClientOid int64 			`json:"client_order_id"`
	Price decimal.Decimal		`json:"price"`
	Volume int64 				`json:"volume"`
	Direction string			`json:"direction"`
	Offset string				`json:"offset"`
	LeverRate int 				`json:"lever_rate"`
	OrderPriceType string 		`json:"order_price_type"`
}

type OrderInfo struct {
	Symbol 			 string
	ContractType 	 string 		`json:"contract_type"`
	ContractCode 	 string 		`json:"contract_code"`
	Volume 			 decimal.Decimal
	Price 			 decimal.Decimal
	Direction 		 string
	Offset 			 string
	LeverRate 		 int 			`json:"lever_rate"`
	OrderId 		 decimal.Decimal`json:"order_id"`
	ClientOid		 decimal.Decimal`json:"client_order_id"`
	CreateAt 		 int64 			`json:"create_at"`
	TradeVolume 	 decimal.Decimal`json:"trade_volume"`
	TradeTurnover    decimal.Decimal`json:"trade_turnover"`
	Fee 			 decimal.Decimal`json:"fee"`
	TradeAvgPrice    decimal.Decimal`json:"trade_avg_price"`
	Status  		 int
	OrderSource 	 string 		`json:"order_source"`
}

func (this *OrderInfo) ToOrderDecimal() *goex.FutureOrderDecimal {
	var status goex.TradeStatus
	//3未成交 4部分成交 5部分成交已撤单 6全部成交 7已撤单
	switch this.Status {
	case 3:
		status = goex.ORDER_UNFINISH
	case 5, 7:
		status = goex.ORDER_CANCEL
	case 6:
		status = goex.ORDER_FINISH
	case 4:
		status = goex.ORDER_PART_FINISH
	}

	var side goex.TradeSide
	var oType int //1：开多 2：开空 3：平多 4： 平空
	switch this.Direction {
	case DirectionBuy:
		side = goex.BUY
		switch this.Offset {
		case OffsetOpen:
			oType = 1
		case OffsetClose:
			oType = 4
		}
	case DirectionSell:
		side = goex.SELL
		switch this.Offset {
		case OffsetOpen:
			oType = 2
		case OffsetClose:
			oType = 3
		}
	}

	return &goex.FutureOrderDecimal{
		Price: this.Price,
		Amount: this.Volume,
		AvgPrice: this.TradeAvgPrice,
		DealAmount: this.TradeVolume,
		OrderID: this.OrderId.String(),
		ClientOrderID: this.ClientOid.String(),
		OrderTime: this.CreateAt,
		Status: status,
		Side: side,
		OType: oType,
		LeverRate: this.LeverRate,
		Fee: this.Fee,
		ContractName: this.ContractCode,
	}
}

type PositionInfo struct {
	Symbol 			 string
	ContractType 	 string 		`json:"contract_type"`
	ContractCode 	 string 		`json:"contract_code"`
	Volume 			 decimal.Decimal
	Available 		 decimal.Decimal
	Frozen 			 decimal.Decimal `json:"cost_open"`
	CostOpen 		 decimal.Decimal
	CostHold 		 decimal.Decimal `json:"cost_hold"`
	ProfitUnreal 	 decimal.Decimal `json:"profit_unreal"`
	ProfitRate 		 decimal.Decimal `json:"profit_rate"`
	PositionMargin   decimal.Decimal `json:"position_margin"`
	LeverRate 		 decimal.Decimal `json:"lever_rate"`
	Direction    	 string 		 `json:"direction"`
}
