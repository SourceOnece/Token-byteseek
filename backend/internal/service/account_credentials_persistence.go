package service

import (
	"context"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

type accountCredentialsUpdater interface {
	UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error
}

func persistAccountCredentials(ctx context.Context, repo AccountRepository, account *Account, credentials map[string]any) error {
	if repo == nil || account == nil {
		return nil
	}

	value := account.ProviderRecord()
	base := providerCredentialStoreAdapter{repo: repo, original: account}
	var store provider.CredentialUpdateStore = base
	if updater, ok := repo.(accountCredentialsUpdater); ok {
		store = providerCredentialFieldsAdapter{providerCredentialStoreAdapter: base, updater: updater}
	}
	updated, err := provider.PersistCredentials(ctx, store, value, credentials, func(message string, args ...any) { slog.Warn(message, args...) })
	if updated {
		account.Credentials = value.Credentials
	}
	return err
}

type providerCredentialStoreAdapter struct {
	repo AccountRepository
	original *Account
}

func (a providerCredentialStoreAdapter) Update(ctx context.Context, value *provider.Record) error {
	// 回退继续写完整原账号；包括分组/运行投影，不能用仅ID和凭据的新对象覆盖。
	a.original.Credentials = value.Credentials
	return a.repo.Update(ctx, a.original)
}

type providerCredentialFieldsAdapter struct {
	providerCredentialStoreAdapter
	updater accountCredentialsUpdater
}

func (a providerCredentialFieldsAdapter) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	// 保留原实现先更新调用对象、再调用专用写入的时机，即使写入失败也不偷偷复原对象。
	a.original.Credentials = credentials
	return a.updater.UpdateCredentials(ctx, id, credentials)
}

// sparkShadowAllowedCredentialKeys 是 spark 影子账号唯一可写的凭据键集合(仅模型映射)。
// 校验(isAllowed)与 sanitize 共用此单一来源,避免两处独立硬编码列表漂移。
var sparkShadowAllowedCredentialKeys = map[string]struct{}{
	"model_mapping":         {},
	"compact_model_mapping": {},
}

func isAllowedSparkShadowCredentialsUpdate(credentials map[string]any) bool {
	if credentials == nil {
		return true
	}
	for key := range credentials {
		if _, ok := sparkShadowAllowedCredentialKeys[key]; !ok {
			return false
		}
	}
	return true
}

func sanitizeSparkShadowCredentials(credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(sparkShadowAllowedCredentialKeys))
	for key := range sparkShadowAllowedCredentialKeys {
		if value, ok := credentials[key]; ok && value != nil {
			out[key] = value
		}
	}
	return out
}
