package provider

import "context"

// WriteStore 是Provider管理编辑的最小写入端口。
// 批量字段/分组/凭据的更宽接口后续按最终Provider数据模型拆分，不能把旧AccountRepository整包暴露。
type WriteStore interface {
	Update(context.Context, *Record) error
	UpdateExtra(context.Context, int64, map[string]any) error
}
