//go:build unit

// 测试私有适配入口，委托所属模块的生产用例。
package middleware

import (
	"github.com/TokenFlux/TokenRouter/internal/apikey"

	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"

	gin "github.com/gin-gonic/gin"
)

const maxAPIKeyAuthorizationHeaderBytes = apikey.MaxAPIKeyCredentialBytes + 128

func abortTeamAPIKeyError(c *gin.Context, err error) bool { return keyhttp.AbortTeamError(c, err) }
