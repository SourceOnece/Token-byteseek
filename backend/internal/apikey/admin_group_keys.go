// 管理端按分组分页读取 API Key，并返回匹配总数。
package apikey

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/pkg/pagination"
)

func (s *Admin) GetGroupAPIKeys(ctx context.Context, groupID int64, page, pageSize int) ([]APIKey, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	keys, result, err := s.Keys.ListByGroupID(ctx, groupID, params)
	if err != nil {
		return nil, 0, err
	}
	return keys, result.Total, nil
}
