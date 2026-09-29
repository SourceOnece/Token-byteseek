package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/identity/httpapi/authctx"
	"github.com/gin-gonic/gin"
)

// adminID 返回当前订阅管理操作的审计主体。
func adminID(c *gin.Context) int64 {
	subject, ok := authctx.GetAuthSubjectFromContext(c)
	if !ok {
		return 0
	}
	return subject.UserID
}
