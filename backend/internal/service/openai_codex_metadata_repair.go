package service

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const codexMetadataRepairExtraKey = "codex_metadata_repair_enabled"
const codexMetadataRepairContextKey = "codex_metadata_repair_snapshot"
const codexMetadataRepairMaxBytes = 16 * 1024

// IsCodexMetadataRepairEnabled 只接受显式布尔 true，不改变存量账号或其它平台。
// @project-doc docs/interfaces/openai_upstream.md#codex_metadata_repair
func (a *Account) IsCodexMetadataRepairEnabled() bool {
	if !a.IsOpenAIOAuthLike() || a.Extra == nil {
		return false
	}
	enabled, _ := a.Extra[codexMetadataRepairExtraKey].(bool)
	return enabled
}

// 快照只保存本次请求的小型元数据，不保存提示词、认证或整包请求。
type codexMetadataRepairSnapshot struct {
	accountID int64
	request   *http.Request
	sourceKey [32]byte
	metadata  map[string]any
	headers   http.Header
}

var codexMetadataRepairIdentity = []struct {
	field   string
	aliases []string
}{
	{"installation_id", []string{"x-codex-installation-id", "installation_id"}},
	{"session_id", []string{"session_id", "session-id"}},
	{"thread_id", []string{"thread_id", "thread-id"}},
	{"turn_id", []string{"turn_id", "turn-id"}},
	{"window_id", []string{"x-codex-window-id", "window_id", "window-id"}},
}

func codexMetadataObject(raw string) map[string]any {
	if len(raw) == 0 || len(raw) > codexMetadataRepairMaxBytes {
		return nil
	}
	var value map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if !gjson.Valid(raw) || decoder.Decode(&value) != nil {
		return nil
	}
	return value
}

func codexMetadataString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// prepareCodexMetadataRepair 从尚未隔离的原始请求构造一次快照；关闭时清空旧快照，
// 防止 Gin context 跨账号 failover 把开启账号的标识带到关闭账号。
func prepareCodexMetadataRepair(c *gin.Context, account *Account, body []byte, sessionHint string) {
	if c == nil {
		return
	}
	previous := stagedCodexMetadataRepair(c, account)
	c.Set(codexMetadataRepairContextKey, (*codexMetadataRepairSnapshot)(nil))
	if !account.IsCodexMetadataRepairEnabled() || isOpenAIResponsesCompactPath(c) {
		return
	}
	// handler 的同账号重试会重新调用 Forward，不能因此重生成缺省 turn ID。
	sourceKey := sha256.Sum256([]byte(gjson.GetBytes(body, "client_metadata").Raw + "\x00" + sessionHint))
	if previous != nil && previous.request == c.Request && previous.sourceKey == sourceKey {
		c.Set(codexMetadataRepairContextKey, previous)
		return
	}
	snapshot := buildCodexMetadataRepair(c, account, body, sessionHint, true)
	if snapshot != nil {
		snapshot.request = c.Request
		snapshot.sourceKey = sourceKey
	}
	c.Set(codexMetadataRepairContextKey, snapshot)
}

