package claude

import (
	"golang.org/x/mod/semver"
	"log/slog"
	"os"
	"strings"
)

// CLIVersionEnv 是 Claude CLI 版本覆盖变量，进程启动时只读取一次。
const CLIVersionEnv = "SUB2API_CLAUDE_CLI_VERSION"

var resolvedCLIVersion = resolveCLIVersion(os.Getenv(CLIVersionEnv))

// CLIVersion 返回当前进程使用的 Claude CLI 版本。
func CLIVersion() string { return resolvedCLIVersion }

// IsSupportedCLIVersion 只接受不低于内置版本的三段纯数字版本。
func IsSupportedCLIVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	canonical := "v" + version
	if !semver.IsValid(canonical) || semver.Canonical(canonical) != canonical || semver.Prerelease(canonical) != "" || semver.Build(canonical) != "" {
		return false
	}
	return semver.Compare(canonical, "v"+CLICurrentVersion) >= 0
}

func resolveCLIVersion(raw string) string {
	version := strings.TrimSpace(raw)
	if version == "" {
		return CLICurrentVersion
	}
	if !IsSupportedCLIVersion(version) {
		slog.Warn("invalid Claude CLI version override; using built-in version", "env", CLIVersionEnv, "builtin", CLICurrentVersion)
		return CLICurrentVersion
	}
	return version
}
