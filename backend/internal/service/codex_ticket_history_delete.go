package service

import (
	"context"
	"errors"
)

// 删除入口与只读历史接口分离，不扩散到账号/质量检测仓储。
var ErrTicketHistoryActive = errors.New("该批次正在采集，请结束后再删除")
var ErrTicketHistoryNotFound = errors.New("采集日志不存在或已删除")

type CodexTicketHistoryDeleteRepository interface {
	DeleteTicketHistory(context.Context, string, int64) (int64, error)
}

func (s *CodexTicketService) DeleteTicketHistory(ctx context.Context, runID string, eventID int64) (int64, error) {
	if s == nil || s.gateway == nil {
		return 0, errors.New("采集历史服务不可用")
	}
	r, ok := s.gateway.accountRepo.(CodexTicketHistoryDeleteRepository)
	if !ok {
		return 0, errors.New("采集历史删除不可用")
	}
	// 空 runID 仅表示专用“清空已结束历史”入口，不接受事件 ID 混用。
	if eventID < 0 || (runID == "" && eventID != 0) {
		return 0, errors.New("删除范围无效")
	}
	return r.DeleteTicketHistory(ctx, runID, eventID)
}
