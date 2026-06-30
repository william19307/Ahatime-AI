package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	seedanceMaintenanceTickInterval = 5 * time.Minute
	seedanceUploadCleanupBatchSize  = 100
	seedanceAssetSyncBatchSize      = 20
)

var (
	seedanceMaintenanceOnce    sync.Once
	seedanceMaintenanceRunning atomic.Bool
)

func StartSeedanceMaintenanceTask() {
	seedanceMaintenanceOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("seedance maintenance task started: tick=%s", seedanceMaintenanceTickInterval))
			ticker := time.NewTicker(seedanceMaintenanceTickInterval)
			defer ticker.Stop()

			runSeedanceMaintenanceOnce()
			for range ticker.C {
				runSeedanceMaintenanceOnce()
			}
		})
	})
}

func runSeedanceMaintenanceOnce() {
	if !seedanceMaintenanceRunning.CompareAndSwap(false, true) {
		return
	}
	defer seedanceMaintenanceRunning.Store(false)

	ctx := context.Background()
	now := time.Now().Unix()
	svc := NewSeedanceAssetService()

	uploads, err := model.ListExpiredSeedanceUploads(now, seedanceUploadCleanupBatchSize)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("seedance upload cleanup list failed: %v", err))
	} else {
		for _, upload := range uploads {
			if upload.StoragePath != "" {
				_ = os.Remove(upload.StoragePath)
			}
			if err := model.DeleteSeedanceUploadByID(upload.Id); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("seedance upload cleanup delete failed id=%d: %v", upload.Id, err))
			}
		}
	}

	assets, err := model.ListSeedanceAssetsPendingSync(seedanceAssetSyncBatchSize)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("seedance asset sync list failed: %v", err))
		return
	}
	for _, asset := range assets {
		if _, syncErr := svc.GetAsset(asset.UserId, asset.Id, true); syncErr != nil && common.DebugEnabled {
			logger.LogDebug(ctx, "seedance asset sync skipped id=%d user=%d err=%v", asset.Id, asset.UserId, syncErr)
		}
	}
}
