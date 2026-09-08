// Package service 实现 Codex 管理员题目测试与调度联动。
package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TokenFlux/TokenRouter/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const codexQualityContextKey accountTestContextKey = "codex_quality_test"

// CodexQualityRequest 是专用批量题目测试配置，不改变原连接测试接口。
type CodexQualityRequest struct {
	AccountIDs        []int64 `json:"account_ids"`
	Model             string  `json:"model"`
	ReasoningEffort   string  `json:"reasoning_effort"`
	APIProtocol       string  `json:"api_protocol"`
	Prompt            string  `json:"prompt"`
	Keyword           string  `json:"keyword"`
	Concurrency       int     `json:"concurrency"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	ConfirmScheduling bool    `json:"confirm_scheduling"`
}

// Normalize 校验失败时尚未发送请求，也不修改任何调度状态。
func (r *CodexQualityRequest) Normalize() error {
	r.Model = strings.TrimSpace(r.Model)
	r.Prompt = strings.TrimSpace(r.Prompt)
	r.Keyword = strings.TrimSpace(r.Keyword)
	if strings.ContainsRune(r.Model+r.Prompt+r.Keyword, '\x00') {
		return errors.New("输入不能包含空字符")
	}
	if !r.ConfirmScheduling {
		return errors.New("必须确认本次测试将自动修改所选账号的调度开关")
	}
	if len(r.AccountIDs) == 0 || len(r.AccountIDs) > 500 {
		return errors.New("每批请选择 1 到 500 个账号")
	}
	seen := map[int64]bool{}
	for _, id := range r.AccountIDs {
		if id <= 0 || seen[id] {
			return errors.New("账号 ID 必须为不重复的正整数")
		}
		seen[id] = true
	}
	if r.Model == "" || len(r.Model) > 200 || strings.ContainsAny(r.Model, "\r\n") {
		return errors.New("请选择有效的模型")
	}
	if r.Prompt == "" || utf8.RuneCountInString(r.Prompt) > 16000 {
		return errors.New("题目长度须为 1 到 16000 字符")
	}
	if r.Keyword == "" || utf8.RuneCountInString(r.Keyword) > 200 {
		return errors.New("关键词长度须为 1 到 200 字符")
	}
	switch r.ReasoningEffort {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return errors.New("不支持的思考等级")
	}
	switch r.APIProtocol {
	case "", string(openai_compat.TextProtocolResponses), string(openai_compat.TextProtocolChatCompletions):
	default:
		return errors.New("请选择一个受支持的 API 协议")
	}
	if r.Concurrency == 0 {
		r.Concurrency = 3
	}
	if r.Concurrency < 1 || r.Concurrency > 5 {
		return errors.New("测试并发须为 1 到 5")
	}
	if r.TimeoutSeconds == 0 {
		r.TimeoutSeconds = 120
	}
	if r.TimeoutSeconds < 10 || r.TimeoutSeconds > 3600 {
		return errors.New("单账号超时须为 10–3600 秒")
	}
	return nil
}

// CodexQualityResult 只保存管理员测试输入和可见回答，不保存推理内容与凭据。
type CodexQualityResult struct {
	AccountID         int64     `json:"account_id"`
	Email             string    `json:"email"`
	AccountName       string    `json:"account_name"`
	Model             string    `json:"model"`
	ReasoningEffort   string    `json:"reasoning_effort"`
	APIProtocol       string    `json:"api_protocol"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Prompt            string    `json:"prompt"`
	Keyword           string    `json:"keyword"`
	Status            string    `json:"status"`
	ResponseText      string    `json:"response_text"`
	Error             string    `json:"error,omitempty"`
	StartedAt         time.Time `json:"started_at"`
	FinishedAt        time.Time `json:"finished_at"`
	SchedulingApplied bool      `json:"scheduling_applied"`
	Schedulable       bool      `json:"schedulable"`
}

// CodexQualityRepository 用窄接口保持现有账号仓储测试替身兼容。
type CodexQualityRepository interface {
	AcquireCodexQualityTest(context.Context, int64, string, int) (bool, error)
	FinishCodexQualityTest(context.Context, *Account, string, *CodexQualityResult) (bool, error)
	ListCodexQualityResults(context.Context, []int64, bool) ([]*CodexQualityResult, error)
}

func (s *AccountTestService) CodexQualityRepository() (CodexQualityRepository, bool) {
	if s == nil {
		return nil, false
	}
	r, ok := s.accountRepo.(CodexQualityRepository)
	return r, ok
}

// 批量入口限制本实例至多一个批次；账号租约负责跨实例的同账号互斥。
func (s *AccountTestService) BeginCodexQualityBatch() bool {
	return s.qualityBatchActive.CompareAndSwap(false, true)
}
func (s *AccountTestService) EndCodexQualityBatch() { s.qualityBatchActive.Store(false) }

