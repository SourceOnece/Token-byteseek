package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
)

// 阈值键不包含消费累计，普通更新不会写入提供商资金快照。
const SettingKeyProviderSchedulingThresholds = "provider_scheduling_thresholds"

// cachedProviderSchedulingThresholds 缓存各平台自动停调阈值。
type cachedProviderSchedulingThresholds struct {
	thresholds map[string]int
	expiresAt  int64 // Unix 纳秒时间戳
}

const (
	providerSchedulingThresholdsCacheTTL  = 60 * time.Second
	providerSchedulingThresholdsErrorTTL  = 5 * time.Second
	providerSchedulingThresholdsDBTimeout = 5 * time.Second
)

func DefaultProviderSchedulingThresholds() map[string]int {
	return map[string]int{
		PlatformOpenAI:    100,
		PlatformAnthropic: 100,
		PlatformGrok:      100,
	}
}

func ValidateAndNormalizeProviderSchedulingThresholds(input map[string]int) (map[string]int, error) {
	normalized := DefaultProviderSchedulingThresholds()
	for platform, value := range input {
		allowed := false
		for _, item := range AllowedSchedulingThresholdPlatforms {
			if item == platform {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, apperror.BadRequest("INVALID_PROVIDER_SCHEDULING_THRESHOLDS", fmt.Sprintf("unknown platform %q", platform))
		}
		if value < 1 || value > 100 {
			return nil, apperror.BadRequest("INVALID_PROVIDER_SCHEDULING_THRESHOLDS", "platform scheduling threshold must be between 1 and 100")
		}
		normalized[platform] = value
	}
	return normalized, nil
}

func ParseProviderSchedulingThresholdsSetting(raw string) (map[string]int, error) {
	thresholds := DefaultProviderSchedulingThresholds()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return thresholds, nil
	}
	parsed := map[string]int{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return thresholds, err
	}
	for _, platform := range AllowedSchedulingThresholdPlatforms {
		if value, ok := parsed[platform]; ok {
			thresholds[platform] = BoundedIntOrDefault(value, 1, 100, 100)
		}
	}
	return thresholds, nil
}

func BoundedIntOrDefault(value, minValue, maxValue, defaultValue int) int {
	if value < minValue || value > maxValue {
		return defaultValue
	}
	return value
}

func CloneProviderSchedulingThresholds(input map[string]int) map[string]int {
	if len(input) == 0 {
		return DefaultProviderSchedulingThresholds()
	}
	cloned := make(map[string]int, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

// GetProviderSchedulingThresholds 保留正常与故障 TTL、singleflight 及拷贝语义。
func (s *RuntimeSettings) GetProviderSchedulingThresholds(ctx context.Context) map[string]int {
	if s == nil || s.settingRepo == nil {
		return DefaultProviderSchedulingThresholds()
	}
	if cached, ok := s.providerSchedulingThresholdsCache.Load().(*cachedProviderSchedulingThresholds); ok {
		if cached != nil && len(cached.thresholds) > 0 && time.Now().UnixNano() < cached.expiresAt {
			return CloneProviderSchedulingThresholds(cached.thresholds)
		}
	}

	result, err, _ := s.providerSchedulingThresholdsSF.Do(SettingKeyProviderSchedulingThresholds, func() (any, error) {
		if cached, ok := s.providerSchedulingThresholdsCache.Load().(*cachedProviderSchedulingThresholds); ok {
			if cached != nil && len(cached.thresholds) > 0 && time.Now().UnixNano() < cached.expiresAt {
				return CloneProviderSchedulingThresholds(cached.thresholds), nil
			}
		}

		thresholds := DefaultProviderSchedulingThresholds()
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), providerSchedulingThresholdsDBTimeout)
		defer cancel()

		raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyProviderSchedulingThresholds)
		if err != nil {
			if errors.Is(err, s.notFound) {
				// 未配置阈值属于稳定默认状态，按正常周期缓存，避免热点路径持续查询数据库。
				s.providerSchedulingThresholdsCache.Store(&cachedProviderSchedulingThresholds{
					thresholds: CloneProviderSchedulingThresholds(thresholds),
					expiresAt:  time.Now().Add(providerSchedulingThresholdsCacheTTL).UnixNano(),
				})
				return CloneProviderSchedulingThresholds(thresholds), nil
			}
			slog.Warn("failed to get provider scheduling thresholds, falling back to defaults", "error", err)
			s.providerSchedulingThresholdsCache.Store(&cachedProviderSchedulingThresholds{
				thresholds: CloneProviderSchedulingThresholds(thresholds),
				expiresAt:  time.Now().Add(providerSchedulingThresholdsErrorTTL).UnixNano(),
			})
			return CloneProviderSchedulingThresholds(thresholds), nil
		}

		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			if parsed, err := ParseProviderSchedulingThresholdsSetting(trimmed); err != nil {
				slog.Warn("failed to parse provider scheduling thresholds, falling back to defaults", "error", err)
			} else {
				thresholds = parsed
			}
		}

		s.providerSchedulingThresholdsCache.Store(&cachedProviderSchedulingThresholds{
			thresholds: CloneProviderSchedulingThresholds(thresholds),
			expiresAt:  time.Now().Add(providerSchedulingThresholdsCacheTTL).UnixNano(),
		})
		return CloneProviderSchedulingThresholds(thresholds), nil
	})
	if err != nil {
		return DefaultProviderSchedulingThresholds()
	}
	if thresholds, ok := result.(map[string]int); ok {
		return CloneProviderSchedulingThresholds(thresholds)
	}
	return DefaultProviderSchedulingThresholds()
}

// ApplySchedulingThresholds 仅在设置提交后发布；省略时保持原清缓存行为。
func (s *RuntimeSettings) ApplySchedulingThresholds(value map[string]int) {
	s.providerSchedulingThresholdsSF.Forget(SettingKeyProviderSchedulingThresholds)
	if value != nil {
		normalizedThresholds, err := ValidateAndNormalizeProviderSchedulingThresholds(value)
		if err != nil {
			normalizedThresholds = DefaultProviderSchedulingThresholds()
		}
		s.providerSchedulingThresholdsCache.Store(&cachedProviderSchedulingThresholds{
			thresholds: CloneProviderSchedulingThresholds(normalizedThresholds),
			expiresAt:  time.Now().Add(providerSchedulingThresholdsCacheTTL).UnixNano(),
		})
	} else {
		// 请求体部分更新或省略该字段时清除缓存，使下次热点读取从数据库重新加载。
		s.providerSchedulingThresholdsCache.Store(&cachedProviderSchedulingThresholds{})
	}
}
