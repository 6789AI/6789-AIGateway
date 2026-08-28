package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func CommitPromotionUse(ctx context.Context, info *relaycommon.RelayInfo) bool {
	if info == nil || info.PromotionActivityKey == "" {
		return true
	}
	if err := model.CommitPromotionUse(info.RequestId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("commit model promotion use failed: %s", err.Error()))
		return false
	}
	info.PromotionActivityKey = ""
	return true
}

func RefundPromotionUse(ctx context.Context, info *relaycommon.RelayInfo) bool {
	if info == nil || info.PromotionActivityKey == "" {
		return true
	}
	if err := model.RefundPromotionUse(info.RequestId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("refund model promotion use failed: %s", err.Error()))
		return false
	}
	info.PromotionActivityKey = ""
	return true
}

func commitTaskPromotionUse(ctx context.Context, task *model.Task) {
	requestId := task.PrivateData.PromotionRequestId
	if requestId == "" {
		return
	}
	if err := model.CommitPromotionUse(requestId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("commit task model promotion use failed (task=%s): %s", task.TaskID, err.Error()))
	}
}

func refundTaskPromotionUse(ctx context.Context, task *model.Task) {
	requestId := task.PrivateData.PromotionRequestId
	if requestId == "" {
		return
	}
	if err := model.RefundPromotionUse(requestId); err != nil {
		logger.LogError(ctx, fmt.Sprintf("refund task model promotion use failed (task=%s): %s", task.TaskID, err.Error()))
	}
}
