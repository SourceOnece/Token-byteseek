package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/net/http/httpguts"
)

const codexMetadataRepairExtraKey = "codex_metadata_repair_enabled"
const codexMetadataRepairContextKey = "codex_metadata_repair_snapshot"
const codexMetadataRepairMaxBytes = 16 * 1024

// 官方完整工具库存只在 body 承载，与兼容请求头分开限额。
const codexMetadataRepairBodyMaxBytes = 256 * 1024

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
	lineage   []codexMetadataLineageBinding
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
	if len(raw) == 0 || len(raw) > codexMetadataRepairBodyMaxBytes {
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

// 与官方 to_ascii_json_string 对齐，JSON 内的中文/非 BMP 字符用 Unicode 转义，
// 避免不同 HTTP/WS header 实现对非 ASCII 的接受范围不同。
func codexMetadataASCIIJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result strings.Builder
	result.Grow(len(raw))
	for _, r := range string(raw) {
		if r < 128 {
			result.WriteByte(byte(r))
		} else if r <= 0xffff {
			fmt.Fprintf(&result, "\\u%04x", r)
		} else {
			hi, lo := utf16.EncodeRune(r)
			fmt.Fprintf(&result, "\\u%04x\\u%04x", hi, lo)
		}
	}
	return []byte(result.String()), nil
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
	sourceKey := codexMetadataRepairSourceKey(c, account, body, sessionHint)
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

// 快照只按确实参与 metadata 构造的来源复用，不保存输入或凭据明文。
// 模型/工具等业务兼容重试不影响回合；身份、租户或请求类型变化则必须重建。
func codexMetadataRepairSourceKey(c *gin.Context, account *Account, body []byte, sessionHint string) [sha256.Size]byte {
	source := codexAccountIdentitySource(c, account)
	parts := []string{
		gjson.GetBytes(body, "client_metadata").Raw,
		sessionHint,
		gjson.GetBytes(body, "prompt_cache_key").String(),
		codexMetadataRepairRequestKind(body),
		fmt.Sprint(getAPIKeyIDFromContext(c)),
		codexAccountIdentityNamespace(source),
		source.GetOpenAIDeviceID(),
		string(source.GetCodexFingerprintMode()),
	}
	seed, _ := codexFingerprintSeed(source.Extra)
	parts = append(parts, seed)
	if c.Request != nil {
		parts = append(parts, c.Request.Header.Get(openAIWSTurnMetadataHeader))
		for _, identity := range codexMetadataRepairIdentity {
			for _, alias := range identity.aliases {
				parts = append(parts, c.Request.Header.Get(alias))
			}
		}
		for _, field := range codexMetadataLineageFields {
			for _, alias := range field.aliases {
				parts = append(parts, c.Request.Header.Get(alias))
			}
		}
	}
	raw, _ := json.Marshal(parts)
	return sha256.Sum256(raw)
}

// 仅从请求本身推导类型；客户端已经明确提供的类型由调用方保留。
func codexMetadataRepairRequestKind(body []byte) string {
	if gjson.GetBytes(body, "generate").Type == gjson.False {
		return "prewarm"
	}
	if HasCompactionTriggerInInput(body) {
		return "compaction"
	}
	return "turn"
}

// buildCodexMetadataRepair 的当前 body 优先于头；后续 WS 回合不复用握手的 turn
// 或环境扩展值，仅把稳定会话头作为缺失身份来源。不伪造安装设备、sandbox 或客户端能力。
func buildCodexMetadataRepair(c *gin.Context, account *Account, body []byte, sessionHint string, firstTurn bool) *codexMetadataRepairSnapshot {
	if c == nil || !account.IsCodexMetadataRepairEnabled() || isOpenAIResponsesCompactPath(c) {
		return nil
	}
	// 缺少可靠账号 namespace 时保留旧路径，不能用未隔离的客户端身份覆盖旧会话头。
	if codexAccountIdentityNamespace(codexAccountIdentitySource(c, account)) == "" {
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
	headerTurnRaw := headers.Get(openAIWSTurnMetadataHeader)
	if len(headerTurnRaw) > codexMetadataRepairMaxBytes {
		return nil
	}
	headerTurn := codexMetadataObject(headerTurnRaw)
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
	lineage, valid := collectCodexMetadataLineage(cm, bodyTurn, headerTurn, headers, firstTurn)
	if !valid {
		return nil
	}
	memory := codexMetadataString(turn["request_kind"]) == "memory"
	// Memory 不从握手或 flat 生成嵌套回合身份，显式提供的 turn 仍保留。
	explicitMemoryTurn := codexMetadataString(bodyTurn["turn_id"])
	if bodyTurnRaw == "" && firstTurn {
		explicitMemoryTurn = codexMetadataString(headerTurn["turn_id"])
	}
	rawIdentity := make(map[string]string)
	for _, identity := range codexMetadataRepairIdentity {
		value := codexMetadataString(bodyTurn[identity.field])
		for _, alias := range identity.aliases {
			if value == "" {
				value = codexMetadataString(cm[alias])
			}
		}
		if value == "" && (firstTurn || identity.field != "turn_id") && !(memory && identity.field == "turn_id") {
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
		if value == "" && identity.field == "turn_id" && !memory {
			value = uuid.NewString() // 仅生成本请求的相关标识；不制造客户端设备/权限事实。
		}
		if value != "" {
			if !httpguts.ValidHeaderFieldValue(value) {
				return nil
			}
			turn[identity.field] = value
			cm[identity.aliases[0]] = value
			rawIdentity[identity.field] = value
		}
	}
	if _, exists := turn["turn_started_at_unix_ms"]; !exists && !memory {
		turn["turn_started_at_unix_ms"] = time.Now().UnixMilli()
	}
	if codexMetadataString(turn["request_kind"]) == "" {
		turn["request_kind"] = codexMetadataRepairRequestKind(body)
	}
	// 只让原隔离器处理 identity 字段，不经其 float64 JSON 解码重写完整库存，
	// 避免大整数精度丢失；当前请求的其余元数据保留 json.Number。
	delete(cm, openAIWSTurnMetadataHeader)
	container := map[string]any{"client_metadata": cm}
	source := codexAccountIdentitySource(c, account)
	rawSessionID := codexMetadataString(turn["session_id"])
	applyCodexAccountIdentityClientMetadataMap(container, source, getAPIKeyIDFromContext(c))
	applyCodexAccountIdentityFields(turn, source, getAPIKeyIDFromContext(c))
	// 与已有收敛模式共存，但快照每回合只派生一次，最终头与体直接复用它。
	clientHeaders := headers.Clone()
	if rawSessionID != "" {
		clientHeaders.Set("session-id", rawSessionID)
	}
	fingerprint := resolveCodexFingerprintIDsFromRequest(source, clientHeaders)
	applyCodexFingerprintClientMetadata(container, fingerprint)
	cm = container["client_metadata"].(map[string]any)
	if fingerprint != nil && fingerprint.mode != codexFingerprintDevice && !memory {
		turn["turn_started_at_unix_ms"] = fingerprint.turnStartedAtUnixMs
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
	projectCodexMetadataLineage(cm, turn, outHeaders, lineage, source, getAPIKeyIDFromContext(c), fingerprint)
	var bindings []codexMetadataLineageBinding
	if fingerprint != nil && fingerprint.mode != codexFingerprintDevice && !memory {
		for _, field := range []struct{ name, kind string }{{"thread_id", "thread"}, {"turn_id", "turn"}} {
			if raw, final := rawIdentity[field.name], codexMetadataString(turn[field.name]); raw != "" && final != "" {
				bindings = append(bindings, codexMetadataLineageBinding{key: codexMetadataLineageKey(source, getAPIKeyIDFromContext(c), field.kind, raw), value: final})
			}
		}
	}
	if memory {
		// 官方 Memory 完整对象无这些线程身份，flat 兼容值和既有隔离仍保留。
		for _, field := range []string{"installation_id", "session_id", "thread_id", "agent_name", "window_id", "window_number", "context_window_id"} {
			delete(turn, field)
		}
		if explicitMemoryTurn == "" {
			delete(turn, "turn_id")
			// flat 的显式 turn 可保留隔离值，但不得向完整对象补造或输出随机值。
			if raw := rawIdentity["turn_id"]; raw != "" {
				cm["turn_id"] = scopeCodexAccountIdentityValue(source, getAPIKeyIDFromContext(c), "turn", raw)
				outHeaders.Set("turn_id", cm["turn_id"].(string))
				outHeaders.Set("turn-id", cm["turn_id"].(string))
			} else {
				delete(cm, "turn_id")
				delete(cm, "turn-id")
				outHeaders.Set("turn_id", "")
				outHeaders.Set("turn-id", "")
			}
		} else {
			turn["turn_id"] = scopeCodexAccountIdentityValue(source, getAPIKeyIDFromContext(c), "turn", explicitMemoryTurn)
			cm["turn_id"] = turn["turn_id"]
			outHeaders.Set("turn_id", turn["turn_id"].(string))
			outHeaders.Set("turn-id", turn["turn_id"].(string))
		}
		if _, exists := cm["turn-id"]; exists {
			cm["turn-id"] = cm["turn_id"]
		}
		// 当前 Memory blob 未提供时间时，不从旧握手继承另一个回合的时间。
		if bodyTurnRaw != "" {
			if _, exists := bodyTurn["turn_started_at_unix_ms"]; !exists {
				delete(turn, "turn_started_at_unix_ms")
			}
		}
	}
	turnJSON, err := codexMetadataASCIIJSON(turn)
	if err != nil || len(turnJSON) > codexMetadataRepairBodyMaxBytes {
		return nil
	}
	cm[openAIWSTurnMetadataHeader] = string(turnJSON)
	encoded, err := json.Marshal(cm)
	if err != nil || len(encoded) > codexMetadataRepairBodyMaxBytes {
		return nil
	}
	// 完整工具库存保持在 body，兼容 header 按官方投影省略该字段。
	delete(turn, "tool_namespaces_info")
	headerJSON, err := codexMetadataASCIIJSON(turn)
	if err != nil {
		return nil
	}
	if len(headerJSON) <= codexMetadataRepairMaxBytes {
		outHeaders.Set(openAIWSTurnMetadataHeader, string(headerJSON))
	} else {
		// 不截断合法完整 body；空值表示最终移除超限的可选兼容头。
		outHeaders.Set(openAIWSTurnMetadataHeader, "")
	}
	return &codexMetadataRepairSnapshot{accountID: account.ID, metadata: cm, headers: outHeaders, lineage: bindings}
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
	out, err := sjson.SetBytes(body, "client_metadata", snapshot.metadata)
	if err == nil {
		snapshot.rememberLineage()
	}
	return out, err
}

func (snapshot *codexMetadataRepairSnapshot) applyMap(body map[string]any) {
	if snapshot == nil || body == nil {
		return
	}
	// 深复制防止后续兼容转换原地写入而污染重试快照。
	raw, _ := json.Marshal(snapshot.metadata)
	body["client_metadata"] = codexMetadataObject(string(raw))
	snapshot.rememberLineage()
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
		if values[0] != "" {
			headers.Set(key, values[0])
		}
	}
}

// 网关主动发送的预热不是用户 turn；只改派生副本，不污染后续真实请求快照。
func applyCodexMetadataPrewarmKind(account *Account, payload map[string]any) {
	if !account.IsCodexMetadataRepairEnabled() || payload == nil {
		return
	}
	cm, ok := payload["client_metadata"].(map[string]any)
	if !ok {
		return
	}
	turn := codexMetadataObject(codexMetadataString(cm[openAIWSTurnMetadataHeader]))
	if turn == nil {
		return
	}
	turn["request_kind"] = "prewarm"
	raw, err := codexMetadataASCIIJSON(turn)
	if err != nil {
		return
	}
	copyMetadata := make(map[string]any, len(cm))
	for key, value := range cm {
		copyMetadata[key] = value
	}
	copyMetadata[openAIWSTurnMetadataHeader] = string(raw)
	payload["client_metadata"] = copyMetadata
}
