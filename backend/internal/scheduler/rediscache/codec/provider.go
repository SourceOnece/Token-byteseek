// 本文件只定义 sched:v4 的快照存储形状；凭据不进入提供商公开 JSON。
package codec

import (
	"encoding/json"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
)

type providerWire struct {
	ID                      int64
	Name                    string
	Notes                   *string
	Platform                string
	Type                    string
	Credentials             map[string]any
	Extra                   map[string]any
	ProxyID                 *int64
	ProxyFallbackOriginID   *int64
	ProxyFallbackOriginName *string
	Concurrency             int
	Priority                int
	RateMultiplier          *float64
	LoadFactor              *int
	Status                  string
	ErrorMessage            string
	LastUsedAt              *time.Time
	ExpiresAt               *time.Time
	AutoPauseOnExpired      bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Schedulable             bool
	RateLimitedAt           *time.Time
	RateLimitResetAt        *time.Time
	OverloadUntil           *time.Time
	TempUnschedulableUntil  *time.Time
	TempUnschedulableReason string
	QuotaAutoPaused         bool `json:"-"`
	SessionWindowStart      *time.Time
	SessionWindowEnd        *time.Time
	SessionWindowStatus     string
	ParentProviderID        *int64
	QuotaDimension          string
	Proxy                   *egress.Proxy
	ProviderGroups          []providerGroupWire
	GroupIDs                []int64
	Groups                  []*groupWire
}

// 原分组记录不回载反向提供商关系；旧字段仍写 null，避免改变现存缓存形状。
type groupWire struct {
	*accessview.GroupConfig
	ProviderGroups []json.RawMessage
}
type providerGroupWire struct {
	ProviderID int64
	GroupID    int64
	CreatedAt  time.Time
	Provider   *providerWire
	Group      *groupWire
}

func encodeGroup(group *accessview.GroupConfig) *groupWire {
	if group == nil {
		return nil
	}
	return &groupWire{GroupConfig: group}
}

func decodeGroup(group *groupWire) *accessview.GroupConfig {
	if group == nil {
		return nil
	}
	return group.GroupConfig
}

// MarshalProviderRecord 保留完整快照中的凭据、额外字段及 nil/空关系集合。
func MarshalProviderRecord(value *provider.Record) ([]byte, error) {
	return json.Marshal(recordToWire(value, make(map[*provider.Record]*providerWire)))
}

func recordToWire(value *provider.Record, seen map[*provider.Record]*providerWire) *providerWire {
	if value == nil {
		return nil
	}
	if previous, ok := seen[value]; ok {
		return previous
	}
	out := &providerWire{}
	seen[value] = out
	out.ID = value.ID
	out.Name = value.Name
	out.Notes = value.Notes
	out.Platform = value.Platform
	out.Type = value.Type
	out.Credentials = value.Credentials
	out.Extra = value.Extra
	out.ProxyID = value.ProxyID
	out.ProxyFallbackOriginID = value.ProxyFallbackOriginID
	out.ProxyFallbackOriginName = value.ProxyFallbackOriginName
	out.Concurrency = value.Concurrency
	out.Priority = value.Priority
	out.RateMultiplier = value.RateMultiplier
	out.LoadFactor = value.LoadFactor
	out.Status = value.Status
	out.ErrorMessage = value.ErrorMessage
	out.LastUsedAt = value.LastUsedAt
	out.ExpiresAt = value.ExpiresAt
	out.AutoPauseOnExpired = value.AutoPauseOnExpired
	out.CreatedAt = value.CreatedAt
	out.UpdatedAt = value.UpdatedAt
	out.Schedulable = value.Schedulable
	out.RateLimitedAt = value.RateLimitedAt
	out.RateLimitResetAt = value.RateLimitResetAt
	out.OverloadUntil = value.OverloadUntil
	out.TempUnschedulableUntil = value.TempUnschedulableUntil
	out.TempUnschedulableReason = value.TempUnschedulableReason
	out.QuotaAutoPaused = value.QuotaAutoPaused
	out.SessionWindowStart = value.SessionWindowStart
	out.SessionWindowEnd = value.SessionWindowEnd
	out.SessionWindowStatus = value.SessionWindowStatus
	out.ParentProviderID = value.ParentProviderID
	out.QuotaDimension = value.QuotaDimension
	out.Proxy = value.Proxy
	out.GroupIDs = value.GroupIDs

	if value.Groups != nil {
		out.Groups = make([]*groupWire, len(value.Groups))
		for i, g := range value.Groups {
			out.Groups[i] = encodeGroup(g)
		}
	}
	if value.ProviderGroups != nil {
		out.ProviderGroups = make([]providerGroupWire, len(value.ProviderGroups))
		for i, g := range value.ProviderGroups {
			out.ProviderGroups[i] = providerGroupWire{ProviderID: g.ProviderID, GroupID: g.GroupID, CreatedAt: g.CreatedAt, Provider: recordToWire(g.Provider, seen), Group: encodeGroup(g.Group)}
		}
	}
	return out
}

