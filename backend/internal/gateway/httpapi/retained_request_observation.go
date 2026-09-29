package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/ops"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// 单模型读取也从完整授权目录取值，不能提前裁掉复合 Key 的模型前缀。
func isStandardModelRetrieveEndpoint(method, path string) bool {
	return method == http.MethodGet && (strings.HasPrefix(path, "/v1/models/") || strings.HasPrefix(path, "/models/"))
}

// 客户端取消前已有上游失败时仍保留故障记录。
func shouldSkipOpsClientClosed(c *gin.Context, service *ops.OpsService, status int) bool {
	return status == StatusClientClosedRequest && service != nil && service.OpsAdvancedSettingsSnapshot().IgnoreContextCanceled && !hasOpsUpstreamErrorContext(c)
}
