//go:build unit

// 测试私有适配入口，委托所属模块的生产用例。
package middleware

import (
	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
)

func googleTeamAPIKeyError(err error) (int, string, bool) { return keyhttp.GoogleTeamError(err) }
