package externalshop

import "time"

type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type ShopInfoResponse struct {
	BaseResponse
	Data ShopInfoData `json:"data"`
}

type ShopInfoData struct {
	Nickname string `json:"nickname"`
	Token    string `json:"token"`
	Link     string `json:"link"`
}

type Category struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Image      string `json:"image"`
	GoodsCount int    `json:"goods_count"`
}

type CategoryListResponse struct {
	BaseResponse
	Data []Category `json:"data"`
}

type GoodsExtend struct {
	StockCount          int64 `json:"stock_count"`
	ShowStockType       int   `json:"show_stock_type"`
	SendOrder           int   `json:"send_order"`
	LimitCount          int   `json:"limit_count"`
	QueryPasswordStatus int   `json:"query_password_status"`
}

type GoodsCategory struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type GoodsItem struct {
	GoodsKey     string       `json:"goods_key"`
	GoodsType    string       `json:"goods_type"`
	Name         string       `json:"name"`
	Price        float64      `json:"price"`
	MarketPrice  float64      `json:"market_price"`
	Description  string       `json:"description"`
	Image        string       `json:"image"`
	CouponStatus int          `json:"coupon_status"`
	Category     GoodsCategory `json:"category"`
	Extend       GoodsExtend  `json:"extend"`
}

type GoodsListData struct {
	Total int         `json:"total"`
	List  []GoodsItem `json:"list"`
}

type GoodsListResponse struct {
	BaseResponse
	Data GoodsListData `json:"data"`
}

type PaymentChannelPayType struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type PaymentChannel struct {
	Id         int                   `json:"id"`
	Name       string                `json:"name"`
	Code       string                `json:"code"`
	ShowName   string                `json:"show_name"`
	Status     int                   `json:"status"`
	CustomStatus int                 `json:"custom_status"`
	Rate       float64               `json:"rate"`
	PayType    PaymentChannelPayType `json:"paytype"`
}

type ChannelListResponse struct {
	BaseResponse
	Data []PaymentChannel `json:"data"`
}

type GoodsPriceData struct {
	OriginalAmount  float64 `json:"original_amount"`
	TotalAmount     float64 `json:"total_amount"`
	Fee             float64 `json:"fee"`
	FeePayer        int     `json:"fee_payer"`
	CouponAvailable int     `json:"coupon_available"`
	CouponPrice     float64 `json:"coupon_price"`
}

type GoodsPriceResponse struct {
	BaseResponse
	Data GoodsPriceData `json:"data"`
}

type PayQueryResponse struct {
	BaseResponse
	Data interface{} `json:"data"`
}

func (r PayQueryResponse) IsPaid() bool {
	return r.Code == 1
}

type OrderInfoDeliverResponse struct {
	Cards          []string `json:"cards"`
	ExportCardsURL string   `json:"export_cards_url"`
}

type OrderInfoData struct {
	TradeNo       string                   `json:"trade_no"`
	Status        int                      `json:"status"`
	Sendout       int                      `json:"sendout"`
	TransactionID string                   `json:"transaction_id"`
	SuccessTime   int64                    `json:"success_time"`
	Response      OrderInfoDeliverResponse `json:"response"`
}

type OrderInfoResponse struct {
	BaseResponse
	Data OrderInfoData `json:"data"`
}

type CreateOrderExtend struct {
	JUUID string `json:"juuid"`
}

type CreateOrderRequest struct {
	GoodsKey       string            `json:"goods_key"`
	Quantity       int               `json:"quantity"`
	CouponCode     string            `json:"coupon_code"`
	ChannelID      int               `json:"channel_id"`
	Contact        string            `json:"contact"`
	QueryPassword  string            `json:"query_password"`
	SelectCardsIDs []int             `json:"select_cards_ids"`
	Extend         CreateOrderExtend `json:"extend"`
}

type CreateOrderData struct {
	TradeNo     string  `json:"trade_no"`
	TotalAmount float64 `json:"total_amount"`
	PayURL      string  `json:"payurl"`
}

type CreateOrderResponse struct {
	BaseResponse
	Data CreateOrderData `json:"data"`
}

const (
	OrderStatusCreated             = "created"
	OrderStatusPendingPayment      = "pending_payment"
	OrderStatusPaidWaitingDelivery = "paid_waiting_delivery"
	OrderStatusDelivered           = "delivered"
	OrderStatusFailed              = "failed"
	OrderStatusExpired             = "expired"
	OrderStatusManualReview        = "manual_review"
)

type OrderSyncState struct {
	Status              string
	PaymentConfirmedAt  *time.Time
	DeliveryConfirmedAt *time.Time
	TransactionID       string
	Cards               []string
}
