package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
	"github.com/TokenFlux/TokenRouter/internal/routing/httpapi/dto"
)

// provideKeyHTTP 直接绑定 Key 用例与容量展示投影，不创建额外认证缓存。
func provideKeyHTTP(keys *apikey.APIKeyService, capacity *routing.CapacityService, catalogue *routing.RequestableCatalogue) *keyhttp.APIKeyHandler[dto.Group] {
	handler := keyhttp.NewAPIKeyHandler(keys, func(group *routing.Group, summary *accessview.GroupCapacitySummary) *dto.Group {
		result := dto.GroupFromRouting(apikey.RoutingGroup(group))
		if result != nil && summary != nil {
			result.Capacity = dto.GroupCapacityFromSummary(summary)
		}
		return result
	})
	handler.SetGroupCapacityService(capacity)
	handler.SetGroupModelsReader(func(ctx context.Context, id int64) ([]string, map[string][]protocol.ProtocolID) {
		result := catalogue.ResolveRequestableModels(ctx, &id, "")
		models := make([]string, 0, len(result.Models))
		protocols := make(map[string][]protocol.ProtocolID)
		for _, model := range result.Models {
			models = append(models, model.ID)
			protocols[model.ID] = model.Protocols
		}
		return models, protocols
	})
	return handler
}

// provideKeyAdminHTTP 直接注入 Key 管理用例，管理展示不经过旧实体往返转换。
func provideKeyAdminHTTP(keys *apikey.Admin) *keyhttp.AdminAPIKeyHandler[dto.Group] {
	return keyhttp.NewAdminAPIKeyHandler(keys, func(group *routing.Group) *dto.Group {
		return dto.GroupFromRouting(apikey.RoutingGroup(group))
	})
}
