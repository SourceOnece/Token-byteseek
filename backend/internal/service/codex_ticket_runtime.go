package service

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// CodexTicketAccountStore 只暴露票据需要的账号读取。
// 原子创建、历史和调度写入继续通过既有可选接口提供，禁止降级为无条件写账号。
type CodexTicketAccountStore interface {
	GetByID(context.Context, int64) (*Account, error)
	GetByIDs(context.Context, []int64) ([]*Account, error)
	ListByPlatform(context.Context, string) ([]Account, error)
}

// CodexTicketRuntime 是票据与网关运行资源之间的迁移边界。
// 新 Provider 实现必须保持凭据解析、只读摘要补全、共享并发槽及采集/复验出口各自的语义。
type CodexTicketRuntime interface {
	TicketAccountStore() CodexTicketAccountStore
	TicketSnapshotAccount(context.Context, int64) (*Account, error)
	GetAccessToken(context.Context, *Account) (string, string, error)
	TicketAccountHeaders(context.Context, http.Header, *Account) error
	TicketSlotTTL() time.Duration
	AcquireTicketSlot(context.Context, *Account) (*AcquireResult, error)
	TicketTransportAvailable() bool
	SendTicketHarvest(*http.Request, string, *Account) (*http.Response, error)
	SendTicketVerification(*http.Request, string, *Account) (*http.Response, error)
}

// NewCodexTicketService 只装配依赖；运行由组合根显式启动，构造不派发后台请求。
func NewCodexTicketService(runtime CodexTicketRuntime, settings SettingRepository, cache CodexTicketCache, cipher SecretEncryptor, ipProber ProxyExitInfoProber, proxies ProxyRepository) *CodexTicketService {
	return &CodexTicketService{runtime: runtime, settings: settings, cache: cache, cipher: cipher, ipProber: ipProber, proxyRepo: proxies}
}

// ticketAccounts 统一缺少运行能力时的边界，不在票据服务访问旧网关的私有字段。
func (s *CodexTicketService) ticketAccounts() CodexTicketAccountStore {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.TicketAccountStore()
}

var errTicketRuntimeUnavailable = errors.New("票据运行服务不可用")
