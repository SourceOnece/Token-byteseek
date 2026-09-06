package service

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http/httpguts"
)

// 关联只用于出站诊断元数据，绝不参与认证、账号选择或实际 response 续接。
var codexMetadataLineageFields = []struct {
	field   string
	kind    string
	aliases []string
}{
	{"parent_thread_id", "thread", []string{codexParentThreadIDHeader, "parent_thread_id"}},
	{"forked_from_thread_id", "thread", []string{"forked_from_thread_id"}},
	{"parent_turn_id", "turn", []string{"parent_turn_id"}},
	{"root_turn_id", "turn", []string{"root_turn_id"}},
	{"subagent_kind", "", []string{openAISubagentHeader, "subagent_kind"}},
}

type codexMetadataLineageBinding struct {
	key   [sha256.Size]byte
	value string
}

type codexMetadataLineageEntry struct {
	binding   codexMetadataLineageBinding
	expiresAt time.Time
	ambiguous bool
}

// 有界 LRU 仅保存摘要键和出站 ID；进程重启/过期时不猜测随机编号。
type codexMetadataLineageCache struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	entries  map[[sha256.Size]byte]*list.Element
	order    *list.List
}

func newCodexMetadataLineageCache(capacity int, ttl time.Duration) *codexMetadataLineageCache {
	return &codexMetadataLineageCache{capacity: capacity, ttl: ttl, entries: make(map[[sha256.Size]byte]*list.Element), order: list.New()}
}

var codexMetadataLineageMappings = newCodexMetadataLineageCache(32768, time.Hour)

func (cache *codexMetadataLineageCache) remember(binding codexMetadataLineageBinding, now time.Time) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if elem := cache.entries[binding.key]; elem != nil {
		entry := elem.Value.(*codexMetadataLineageEntry)
		if !now.Before(entry.expiresAt) {
			cache.order.Remove(elem)
			delete(cache.entries, binding.key)
		} else {
			// 同一来源在有效期内映射成不同 ID 时标记歧义，不能采用最后写入者。
			entry.ambiguous = entry.ambiguous || entry.binding.value != binding.value
			cache.order.MoveToFront(elem)
			return
		}
	}
	if cache.capacity <= 0 {
		return
	}
	for len(cache.entries) >= cache.capacity {
		oldest := cache.order.Back()
		delete(cache.entries, oldest.Value.(*codexMetadataLineageEntry).binding.key)
		cache.order.Remove(oldest)
	}
	elem := cache.order.PushFront(&codexMetadataLineageEntry{binding: binding, expiresAt: now.Add(cache.ttl)})
	cache.entries[binding.key] = elem
}

func (cache *codexMetadataLineageCache) lookup(key [sha256.Size]byte, now time.Time) (string, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	elem := cache.entries[key]
	if elem == nil {
		return "", false
	}
	entry := elem.Value.(*codexMetadataLineageEntry)
	if !now.Before(entry.expiresAt) {
		delete(cache.entries, key)
		cache.order.Remove(elem)
		return "", false
	}
	if entry.ambiguous {
		return "", false
	}
	cache.order.MoveToFront(elem)
	return entry.binding.value, true
}

func codexMetadataLineageKey(account *Account, apiKeyID int64, kind, raw string) [sha256.Size]byte {
	seed, _ := codexFingerprintSeed(account.Extra)
	encoded, _ := json.Marshal([]string{codexAccountIdentityNamespace(account), fmt.Sprint(apiKeyID), string(account.GetCodexFingerprintMode()), seed, account.GetOpenAIDeviceID(), kind, raw})
	return sha256.Sum256(encoded)
}

// 当前 body 内嵌值优先于 flat 与首轮握手；后续轮不继承旧的父子回合关系。
func collectCodexMetadataLineage(cm, bodyTurn, headerTurn map[string]any, headers http.Header, firstTurn bool) (map[string]string, bool) {
	values := make(map[string]string)
	for _, field := range codexMetadataLineageFields {
		candidates := []any{bodyTurn[field.field]}
		for _, alias := range field.aliases {
			candidates = append(candidates, cm[alias])
		}
		if firstTurn {
			candidates = append(candidates, headerTurn[field.field])
			for _, alias := range field.aliases {
				candidates = append(candidates, headers.Get(alias))
			}
		}
		for _, candidate := range candidates {
			if candidate == nil {
				continue
			}
			if _, ok := candidate.(string); !ok {
				return nil, false
			}
			value := codexMetadataString(candidate)
			if value == "" {
				continue
			}
			if !httpguts.ValidHeaderFieldValue(value) || len(value) > codexMetadataRepairMaxBytes {
				return nil, false
			}
			values[field.field] = value
			break
		}
	}
	return values, true
}

// 引用必须与对应主 thread/turn 的出站值同源；无法推导的收敛映射不冒用原始 ID。
func projectCodexMetadataLineage(cm, turn map[string]any, headers http.Header, raw map[string]string, account *Account, apiKeyID int64, fingerprint *codexFingerprintIDs) {
	headers.Set(codexParentThreadIDHeader, "")
	headers.Set(openAISubagentHeader, "")
	for _, field := range codexMetadataLineageFields {
		existingAliases := make(map[string]bool, len(field.aliases))
		delete(turn, field.field)
		for _, alias := range field.aliases {
			_, existingAliases[alias] = cm[alias]
			delete(cm, alias)
		}
		value := raw[field.field]
		if value == "" {
			continue
		}
		if field.kind != "" {
			if fingerprint != nil && fingerprint.mode != codexFingerprintDevice {
				mapped, ok := codexMetadataLineageMappings.lookup(codexMetadataLineageKey(account, apiKeyID, field.kind, value), time.Now())
				if !ok {
					continue
				}
				value = mapped
			} else {
				value = scopeCodexAccountIdentityValue(account, apiKeyID, field.kind, value)
			}
		}
		turn[field.field] = value
		// fork 来源仅在官方完整对象中定义；不凭空增加非官方 flat 属性。
		if field.field != "forked_from_thread_id" {
			cm[field.aliases[0]] = value
		}
		for _, alias := range field.aliases {
			if existingAliases[alias] {
				cm[alias] = value
			}
		}
		if field.field == "parent_thread_id" || field.field == "subagent_kind" {
			headers.Set(field.aliases[0], value)
		}
	}
}

// 只在快照实际投影到出站 body 后登记，不登记解析失败或尚未应用的候选。
func (snapshot *codexMetadataRepairSnapshot) rememberLineage() {
	if snapshot == nil {
		return
	}
	for _, binding := range snapshot.lineage {
		codexMetadataLineageMappings.remember(binding, time.Now())
	}
}
