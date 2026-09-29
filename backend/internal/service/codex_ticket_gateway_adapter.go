package service

import (
	"context"
	"net/http"
	"time"
)

// 旧网关适配器保持现行资源所有者，迁移过程不能额外创建连接池、并发槽或凭据缓存。
var _ CodexTicketRuntime = (*OpenAIGatewayService)(nil)

func (s *OpenAIGatewayService) TicketAccountStore() CodexTicketAccountStore {
	if s == nil {
		return nil
	}
	return s.accountRepo
}

func (s *OpenAIGatewayService) TicketSnapshotAccount(ctx context.Context, id int64) (*Account, error) {
	if s == nil {
		return nil, errTicketRuntimeUnavailable
	}
	// 摘要服务存在时不能在查询失败后偷偷回源，仍由摘要服务执行原有受限回退。
	if s.schedulerSnapshot != nil {
		return s.schedulerSnapshot.GetAccount(ctx, id)
	}
	if s.accountRepo == nil {
		return nil, errTicketRuntimeUnavailable
	}
	return s.accountRepo.GetByID(ctx, id)
}

func (s *OpenAIGatewayService) TicketAccountHeaders(ctx context.Context, headers http.Header, account *Account) error {
	if s == nil {
		return errTicketRuntimeUnavailable
	}
	return resolveAndSetOpenAIChatGPTAccountHeaders(ctx, s.accountRepo, headers, account)
}

func (s *OpenAIGatewayService) TicketSlotTTL() time.Duration {
	if s == nil || s.concurrencyService == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.Gateway.ConcurrencySlotTTLMinutes) * time.Minute
}

func (s *OpenAIGatewayService) AcquireTicketSlot(ctx context.Context, account *Account) (*AcquireResult, error) {
	if s == nil || account == nil {
		return nil, errTicketRuntimeUnavailable
	}
	if s.concurrencyService == nil {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	return s.concurrencyService.AcquireAccountSlot(ctx, account.ID, account.Concurrency)
}

func (s *OpenAIGatewayService) TicketTransportAvailable() bool {
	return s != nil && s.httpUpstream != nil
}

func (s *OpenAIGatewayService) SendTicketHarvest(req *http.Request, proxy string, account *Account) (*http.Response, error) {
	if !s.TicketTransportAvailable() || account == nil {
		return nil, errTicketRuntimeUnavailable
	}
	// 候选继续走专用短连接采集 profile，不能悄悄改用业务 TLS 或业务代理。
	return s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
}

func (s *OpenAIGatewayService) SendTicketVerification(req *http.Request, proxy string, account *Account) (*http.Response, error) {
	if !s.TicketTransportAvailable() || account == nil {
		return nil, errTicketRuntimeUnavailable
	}
	// 复验仍使用账号业务代理和其有效 TLS 模板；账号/凭据变化检查由票据服务保留。
	return s.httpUpstream.DoWithTLS(req, proxy, account.ID, account.Concurrency, s.resolveOpenAITLSProfile(account))
}
