package externalshop

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	autoSyncTickInterval = 2 * time.Minute
	autoSyncMinInterval  = 30 * time.Second
	autoSyncBatchSize    = 100
)

var (
	autoSyncOnce    sync.Once
	autoSyncRunning atomic.Bool
)

type AutoSyncSummary struct {
	Scanned   int `json:"scanned"`
	Attempted int `json:"attempted"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

func StartExternalShopOrderAutoSyncTask() {
	autoSyncOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("external shop order auto-sync task started: tick=%s", autoSyncTickInterval))
			ticker := time.NewTicker(autoSyncTickInterval)
			defer ticker.Stop()

			runExternalShopOrderAutoSyncOnce()
			for range ticker.C {
				runExternalShopOrderAutoSyncOnce()
			}
		})
	})
}

func RunExternalShopOrderAutoSyncOnce() (*AutoSyncSummary, error) {
	return runExternalShopOrderAutoSyncOnce()
}

func runExternalShopOrderAutoSyncOnce() (*AutoSyncSummary, error) {
	if !autoSyncRunning.CompareAndSwap(false, true) {
		return &AutoSyncSummary{}, nil
	}
	defer autoSyncRunning.Store(false)

	cfg := GetConfig()
	if !cfg.IsReady() {
		return &AutoSyncSummary{}, nil
	}

	orders, err := model.ListExternalShopOrdersForAutoSync(
		[]string{OrderStatusCreated, OrderStatusPendingPayment, OrderStatusPaidWaitingDelivery},
		time.Now().Add(-autoSyncMinInterval).Unix(),
		autoSyncBatchSize,
	)
	if err != nil {
		return nil, err
	}
	summary := syncOrdersWithRefresher(
		context.Background(),
		orders,
		time.Now(),
		autoSyncMinInterval,
		RefreshLocalOrder,
	)
	return &summary, nil
}

func syncOrdersWithRefresher(
	ctx context.Context,
	orders []*model.ExternalShopOrder,
	now time.Time,
	minInterval time.Duration,
	refresh func(context.Context, *model.ExternalShopOrder) (*model.ExternalShopOrder, error),
) AutoSyncSummary {
	summary := AutoSyncSummary{
		Scanned: len(orders),
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if !shouldSyncOrder(order, now, minInterval) {
			continue
		}
		summary.Attempted++
		if _, err := refresh(ctx, order); err != nil {
			summary.Failed++
			continue
		}
		summary.Succeeded++
	}
	return summary
}

func shouldSyncOrder(order *model.ExternalShopOrder, now time.Time, minInterval time.Duration) bool {
	if order == nil {
		return false
	}
	switch order.Status {
	case OrderStatusCreated, OrderStatusPendingPayment, OrderStatusPaidWaitingDelivery:
	default:
		return false
	}
	if minInterval <= 0 || order.LastSyncAt <= 0 {
		return true
	}
	return now.Unix()-order.LastSyncAt >= int64(minInterval/time.Second)
}
