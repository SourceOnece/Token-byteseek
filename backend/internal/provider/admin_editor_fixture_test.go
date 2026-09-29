package provider_test

import (
	"time"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/google/uuid"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

// newProviderEditorForTest 保留旧编辑入口的时间、指纹种子与平台凭据校验注入。
func newProviderEditorForTest(repo providercore.AdminStore, groupPorts ...providercore.AdminGroups) *providercore.Admin {
	var groups providercore.AdminGroups
	if len(groupPorts) > 0 {
		groups = groupPorts[0]
	}
	quota, _ := repo.(providercore.ProviderQuotaResetter)
	duplicates, _ := repo.(providercore.DuplicateStore)
	return providercore.NewAdmin(repo, providercore.AdminOptions{Duplicates: duplicates, Groups: groups, Quotas: quota, ShadowModels: provideradapter.DefaultSparkShadowModels, Creation: providercore.CreationOptions{Now: time.Now, LoadLocation: time.LoadLocation, NewSeed: uuid.NewString}, Credentials: provideradapter.CreateCredentialHooks(nil, nil)})
}
