// Slice 按分页参数截取内存集合，并约束切片边界。
package pagination

func Slice[T any](items []T, params PaginationParams) []T {
	if len(items) == 0 {
		return []T{}
	}

	offset := params.Offset()
	if offset >= len(items) {
		return []T{}
	}

	limit := params.Limit()
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}

	return items[offset:end]
}