// buildCodexMetadataRepair 的当前 body 优先于头；后续 WS 回合不复用握手的 turn
// 或环境扩展值，仅把稳定会话头作为缺失身份来源。不伪造安装设备、sandbox 或客户端能力。
func buildCodexMetadataRepair(c *gin.Context, account *Account, body []byte, sessionHint string, firstTurn bool) *codexMetadataRepairSnapshot {
	if c == nil || !account.IsCodexMetadataRepairEnabled() || isOpenAIResponsesCompactPath(c) {
		return nil
	}
	cm := map[string]any{}
	if existing := gjson.GetBytes(body, "client_metadata"); existing.Exists() && existing.Type != gjson.Null {
		cm = codexMetadataObject(existing.Raw)
		if cm == nil {
			return nil // 非对象或超限输入保留原协议处理，不悄悄删掉客户端数据。
		}
	}
	bodyTurnRaw := codexMetadataString(cm[openAIWSTurnMetadataHeader])
	if value, exists := cm[openAIWSTurnMetadataHeader]; exists && value != nil {
		if _, ok := value.(string); !ok {
			return nil
		}
	}
	bodyTurn := codexMetadataObject(bodyTurnRaw)
	if bodyTurnRaw != "" && bodyTurn == nil {
		return nil
	}
	headers := http.Header{}
	if c.Request != nil {
		headers = c.Request.Header
	}
	headerTurn := codexMetadataObject(headers.Get(openAIWSTurnMetadataHeader))
	if firstTurn && strings.TrimSpace(headers.Get(openAIWSTurnMetadataHeader)) != "" && headerTurn == nil {
		return nil
	}
	turn := map[string]any{}
	if firstTurn {
		for key, value := range headerTurn {
			turn[key] = value
		}
	}
	for key, value := range bodyTurn {
		turn[key] = value
	}
	for _, identity := range codexMetadataRepairIdentity {
		value := codexMetadataString(bodyTurn[identity.field])
		for _, alias := range identity.aliases {
			if value == "" {
				value = codexMetadataString(cm[alias])
			}
		}
		if value == "" && (firstTurn || identity.field != "turn_id") {
			value = codexMetadataString(headerTurn[identity.field])
			for _, alias := range identity.aliases {
				if value == "" {
					value = strings.TrimSpace(headers.Get(alias))
				}
			}
		}
		if value == "" && identity.field == "installation_id" {
			value = codexAccountIdentitySource(c, account).GetOpenAIDeviceID()
		}
		if value == "" && identity.field == "session_id" {
			value = strings.TrimSpace(sessionHint)
			if value == "" {
				value = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
			}
		}
		if value == "" && identity.field == "turn_id" {
			value = uuid.NewString() // 仅生成本请求的相关标识；不制造客户端设备/权限事实。
		}
		if value != "" {
			if strings.ContainsAny(value, "\r\n\x00") {
				return nil
			}
			turn[identity.field] = value
			cm[identity.aliases[0]] = value
		}
	}
	if _, exists := turn["turn_started_at_unix_ms"]; !exists {
		turn["turn_started_at_unix_ms"] = time.Now().UnixMilli()
	}
	turnJSON, err := json.Marshal(turn)
	if err != nil || len(turnJSON) > codexMetadataRepairMaxBytes {
		return nil
	}
	cm[openAIWSTurnMetadataHeader] = string(turnJSON)
	container := map[string]any{"client_metadata": cm}
	source := codexAccountIdentitySource(c, account)
	applyCodexAccountIdentityClientMetadataMap(container, source, getAPIKeyIDFromContext(c))
	// 与已有收敛模式共存，但快照每回合只派生一次，最终头与体直接复用它。
	clientHeaders := headers.Clone()
	if session := codexMetadataString(turn["session_id"]); session != "" {
		clientHeaders.Set("session-id", session)
	}
	applyCodexFingerprintClientMetadata(container, resolveCodexFingerprintIDsFromRequest(source, clientHeaders))
	cm = container["client_metadata"].(map[string]any)
	turn = codexMetadataObject(codexMetadataString(cm[openAIWSTurnMetadataHeader]))
	if turn == nil {
		return nil
	}
	outHeaders := make(http.Header)
	for _, identity := range codexMetadataRepairIdentity {
		value := codexMetadataString(cm[identity.aliases[0]])
		if value == "" {
			continue
		}
		turn[identity.field] = value
		for _, alias := range identity.aliases {
			outHeaders.Set(alias, value)
			// 保留原字段名，同时修正已存在的别名；不扩散多余 body 属性。
			if _, exists := cm[alias]; exists {
				cm[alias] = value
			}
		}
	}
	turnJSON, err = json.Marshal(turn)
	if err != nil || len(turnJSON) > codexMetadataRepairMaxBytes {
		return nil
	}
	cm[openAIWSTurnMetadataHeader] = string(turnJSON)
	encoded, err := json.Marshal(cm)
	if err != nil || len(encoded) > codexMetadataRepairMaxBytes {
		return nil
	}
	outHeaders.Set(openAIWSTurnMetadataHeader, string(turnJSON))
	return &codexMetadataRepairSnapshot{accountID: account.ID, metadata: cm, headers: outHeaders}
}

func stagedCodexMetadataRepair(c *gin.Context, account *Account) *codexMetadataRepairSnapshot {
	if c == nil || !account.IsCodexMetadataRepairEnabled() || isOpenAIResponsesCompactPath(c) {
		return nil
	}
	value, _ := c.Get(codexMetadataRepairContextKey)
	snapshot, _ := value.(*codexMetadataRepairSnapshot)
	if snapshot == nil || snapshot.accountID != account.ID {
		return nil
	}
	return snapshot
}

// applyRaw 仅替换小型 client_metadata，不解码大 input；快照不随重试再次派生。
func (snapshot *codexMetadataRepairSnapshot) applyRaw(body []byte) ([]byte, error) {
	if snapshot == nil {
		return body, nil
	}
	return sjson.SetBytes(body, "client_metadata", snapshot.metadata)
}

func (snapshot *codexMetadataRepairSnapshot) applyMap(body map[string]any) {
	if snapshot == nil || body == nil {
		return
	}
	// 深复制防止后续兼容转换原地写入而污染重试快照。
	raw, _ := json.Marshal(snapshot.metadata)
	body["client_metadata"] = codexMetadataObject(string(raw))
}

func (snapshot *codexMetadataRepairSnapshot) applyHeaders(headers http.Header) {
	if snapshot == nil || headers == nil {
		return
	}
	for key, values := range snapshot.headers {
		// 客户端 Header map 可能使用原始大小写，先清除全部变体避免重复身份头。
		for existing := range headers {
			if strings.EqualFold(existing, key) {
				delete(headers, existing)
			}
		}
		headers.Set(key, values[0])
	}
}
