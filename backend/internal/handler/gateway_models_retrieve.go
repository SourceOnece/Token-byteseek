package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 单项发现复用最终可见列表，保留别名、复合前缀及元数据，不直接查询任意上游模型。
func writeModelsListResponse(c *gin.Context, models any) {
	response := gin.H{"object": "list", "data": models}
	modelID := strings.TrimPrefix(c.Param("model"), "/")
	if c.Param("model") == "" {
		c.JSON(http.StatusOK, response)
		return
	}
	body, err := json.Marshal(response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "Failed to encode model catalogue"}})
		return
	}
	var catalogue struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &catalogue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "Invalid model catalogue"}})
		return
	}
	for _, raw := range catalogue.Data {
		var item struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "Invalid model catalogue entry"}})
			return
		}
		if item.ID == modelID {
			c.Data(http.StatusOK, "application/json", raw)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
		"type": "invalid_request_error", "code": "model_not_found", "param": "model",
		"message": fmt.Sprintf("Model %q does not exist or is not available for this group", modelID),
	}})
}
