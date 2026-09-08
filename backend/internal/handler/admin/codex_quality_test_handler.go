package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/response"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
)

// BatchCodexQualityTest 仅通过管理员路由执行，显式确认后才可能消耗上游额度并修改调度。
func (h *AccountHandler) BatchCodexQualityTest(c *gin.Context) {
	var req service.CodexQualityRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "测试参数格式无效或请求过大")
		return
	}
	if err := req.Normalize(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if _, ok := h.accountTestService.CodexQualityRepository(); !ok {
		response.Error(c, 503, "测试服务暂不可用")
		return
	}
	if !h.accountTestService.BeginCodexQualityBatch() {
		response.Error(c, 409, "本实例正在执行另一批测试，请稍后重试")
		return
	}
	defer h.accountTestService.EndCodexQualityBatch()
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()
	write := func(kind string, data any) bool {
		payload, err := json.Marshal(map[string]any{"type": kind, "data": data})
		if err != nil {
			return false
		}
		if _, err = fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
			cancel()
			return false
		}
		c.Writer.Flush()
		return true
	}
	if !write("start", map[string]int{"total": len(req.AccountIDs)}) {
		return
	}
	jobs := make(chan int64)
	results := make(chan *service.CodexQualityResult, req.Concurrency)
	var workers sync.WaitGroup
	for i := 0; i < req.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}
				result := h.accountTestService.RunCodexQualityTest(ctx, id, &req)
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, id := range req.AccountIDs {
			select {
			case jobs <- id:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { workers.Wait(); close(results) }()
	// 退出时等在途测试保存取消状态，避免已释放批次锁但旧任务仍在运行。
	defer func() { cancel(); workers.Wait() }()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	counts := map[string]int{"full": 0, "degraded": 0, "failed": 0, "skipped": 0, "cancelled": 0, "stale": 0}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		case result, ok := <-results:
			if !ok {
				write("complete", counts)
				return
			}
			counts[result.Status]++
			if !write("result", result) {
				return
			}
		}
	}
}

// ListCodexQualityResults 按当前页账号读取最近结果，和账号列表分页独立，避免携带长回答进入网关缓存。
func (h *AccountHandler) ListCodexQualityResults(c *gin.Context) {
	parts := strings.Split(c.Query("account_ids"), ",")
	if len(parts) == 0 || len(parts) > 500 {
		response.BadRequest(c, "每次最多查询 500 个账号")
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "账号 ID 无效")
			return
		}
		ids = append(ids, id)
	}
	repo, ok := h.accountTestService.CodexQualityRepository()
	if !ok {
		response.Error(c, 503, "测试服务暂不可用")
		return
	}
	results, err := repo.ListCodexQualityResults(c.Request.Context(), ids, c.Query("detail") == "true" && len(ids) == 1)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取测试结果失败")
		return
	}
	response.Success(c, results)
}
