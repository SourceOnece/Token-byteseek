package provider

import "context"

// CredentialUpdateStore 是凭据专用写入端口，避免凭据刷新覆盖并发修改的其它字段。
type CredentialUpdateStore interface {
	Update(context.Context, *Record) error
}

type CredentialFieldsUpdater interface {
	UpdateCredentials(context.Context, int64, map[string]any) error
}

// PersistCredentials 保留影子不持凭据和专用更新时机；普通配置仍由上层CAS/事务控制。
func PersistCredentials(ctx context.Context, store CredentialUpdateStore, value *Record, credentials map[string]any, warn func(string, ...any)) (bool, error) {
	if store == nil || value == nil {
		return false, nil
	}
	if value.ParentProviderID != nil {
		if warn != nil {
			warn("skip persisting credentials to provider shadow", "provider_id", value.ID, "parent_id", *value.ParentProviderID)
		}
		return false, nil
	}
	if credentials == nil {
		credentials = map[string]any{}
	}
	value.Credentials = cloneCredentialMap(credentials)
	if updater, ok := store.(CredentialFieldsUpdater); ok {
		return true, updater.UpdateCredentials(ctx, value.ID, value.Credentials)
	}
	return true, store.Update(ctx, value)
}

func cloneCredentialMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
