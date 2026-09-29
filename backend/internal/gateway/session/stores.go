// 消费者按会话/回放/视频/风控分别声明能力；不合并不同缓存语义。
package session

import (
	"context"
	"errors"
	"time"
)

// OpenAIWSSessionPreemptionCache is an optional GatewayCache capability. The
// production Redis cache implements all operations atomically; cache stubs do
// not need to implement it for ordinary gateway tests.
type OpenAIWSSessionPreemptionCache interface {
	ClaimOpenAIResponsesSessionWindow(ctx context.Context, groupID int64, sessionHash string, owner []byte, ttl time.Duration) ([]byte, error)
	CompareAndRefreshOpenAIResponsesSessionWindow(ctx context.Context, groupID int64, sessionHash string, expected []byte, ttl time.Duration) (bool, error)
	CompareAndDeleteOpenAIResponsesSessionWindow(ctx context.Context, groupID int64, sessionHash string, expected []byte) (bool, error)
}

// CyberSessionBlockStore 是 cyber 会话屏蔽表的存取接口。
// rediscache 的 gatewayCache 通过原生可选能力适配接入，测试桩未实现时自动降级为关闭。
type CyberSessionBlockStore interface {
	SetCyberSessionBlocked(ctx context.Context, scopeKey string, keys []string, ttl time.Duration) error
	IsCyberSessionScopeActive(ctx context.Context, scopeKey string) (bool, error)
	FindCyberSessionBlocked(ctx context.Context, keys []string) (string, error)
}

// ErrReasoningContentNotFound 表示按 reasoning item id 查询缓存时未命中。
var ErrReasoningContentNotFound = errors.New("reasoning content not found")

// GatewayCache 定义网关服务的缓存操作接口。
// 提供粘性会话（Sticky Session）的存储、查询、刷新和删除功能。
//
// GatewayCache defines cache operations for gateway service.
// Provides sticky session storage, retrieval, refresh and deletion capabilities.
type GatewayCache interface {
	// GetSessionProviderID 获取粘性会话绑定的提供商 ID
	// Get the provider ID bound to a sticky session
	GetSessionProviderID(ctx context.Context, groupID int64, sessionHash string) (int64, error)
	// SetSessionProviderID 设置粘性会话与提供商的绑定关系
	// Set the binding between sticky session and provider
	SetSessionProviderID(ctx context.Context, groupID int64, sessionHash string, providerID int64, ttl time.Duration) error
	// RefreshSessionTTL 刷新粘性会话的过期时间
	// Refresh the expiration time of a sticky session
	RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error
	// DeleteSessionProviderID 删除粘性会话绑定，用于提供商不可用时主动清理
	// Delete sticky session binding, used to proactively clean up when provider becomes unavailable
	DeleteSessionProviderID(ctx context.Context, groupID int64, sessionHash string) error
	// SetSessionOwnerGroupID 首次记录显式会话所属分组；返回 true 表示本次写入成功。
	SetSessionOwnerGroupID(ctx context.Context, userID int64, source, sessionHash string, groupID int64, ttl time.Duration) (bool, error)
	// GetSessionOwnerGroupID 读取显式会话首次归属分组。
	GetSessionOwnerGroupID(ctx context.Context, userID int64, source, sessionHash string) (int64, error)
	// RefreshSessionOwnerTTL 刷新显式会话归属记录的过期时间。
	RefreshSessionOwnerTTL(ctx context.Context, userID int64, source, sessionHash string, ttl time.Duration) error
}

// ReasoningContentCache 是 Responses→Chat 桥接使用的可选缓存能力。
// 与 GatewayCache 分离，避免不需要 reasoning 回放的缓存实现被迫扩展接口。
type ReasoningContentCache interface {
	SetReasoningContent(ctx context.Context, itemID string, content string, ttl time.Duration) error
	GetReasoningContent(ctx context.Context, itemID string) (string, error)
}

// GrokVideoBillingCache 为异步视频任务保存创建时定价快照，并跨实例防止轮询重复扣费。
// 它保持为独立子接口，避免与 Grok 无关的缓存实现被迫提供这组能力。
type GrokVideoBillingCache interface {
	SetGrokVideoPendingBilling(ctx context.Context, key string, payload []byte, ttl time.Duration) error
	GetGrokVideoPendingBilling(ctx context.Context, key string) ([]byte, error)
	ClaimGrokVideoBilled(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseGrokVideoBilled(ctx context.Context, key string) error
}
