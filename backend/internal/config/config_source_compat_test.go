package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 两个默认安装目录都能找到配置；显式文件和现有数据目录仍优先。
func TestTokenRouterConfigDirectoryCompatibility(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("DATA_DIR", "/tmp/byteseek-config-fixture")
	var file string
	var paths []string
	configureConfigSource(func(value string) { file = value }, func(value string) { paths = append(paths, value) })
	require.Empty(t, file)
	require.Equal(t, []string{"/tmp/byteseek-config-fixture", "/app/data", ".", "./config", "/etc/tokenrouter", "/etc/sub2api"}, paths)
	t.Setenv("CONFIG_FILE", "/tmp/byteseek-explicit.yaml")
	paths = nil
	configureConfigSource(func(value string) { file = value }, func(value string) { paths = append(paths, value) })
	require.Equal(t, "/tmp/byteseek-explicit.yaml", file)
	require.Empty(t, paths)
}
