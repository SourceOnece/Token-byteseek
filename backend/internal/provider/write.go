package provider

import "context"

// WriteStore 是Provider管理编辑的最小写入端口。
// 字段批量与凭据分别保持专用方法，不能把旧AccountRepository整包暴露。
type WriteStore interface {
	Update(context.Context, *Record) error
	UpdateExtra(context.Context, int64, map[string]any) error
	UpdateCredentials(context.Context, int64, map[string]any) error
	BulkUpdate(context.Context, []int64, BulkUpdate) (int64, error)
}