// UnmarshalProviderRecord 只恢复受控完整读取结果，不把凭据交给调度评分核心。
func UnmarshalProviderRecord(payload []byte) (*provider.Record, error) {
	var wire providerWire
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, err
	}
	return recordFromWire(&wire), nil
}

func recordFromWire(value *providerWire) *provider.Record {
	if value == nil {
		return nil
	}
	out := &provider.Record{Now: time.Now, LoadLocation: time.LoadLocation}
	out.ID = value.ID
	out.Name = value.Name
	out.Notes = value.Notes
	out.Platform = value.Platform
	out.Type = value.Type
	out.Credentials = value.Credentials
	out.Extra = value.Extra
	out.ProxyID = value.ProxyID
	out.ProxyFallbackOriginID = value.ProxyFallbackOriginID
	out.ProxyFallbackOriginName = value.ProxyFallbackOriginName
	out.Concurrency = value.Concurrency
	out.Priority = value.Priority
	out.RateMultiplier = value.RateMultiplier
	out.LoadFactor = value.LoadFactor
	out.Status = value.Status
	out.ErrorMessage = value.ErrorMessage
	out.LastUsedAt = value.LastUsedAt
	out.ExpiresAt = value.ExpiresAt
	out.AutoPauseOnExpired = value.AutoPauseOnExpired
	out.CreatedAt = value.CreatedAt
	out.UpdatedAt = value.UpdatedAt
	out.Schedulable = value.Schedulable
	out.RateLimitedAt = value.RateLimitedAt
	out.RateLimitResetAt = value.RateLimitResetAt
	out.OverloadUntil = value.OverloadUntil
	out.TempUnschedulableUntil = value.TempUnschedulableUntil
	out.TempUnschedulableReason = value.TempUnschedulableReason
	out.QuotaAutoPaused = value.QuotaAutoPaused
	out.SessionWindowStart = value.SessionWindowStart
	out.SessionWindowEnd = value.SessionWindowEnd
	out.SessionWindowStatus = value.SessionWindowStatus
	out.ParentProviderID = value.ParentProviderID
	out.QuotaDimension = value.QuotaDimension
	out.Proxy = value.Proxy
	out.GroupIDs = value.GroupIDs

	if value.Groups != nil {
		out.Groups = make([]*accessview.GroupConfig, len(value.Groups))
		for i, g := range value.Groups {
			out.Groups[i] = decodeGroup(g)
		}
	}
	if value.ProviderGroups != nil {
		out.ProviderGroups = make([]provider.GroupMembership, len(value.ProviderGroups))
		for i, g := range value.ProviderGroups {
			out.ProviderGroups[i] = provider.GroupMembership{ProviderID: g.ProviderID, GroupID: g.GroupID, CreatedAt: g.CreatedAt, Provider: recordFromWire(g.Provider), Group: decodeGroup(g.Group)}
		}
	}
	return out
}
