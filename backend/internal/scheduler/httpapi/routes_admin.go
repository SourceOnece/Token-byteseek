package httpapi

import "github.com/gin-gonic/gin"

// RegisterProviderDiagnostics 在提供商路径下注册唯一评分诊断，组权限由 app 安装。
func RegisterProviderDiagnostics(providers *gin.RouterGroup, endpoint *DiagnosticsHandler) {
	providers.GET("/:id/advanced-scheduler-score", endpoint.GetAdvancedSchedulerScore)
	providers.POST("/:id/advanced-scheduler-score/preview", endpoint.PreviewAdvancedSchedulerScore)
}
