package httpapi

import (
	"context"
	"errors"

	"github.com/TokenFlux/TokenRouter/internal/gateway/clientmeta"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// PrepareMessageClientContext 在路由与提供商选择前解析客户端、探针和 thinking 的可信请求内状态。
func PrepareMessageClientContext(c *gin.Context, body []byte, bounds func(context.Context) (string, string)) error {
	// 身份探测沿用 Messages 入口的宽松 stream 读取；仅规范化解析副本，出站报文不变。
	identityBody := body
	if value := gjson.GetBytes(body, "stream"); value.Exists() && value.Type != gjson.True && value.Type != gjson.False {
		var err error
		identityBody, err = sjson.SetBytes(body, "stream", value.Bool())
		if err != nil {
			return err
		}
	}
	parsed, err := requeststate.ParseGatewayRequest(requeststate.NewRequestBodyRef(identityBody), "anthropic")
	if err != nil {
		return err
	}
	probe := clientmeta.IsHaikuProbe(parsed.Model, parsed.MaxTokens)
	ctx := requeststate.WithIsMaxTokensOneHaikuRequest(c.Request.Context(), probe)
	c.Request = c.Request.WithContext(ctx)
	detected := DetectClaudeCodeRequest(c, body, parsed, probe)
	ctx = requeststate.SetClaudeCodeClient(ctx, detected.ClaudeCode)
	ctx = requeststate.SetClaudeCodeVersion(ctx, detected.Version)
	ctx = requeststate.WithThinkingEnabled(ctx, parsed.ThinkingEnabled)
	c.Request = c.Request.WithContext(ctx)
	if detected.ClaudeCode && bounds != nil {
		minimum, maximum := bounds(ctx)
		if message := clientmeta.ClaudeVersionRejection(detected.Version, minimum, maximum); message != "" {
			return errors.New(message)
		}
	}
	return nil
}
