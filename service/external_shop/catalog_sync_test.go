package externalshop

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

type fakeCatalogClient struct {
	shopInfo        ShopInfoResponse
	categories      []Category
	goodsByCategory map[int][]GoodsItem
	pageCalls       []int
}

func (f *fakeCatalogClient) GetShopInfo(ctx context.Context) (ShopInfoResponse, error) {
	return f.shopInfo, nil
}

func (f *fakeCatalogClient) ListCategories(ctx context.Context, goodsType string) (CategoryListResponse, error) {
	return CategoryListResponse{
		BaseResponse: BaseResponse{Code: 1},
		Data:         f.categories,
	}, nil
}

func (f *fakeCatalogClient) ListGoods(ctx context.Context, categoryID int, goodsType string, current int, pageSize int, keywords string) (GoodsListResponse, error) {
	f.pageCalls = append(f.pageCalls, current)
	all := f.goodsByCategory[categoryID]
	start := (current - 1) * pageSize
	if start >= len(all) {
		return GoodsListResponse{
			BaseResponse: BaseResponse{Code: 1},
			Data: GoodsListData{
				Total: len(all),
				List:  []GoodsItem{},
			},
		}, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return GoodsListResponse{
		BaseResponse: BaseResponse{Code: 1},
		Data: GoodsListData{
			Total: len(all),
			List:  all[start:end],
		},
	}, nil
}

func TestSyncCatalogWithClientPaginatesAndCountsDisabledGoods(t *testing.T) {
	t.Parallel()

	client := &fakeCatalogClient{
		shopInfo: ShopInfoResponse{
			BaseResponse: BaseResponse{Code: 1},
			Data:         ShopInfoData{Nickname: "AI工具圈"},
		},
		categories: []Category{
			{Id: 1, Name: "默认分类"},
		},
		goodsByCategory: map[int][]GoodsItem{
			1: {
				{GoodsKey: "g1", GoodsType: "card", Name: "商品1", Category: GoodsCategory{Id: 1, Name: "默认分类"}},
				{GoodsKey: "g2", GoodsType: "card", Name: "商品2", Category: GoodsCategory{Id: 1, Name: "默认分类"}},
				{GoodsKey: "g3", GoodsType: "card", Name: "商品3", Category: GoodsCategory{Id: 1, Name: "默认分类"}},
			},
		},
	}

	var upserted []string
	summary, err := syncCatalogWithClient(
		context.Background(),
		client,
		Config{
			Enabled:   true,
			BaseURL:   "https://pay.ldxp.cn",
			ShopToken: "shop-token",
		},
		2,
		func(good *model.ExternalShopGood) error {
			upserted = append(upserted, good.GoodsKey)
			return nil
		},
		func(provider string, shopToken string, keepGoodsKeys []string) (int64, error) {
			require.Equal(t, ProviderLDXP, provider)
			require.Equal(t, "shop-token", shopToken)
			require.ElementsMatch(t, []string{"g1", "g2", "g3"}, keepGoodsKeys)
			return 2, nil
		},
		func() time.Time {
			return time.Unix(1773380000, 0)
		},
	)
	require.NoError(t, err)
	require.Equal(t, "AI工具圈", summary.ShopName)
	require.Equal(t, 1, summary.Categories)
	require.Equal(t, 3, summary.GoodsSynced)
	require.Equal(t, 2, summary.GoodsDisabled)
	require.ElementsMatch(t, []string{"g1", "g2", "g3"}, upserted)
	require.Equal(t, []int{1, 2}, client.pageCalls)
}
