package provider

import (
	"context"
)

type GroupReference struct {
	RequireOAuthOnly bool
	ID               int64
	Name             string
}
type AdminGroups interface {
	ActiveGroups(context.Context, string) ([]GroupReference, error)
	ValidateGroups(context.Context, []int64) error
	GetGroup(context.Context, int64) (*GroupReference, error)
}
type CreateCredentialHooks struct {
	// 平台端口保留 Qoder 站点切换与 PAT 校验的旧时序。
	Site         func(*Record) (string, error)
	ValidateEdit func(context.Context, *Record, bool) error
	Prepare      func(*Record)
	Validate     func(context.Context, *Record) error
}

// DuplicateStore 拥有原提供商与关联一次提交，不执行平台交换。
type DuplicateStore interface {
	CreateWithProviderGroups(context.Context, *Record, []GroupMembership) error
}

// ShadowProxyStore 保留原同步传播顺序，配置字段写权限继续收口。
type ShadowProxyStore interface {
	ListShadowsByParent(context.Context, int64) ([]*Record, error)
	Update(context.Context, *Record) error
}
