package httpapi

import (
	"context"
	"strconv"

	idempotencyhttp "github.com/TokenFlux/TokenRouter/internal/idempotency/httpapi"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	response "github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
)

// ProviderManagement 只表达 HTTP 使用的提供商用例，不暴露具体存储。
type ProviderManagement interface {
	BulkUpdateProviders(context.Context, *providercore.BulkUpdateProvidersInput) (*providercore.BulkUpdateProvidersResult, error)
	SetProviderSchedulable(context.Context, int64, bool) (*providercore.Record, error)
	ResetProviderQuota(context.Context, int64) error
	RevertProviderProxyFallback(context.Context, int64) error
	GetProvidersByIDs(context.Context, []int64) ([]*providercore.Record, error)
	CreateProvider(context.Context, *providercore.CreateProviderInput) (*providercore.Record, error)
	UpdateProvider(context.Context, int64, *providercore.UpdateProviderInput) (*providercore.Record, error)
	DuplicateProvider(context.Context, int64, string, string) (*providercore.Record, error)
	RecoverDuplicateProvider(context.Context, int64, string, string) (*providercore.Record, error)
	GetProvider(context.Context, int64) (*providercore.Record, error)
	DeleteProvider(context.Context, int64) error
}
type ProviderRuntimePresenter interface {
	Present(context.Context, *providercore.Record) ProviderWithConcurrency
}
type ProviderRuntimePresenterFunc func(context.Context, *providercore.Record) ProviderWithConcurrency

func (f ProviderRuntimePresenterFunc) Present(ctx context.Context, v *providercore.Record) ProviderWithConcurrency {
	return f(ctx, v)
}

// ManagementHandler 保留原管理员 HTTP 契约，运行投影与 Ollama 用量通过已装配端口取得。
type ManagementHandler struct {
	CustomFilters func(context.Context, string, string) (context.Context, error)
	idempotencyhttp.Executor

	models           *providercore.ModelSyncService
	reports          ProviderReportOptions
	tier             *providercore.TierManagement
	catalog          *routing.AdminCatalog
	modelDefaults    providercore.ModelMappingDefaults
	listing          *providercore.ManagementList
	runtimePresenter *RuntimePresenter
	recovery         *providercore.RecoveryService
	batch            *providercore.ManagementBatch
	managed          *providercore.ManagedRefreshService
	adminService     ProviderManagement
	presenter        ProviderRuntimePresenter
	ollamaCloudUsage *providercore.OllamaCloudUsageService
	privacy          ProviderCreationPrivacy
	afterCreate      func(*providercore.Record)
}

type ProviderCreationPrivacy interface {
	ForceAntigravityPrivacy(context.Context, *providercore.Record) string
	ForceOpenAIPrivacy(context.Context, *providercore.Record) string
}
type ManagementOptions struct {
	CustomFilters    func(context.Context, string, string) (context.Context, error)
	Models           *providercore.ModelSyncService
	Reports          ProviderReportOptions
	Tier             *providercore.TierManagement
	Catalog          *routing.AdminCatalog
	ModelDefaults    providercore.ModelMappingDefaults
	List             *providercore.ManagementList
	RuntimePresenter *RuntimePresenter
	Recovery         *providercore.RecoveryService
	Batch            *providercore.ManagementBatch
	Managed          *providercore.ManagedRefreshService
	Presenter        ProviderRuntimePresenter
	Ollama           *providercore.OllamaCloudUsageService
	Privacy          ProviderCreationPrivacy
	AfterCreate      func(*providercore.Record)
}

func NewManagementHandler(admin ProviderManagement, options ManagementOptions) *ManagementHandler {
	return &ManagementHandler{CustomFilters: options.CustomFilters, models: options.Models, reports: options.Reports, tier: options.Tier, catalog: options.Catalog, modelDefaults: options.ModelDefaults, listing: options.List, runtimePresenter: options.RuntimePresenter, recovery: options.Recovery, batch: options.Batch, managed: options.Managed, adminService: admin, presenter: options.Presenter, ollamaCloudUsage: options.Ollama, privacy: options.Privacy, afterCreate: options.AfterCreate}
}

// GetByID 按原状态码与展示流程查询提供商。
// GET /api/v1/admin/providers/:id
func (h *ManagementHandler) GetByID(c *gin.Context) {
	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid provider ID")
		return
	}

	provider, err := h.adminService.GetProvider(c.Request.Context(), providerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.ollamaCloudUsage != nil {
		if err := h.ollamaCloudUsage.ResolveProviders(c.Request.Context(), []*providercore.Record{provider}); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	response.Success(c, h.presenter.Present(c.Request.Context(), provider))
}

// Delete 删除提供商并保持原确认响应。
// DELETE /api/v1/admin/providers/:id
func (h *ManagementHandler) Delete(c *gin.Context) {
	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid provider ID")
		return
	}

	err = h.adminService.DeleteProvider(c.Request.Context(), providerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Provider deleted successfully"})
}
