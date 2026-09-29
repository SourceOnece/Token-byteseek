// 根据审核查询总数和分页参数构造分页结果。
package postgres

import (
	"github.com/TokenFlux/TokenRouter/internal/pkg/pagination"
)

func paginationResultFromTotal(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	return pagination.ResultFromTotal(total, params)
}
