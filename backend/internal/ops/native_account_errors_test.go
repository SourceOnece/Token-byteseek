package ops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 登录用户和上游原始错误里的 account 不是本地接入实体，保留原文才能正确分类。
func TestNativeAccountErrorsKeepTheirClassification(t *testing.T) {
	require.True(t, isOpsClientAuthError("", "user account is not active"))
	require.True(t, isOpsLocalBusinessLimitError("", "insufficient account balance"))
}
