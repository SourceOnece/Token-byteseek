package postgres

import (
	"context"
	"errors"
	"strings"

	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// CreateRecord 只写核心行并回填身份/版本；事务由原调用方持有，分组/票据/outbox不能在此提前提交。
func (s *Store) CreateRecord(ctx context.Context, record *provider.Record) error {
	if record == nil {
		return errors.New("provider record cannot be nil")
	}
	builder := s.client.Account.Create().
		SetName(record.Name).
		SetNillableNotes(record.Notes).
		SetPlatform(record.Platform).
		SetType(record.Type).
		SetCredentials(normalizeJSONMap(record.Credentials)).
		SetExtra(normalizeJSONMap(record.Extra)).
		SetConcurrency(record.Concurrency).
		SetPriority(record.Priority).
		SetStatus(record.Status).
		SetErrorMessage(record.ErrorMessage).
		SetSchedulable(record.Schedulable).
		SetAutoPauseOnExpired(record.AutoPauseOnExpired)

	if record.RateMultiplier != nil {
		builder.SetRateMultiplier(*record.RateMultiplier)
	}
	if record.LoadFactor != nil {
		builder.SetLoadFactor(*record.LoadFactor)
	}

	if record.ProxyID != nil {
		builder.SetProxyID(*record.ProxyID)
	}
	if record.LastUsedAt != nil {
		builder.SetLastUsedAt(*record.LastUsedAt)
	}
	if record.ExpiresAt != nil {
		builder.SetExpiresAt(*record.ExpiresAt)
	}
	if record.RateLimitedAt != nil {
		builder.SetRateLimitedAt(*record.RateLimitedAt)
	}
	if record.RateLimitResetAt != nil {
		builder.SetRateLimitResetAt(*record.RateLimitResetAt)
	}
	if record.OverloadUntil != nil {
		builder.SetOverloadUntil(*record.OverloadUntil)
	}
	if record.SessionWindowStart != nil {
		builder.SetSessionWindowStart(*record.SessionWindowStart)
	}
	if record.SessionWindowEnd != nil {
		builder.SetSessionWindowEnd(*record.SessionWindowEnd)
	}
	if record.SessionWindowStatus != "" {
		builder.SetSessionWindowStatus(record.SessionWindowStatus)
	}

	builder.SetQuotaDimension(dbaccount.QuotaDimension(quotaDimension(record.QuotaDimension)))
	if record.ParentProviderID != nil {
		builder.SetParentAccountID(*record.ParentProviderID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	record.ID = created.ID
	record.CreatedAt = created.CreatedAt
	record.UpdatedAt = created.UpdatedAt
	return nil
}

// 零值保持原全局额度维度；不恢复已废弃账号配置。
func quotaDimension(value string) string {
	if strings.TrimSpace(value) == "" {
		return "global"
	}
	return value
}
func normalizeJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
