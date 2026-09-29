//go:build integration

package billing_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func uniqueTeamTestEmail(prefix string) string {
	return fmt.Sprintf("team-%s-%s@example.com", prefix, uuid.NewString())
}

// uniqueTestValue 为兑换存储测试生成格式固定的唯一名称。
func uniqueTestValue(t *testing.T, prefix string) string {
	t.Helper()
	safeName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	return fmt.Sprintf("%s-%s", prefix, safeName)
}
