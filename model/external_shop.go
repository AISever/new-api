package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type ExternalShopGood struct {
	Id int `json:"id"`

	Provider  string `json:"provider" gorm:"type:varchar(32);not null;index:idx_external_shop_goods_provider_token_key,priority:1"`
	ShopToken string `json:"shop_token" gorm:"type:varchar(64);not null;index:idx_external_shop_goods_provider_token_key,priority:2"`
	GoodsKey  string `json:"goods_key" gorm:"type:varchar(64);not null;index:idx_external_shop_goods_provider_token_key,priority:3"`

	GoodsType    string `json:"goods_type" gorm:"type:varchar(32);default:'card'"`
	Name         string `json:"name" gorm:"type:varchar(255);not null"`
	Description  string `json:"description" gorm:"type:text"`
	Image        string `json:"image" gorm:"type:text"`
	CategoryId   int    `json:"category_id" gorm:"index"`
	CategoryName string `json:"category_name" gorm:"type:varchar(128);default:''"`

	Price       float64 `json:"price" gorm:"type:decimal(10,6);not null;default:0"`
	MarketPrice float64 `json:"market_price" gorm:"type:decimal(10,6);not null;default:0"`

	CouponStatus        int   `json:"coupon_status" gorm:"default:0"`
	StockCount          int64 `json:"stock_count" gorm:"default:0"`
	ShowStockType       int   `json:"show_stock_type" gorm:"default:0"`
	LimitCount          int   `json:"limit_count" gorm:"default:0"`
	SendOrder           int   `json:"send_order" gorm:"default:0"`
	QueryPasswordStatus int   `json:"query_password_status" gorm:"default:0"`

	Enabled    bool   `json:"enabled" gorm:"default:true;index"`
	RawPayload string `json:"raw_payload" gorm:"type:text"`
	SyncedAt   int64  `json:"synced_at" gorm:"bigint;index"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (g *ExternalShopGood) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	g.CreatedAt = now
	g.UpdatedAt = now
	return nil
}

func (g *ExternalShopGood) BeforeUpdate(tx *gorm.DB) error {
	g.UpdatedAt = time.Now().Unix()
	return nil
}

func UpsertExternalShopGood(good *ExternalShopGood) error {
	if good == nil {
		return errors.New("external shop good is nil")
	}
	var existing ExternalShopGood
	err := DB.Where("provider = ? AND shop_token = ? AND goods_key = ?", good.Provider, good.ShopToken, good.GoodsKey).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DB.Create(good).Error
		}
		return err
	}
	good.Id = existing.Id
	return DB.Model(&existing).Updates(map[string]interface{}{
		"goods_type":            good.GoodsType,
		"name":                  good.Name,
		"description":           good.Description,
		"image":                 good.Image,
		"category_id":           good.CategoryId,
		"category_name":         good.CategoryName,
		"price":                 good.Price,
		"market_price":          good.MarketPrice,
		"coupon_status":         good.CouponStatus,
		"stock_count":           good.StockCount,
		"show_stock_type":       good.ShowStockType,
		"limit_count":           good.LimitCount,
		"send_order":            good.SendOrder,
		"query_password_status": good.QueryPasswordStatus,
		"enabled":               good.Enabled,
		"raw_payload":           good.RawPayload,
		"synced_at":             good.SyncedAt,
		"updated_at":            time.Now().Unix(),
	}).Error
}

func ListExternalShopGoods(provider string, shopToken string, enabledOnly bool) ([]ExternalShopGood, error) {
	query := DB.Model(&ExternalShopGood{}).Where("provider = ? AND shop_token = ?", provider, shopToken)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	var goods []ExternalShopGood
	err := query.Order("category_id asc, id desc").Find(&goods).Error
	return goods, err
}

func GetExternalShopGoodByGoodsKey(provider string, shopToken string, goodsKey string) (*ExternalShopGood, error) {
	var good ExternalShopGood
	err := DB.Where("provider = ? AND shop_token = ? AND goods_key = ?", provider, shopToken, goodsKey).First(&good).Error
	if err != nil {
		return nil, err
	}
	return &good, nil
}

func DisableMissingExternalShopGoods(provider string, shopToken string, keepGoodsKeys []string) (int64, error) {
	query := DB.Model(&ExternalShopGood{}).Where("provider = ? AND shop_token = ?", provider, shopToken)
	if len(keepGoodsKeys) > 0 {
		query = query.Where("goods_key NOT IN ?", keepGoodsKeys)
	}
	result := query.Updates(map[string]interface{}{
		"enabled":    false,
		"updated_at": time.Now().Unix(),
	})
	return result.RowsAffected, result.Error
}

func MarkExternalShopGoodOutOfStock(provider string, shopToken string, goodsKey string) error {
	return DB.Model(&ExternalShopGood{}).
		Where("provider = ? AND shop_token = ? AND goods_key = ?", provider, shopToken, goodsKey).
		Updates(map[string]interface{}{
			"stock_count": 0,
			"updated_at":  time.Now().Unix(),
		}).Error
}

type ExternalShopOrder struct {
	Id int `json:"id"`

	LocalTradeNo    string `json:"local_trade_no" gorm:"type:varchar(64);uniqueIndex"`
	UpstreamTradeNo string `json:"upstream_trade_no" gorm:"type:varchar(64);index"`

	UserId int `json:"user_id" gorm:"index"`

	Provider  string `json:"provider" gorm:"type:varchar(32);not null;index"`
	ShopToken string `json:"shop_token" gorm:"type:varchar(64);not null;index"`
	GoodsKey  string `json:"goods_key" gorm:"type:varchar(64);not null;index"`
	GoodsName string `json:"goods_name" gorm:"type:varchar(255);default:''"`

	Quantity    int    `json:"quantity" gorm:"default:1"`
	Contact     string `json:"contact" gorm:"type:varchar(255);default:''"`
	ChannelId   int    `json:"channel_id" gorm:"default:0"`
	ChannelCode string `json:"channel_code" gorm:"type:varchar(64);default:''"`
	ChannelName string `json:"channel_name" gorm:"type:varchar(128);default:''"`

	Amount          float64 `json:"amount" gorm:"type:decimal(10,6);default:0"`
	Status          string  `json:"status" gorm:"type:varchar(32);index"`
	PayUrl          string  `json:"pay_url" gorm:"type:text"`
	TransactionId   string  `json:"transaction_id" gorm:"type:varchar(128);default:''"`
	DeliveryPayload string  `json:"delivery_payload" gorm:"type:text"`
	ProviderPayload string  `json:"provider_payload" gorm:"type:text"`
	LastError       string  `json:"last_error" gorm:"type:text"`

	PaymentConfirmedAt  int64 `json:"payment_confirmed_at" gorm:"bigint;default:0"`
	DeliveryConfirmedAt int64 `json:"delivery_confirmed_at" gorm:"bigint;default:0"`
	LastSyncAt          int64 `json:"last_sync_at" gorm:"bigint;default:0"`
	CreatedAt           int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64 `json:"updated_at" gorm:"bigint"`
}

func (o *ExternalShopOrder) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	o.CreatedAt = now
	o.UpdatedAt = now
	return nil
}

func (o *ExternalShopOrder) BeforeUpdate(tx *gorm.DB) error {
	o.UpdatedAt = time.Now().Unix()
	return nil
}

func (o *ExternalShopOrder) Insert() error {
	return DB.Create(o).Error
}

func (o *ExternalShopOrder) Update() error {
	return DB.Save(o).Error
}

func (o *ExternalShopOrder) Delete() error {
	return DB.Delete(o).Error
}

func GetExternalShopOrderByLocalTradeNo(localTradeNo string) (*ExternalShopOrder, error) {
	var order ExternalShopOrder
	err := DB.Where("local_trade_no = ?", localTradeNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetExternalShopOrderByUpstreamTradeNo(upstreamTradeNo string) (*ExternalShopOrder, error) {
	var order ExternalShopOrder
	err := DB.Where("upstream_trade_no = ?", upstreamTradeNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func ListExternalShopOrdersByUser(userId int, pageInfo *common.PageInfo, sortBy string, sortOrder string) ([]ExternalShopOrder, int64, error) {
	var (
		orders []ExternalShopOrder
		total  int64
	)
	query := DB.Model(&ExternalShopOrder{}).Where("user_id = ?", userId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order(buildExternalShopOrderSort(sortBy, sortOrder)).Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

type ExternalShopOrderQuery struct {
	Keyword   string
	Status    string
	UserId    int
	SortBy    string
	SortOrder string
}

func ListExternalShopOrders(pageInfo *common.PageInfo, queryParams ExternalShopOrderQuery) ([]ExternalShopOrder, int64, error) {
	var (
		orders []ExternalShopOrder
		total  int64
	)
	query := DB.Model(&ExternalShopOrder{})
	if queryParams.UserId > 0 {
		query = query.Where("user_id = ?", queryParams.UserId)
	}
	if status := strings.TrimSpace(queryParams.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(queryParams.Keyword); keyword != "" {
		like := "%%" + keyword + "%%"
		query = query.Where(
			"local_trade_no LIKE ? OR upstream_trade_no LIKE ? OR goods_name LIKE ? OR contact LIKE ?",
			like, like, like, like,
		)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order(buildExternalShopOrderSort(queryParams.SortBy, queryParams.SortOrder)).Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func buildExternalShopOrderSort(sortBy string, sortOrder string) string {
	columnMap := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"amount":     "amount",
		"status":     "status",
	}

	column, ok := columnMap[strings.TrimSpace(sortBy)]
	if !ok {
		column = "id"
	}

	order := strings.ToLower(strings.TrimSpace(sortOrder))
	if order != "asc" {
		order = "desc"
	}

	return column + " " + order
}

func ListExternalShopOrdersForAutoSync(statuses []string, minLastSyncAt int64, limit int) ([]*ExternalShopOrder, error) {
	if limit <= 0 {
		limit = 100
	}
	query := DB.Model(&ExternalShopOrder{}).Where("status IN ?", statuses)
	if minLastSyncAt > 0 {
		query = query.Where("last_sync_at = 0 OR last_sync_at <= ?", minLastSyncAt)
	}
	var orders []*ExternalShopOrder
	err := query.Order("id asc").Limit(limit).Find(&orders).Error
	return orders, err
}