// qualityTestOptions 仅通过服务端上下文启用，不能由普通测试请求误触发。
func qualityTestOptions(ctx context.Context) *CodexQualityRequest {
	r, _ := ctx.Value(codexQualityContextKey).(*CodexQualityRequest)
	return r
}

// IsOpenAIQualityTestable 统一限定质量检测的账号边界，兼容 OAuth 与 API Key 上游。
// 影子账号和 Agent Identity 没有独立可验证的凭据，仍由主账号/专用流程处理。
func IsOpenAIQualityTestable(account *Account) bool {
	if account == nil || account.Platform != PlatformOpenAI {
		return false
	}
	if account.Type != AccountTypeOAuth && account.Type != AccountTypeAPIKey {
		return false
	}
	return !account.IsCredentialShadow() && !account.IsOpenAIAgentIdentity()
}

// 一次检测只使用一个协议；未指定的旧计划保留账号原协议选择，不自动轮测。
func resolveQualityTextProtocol(account *Account, options *CodexQualityRequest) openai_compat.TextProtocol {
	if account.IsOpenAIOAuth() {
		return openai_compat.TextProtocolResponses
	}
	if options != nil && options.APIProtocol != "" {
		return openai_compat.TextProtocol(options.APIProtocol)
	}
	return openai_compat.ResolveUpstreamTextProtocol(account.Extra, openai_compat.TextProtocolResponses)
}

// RunCodexQualityTest 复用 OAuth 测试链路，并在完成后原子写入判定与调度状态。
// @project-doc docs/operations/account_maintenance.md#codex_quality_testing
func (s *AccountTestService) RunCodexQualityTest(ctx context.Context, id int64, options *CodexQualityRequest) *CodexQualityResult {
	timeout := options.TimeoutSeconds
	if timeout == 0 {
		timeout = 120
	}
	result := &CodexQualityResult{AccountID: id, Model: options.Model, ReasoningEffort: options.ReasoningEffort,
		Prompt: options.Prompt, Keyword: options.Keyword, Status: "skipped", StartedAt: time.Now(), TimeoutSeconds: timeout}
	finish := func(message string) *CodexQualityResult {
		result.Error = message
		result.FinishedAt = time.Now()
		return result
	}
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil || account == nil {
		return finish("账号不存在或已删除")
	}
	result.AccountName = account.Name
	result.Email = firstStringValue(account.Credentials, "email")
	if result.Email == "" {
		result.Email = firstStringValue(account.Extra, "email", "email_address")
	}
	if !IsOpenAIQualityTestable(account) {
		return finish("仅支持独立 OpenAI OAuth 或 API Key 上游；不测试其他平台、影子或 Agent Identity 账号")
	}
	result.APIProtocol = string(resolveQualityTextProtocol(account, options))
	repo, ok := s.CodexQualityRepository()
	if !ok {
		return finish("测试结果存储不可用")
	}
	runID := uuid.NewString()
	acquired, err := repo.AcquireCodexQualityTest(ctx, id, runID, timeout)
	if err != nil {
		return finish("无法取得测试租约，请稍后重试")
	}
	if !acquired {
		return finish("该账号正在其他批次测试，请勿重复提交")
	}

	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	testCtx = context.WithValue(withAccountTestUserAgent(testCtx, ""), codexQualityContextKey, options)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/quality-test", nil).WithContext(testCtx)
	// 使用已校验的账号快照，避免测试中途替换凭据后把结果归给新账号。
	var testErr error
	if testErr = s.prepareOpenAIAutomaticProbe(c, account); testErr == nil {
		testErr = s.testOpenAIAccountConnection(c, account, options.Model, options.Prompt, AccountTestModeDefault, AccountTestTypeText)
	}
	result.ResponseText, result.Error = parseTestSSEOutput(recorder.Body.String())
	if testErr != nil {
		result.Error = testErr.Error()
	}
	result.Status = "failed"
	if result.Error == "" && strings.TrimSpace(result.ResponseText) != "" {
		result.Status = "degraded"
		if strings.Contains(result.ResponseText, options.Keyword) {
			result.Status = "full"
		}
	} else if result.Error == "" {
		result.Error = "上游没有返回可见回答"
	}
	if ctx.Err() != nil {
		result.Status = "cancelled"
		result.Error = "测试已取消，未修改调度"
	}
	result.FinishedAt = time.Now()
	// 取消或响应连接断开后仍用短上下文保存结果，不再发起上游请求。
	saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer saveCancel()
	// 正常用量/最后使用时间也会更新 updated_at。配置未变时刷新 CAS 基准，
	// 既不被正常流量误伤，也不覆盖测试期间管理员更换的凭据与调度开关。
	current, readErr := s.accountRepo.GetByID(saveCtx, id)
	if readErr == nil && sameQualityAccountConfiguration(account, current) {
		account.UpdatedAt = current.UpdatedAt
	} else if result.Status != "cancelled" {
		result.Status = "stale"
		result.Error = "账号配置或代理已变化，未修改调度，请重新测试"
	}
	applied, saveErr := repo.FinishCodexQualityTest(saveCtx, account, runID, result)
	if saveErr != nil {
		result.Status = "failed"
		result.SchedulingApplied = false
		result.Error = "结果保存失败，调度状态未确认，请刷新账号检查"
		return result
	}
	if !applied && result.Status != "cancelled" {
		result.Status = "stale"
		result.Error = "测试期间账号已变化，结果未应用调度，请重新测试"
	}
	return result
}

