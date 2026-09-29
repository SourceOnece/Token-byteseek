package provider

import (
	"context"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

type providerCredentialsUpdater interface {
	UpdateCredentials(context.Context, int64, map[string]any) error
}
type executionCredentialStore struct {
	source   ExecutionProviderStore
	original *ExecutionProvider
}

func (s executionCredentialStore) Update(ctx context.Context, value *provider.Record) error {
	s.original.Record.Credentials = value.Credentials
	return s.source.Update(ctx, s.original)
}

type executionCredentialFields struct {
	executionCredentialStore
	updater providerCredentialsUpdater
}

func (s executionCredentialFields) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	s.original.Record.Credentials = credentials
	return s.updater.UpdateCredentials(ctx, id, credentials)
}

// PersistExecutionCredentials 只投影专用字段及旧调用者的赋值时机，所有写入规则由 provider 提供。
func PersistExecutionCredentials(ctx context.Context, repo ExecutionProviderStore, value *ExecutionProvider, credentials map[string]any) error {
	if repo == nil || value == nil {
		return nil
	}
	var store provider.CredentialUpdateStore = executionCredentialStore{source: repo, original: value}
	if updater, ok := repo.(providerCredentialsUpdater); ok {
		store = executionCredentialFields{executionCredentialStore: executionCredentialStore{source: repo, original: value}, updater: updater}
	}
	view := &provider.Record{ID: value.Record.ID, Platform: value.Record.Platform, Type: value.Record.Type, ParentProviderID: value.Record.ParentProviderID, QuotaDimension: value.Record.QuotaDimension}
	changed, err := provider.PersistCredentials(ctx, store, view, credentials, slog.Warn)
	if changed {
		value.Record.Credentials = view.Credentials
	}
	return err
}