func sameQualityAccountConfiguration(before, after *Account) bool {
	if before == nil || after == nil {
		return false
	}
	identity := func(account *Account) string {
		extra := map[string]any{}
		for key, value := range account.Extra {
			if strings.HasPrefix(key, "codex_primary_") || strings.HasPrefix(key, "codex_secondary_") ||
				strings.HasPrefix(key, "codex_5h_") || strings.HasPrefix(key, "codex_7d_") ||
				strings.HasPrefix(key, "codex_reset_credit_") || strings.HasPrefix(key, "passive_usage_") ||
				key == "codex_usage_updated_at" || key == "model_rate_limits" {
				continue
			}
			extra[key] = value
		}
		body, _ := json.Marshal([]any{account.Credentials, extra, account.Status, account.Schedulable,
			account.ProxyID, account.ParentAccountID, account.Concurrency, account.ExpiresAt, account.AutoPauseOnExpired})
		return string(body)
	}
	if before.Platform != after.Platform || before.Type != after.Type || identity(before) != identity(after) {
		return false
	}
	return reflect.DeepEqual(before.Proxy, after.Proxy)
}

func qualityStreamText(event map[string]any) (string, bool, error) {
	kind, _ := event["type"].(string)
	switch kind {
	case "response.output_text.delta":
		text, _ := event["delta"].(string)
		return text, false, nil
	case "response.completed", "response.done":
		response, _ := event["response"].(map[string]any)
		if status, _ := response["status"].(string); status != "" && status != "completed" {
			return "", false, fmt.Errorf("上游响应未完成：%s", status)
		}
		var text strings.Builder
		items, _ := response["output"].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			if item["type"] != "message" || item["role"] != "assistant" {
				continue
			}
			contents, _ := item["content"].([]any)
			for _, rawContent := range contents {
				content, _ := rawContent.(map[string]any)
				if content["type"] == "output_text" {
					value, _ := content["text"].(string)
					text.WriteString(value)
				}
			}
		}
		return text.String(), true, nil
	case "response.incomplete", "response.failed", "error":
		return "", false, errors.New("上游响应失败或未完整结束")
	}
	return "", false, nil
}

// processCodexQualityStream 不接受半截回答，也不把推理摘要、错误或状态提示当作答案。
func (s *AccountTestService) processCodexQualityStream(c *gin.Context, body io.Reader) error {
	return s.processQualitySSE(c, body, func(data string) (string, bool, error) {
		if data == "[DONE]" {
			return "", false, errors.New("回答未收到完成事件")
		}
		var event map[string]any
		if json.Unmarshal([]byte(data), &event) != nil {
			return "", false, errors.New("上游 SSE 格式无效")
		}
		return qualityStreamText(event)
	})
}

// 两种协议共用有界 SSE 事件读取，保留跨行 JSON 和断流保护。
func (s *AccountTestService) processQualitySSE(c *gin.Context, body io.Reader, decode func(string) (string, bool, error)) error {
	// 包含推理等非答案事件在内的读取总量也受限，防止异常流耗尽内存。
	scanner := bufio.NewScanner(io.LimitReader(body, 8*1024*1024))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var answer strings.Builder
	var eventData []string
	eventBytes := 0
	consume := func() (bool, error) {
		if len(eventData) == 0 {
			return false, nil
		}
		data := strings.Join(eventData, "\n")
		eventData = nil
		eventBytes = 0
		text, done, err := decode(data)
		if err != nil {
			return false, err
		}
		if done && text != "" {
			answer.Reset()
		}
		answer.WriteString(strings.ReplaceAll(text, "\x00", ""))
		if answer.Len() > 128*1024 {
			return false, errors.New("回答超过 128 KiB 测试上限")
		}
		if done {
			s.sendEvent(c, TestEvent{Type: "content", Text: answer.String()})
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		}
		return done, nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			done, err := consume()
			if err != nil {
				return s.sendErrorAndEnd(c, err.Error())
			}
			if done {
				return nil
			}
		} else if strings.HasPrefix(line, "data:") {
			eventData = append(eventData, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			eventBytes += len(line)
		}
		if len(eventData) > 1024 || eventBytes > 1024*1024 {
			return s.sendErrorAndEnd(c, "上游 SSE 事件过大")
		}
	}
	if err := scanner.Err(); err != nil {
		return s.sendErrorAndEnd(c, "读取上游响应失败")
	}
	done, err := consume()
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	if done {
		return nil
	}
	return s.sendErrorAndEnd(c, "回答未完整结束")
}
