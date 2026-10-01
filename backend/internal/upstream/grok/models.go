package grok

import (
	"strings"
	"sync/atomic"
)

var (
	runtimeMappingOpts    atomic.Value // ModelMappingOptions
	runtimeMappingVersion atomic.Uint64
)

func init() {
	runtimeMappingOpts.Store(ModelMappingOptions{})
	runtimeMappingVersion.Store(1)
}

// SetRuntimeModelMappingOptions 更新 Grok 缺省文本型号的进程级快照。
func SetRuntimeModelMappingOptions(opts ModelMappingOptions) {
	runtimeMappingOpts.Store(opts)
	runtimeMappingVersion.Add(1)
}

// RuntimeModelMappingOptions 返回最近一次已发布的 Grok 运行时映射配置。
func RuntimeModelMappingOptions() ModelMappingOptions {
	if value := runtimeMappingOpts.Load(); value != nil {
		if opts, ok := value.(ModelMappingOptions); ok {
			return opts
		}
	}
	return ModelMappingOptions{}
}

// DefaultResponsesModel 是 Grok Responses 请求未指定模型时使用的默认模型。
const DefaultResponsesModel = "grok-4.5"

// Model 描述 xAI OpenAI 兼容 /models 响应里的模型。
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Type        string `json:"type,omitempty"`
	Created     int64  `json:"created,omitempty"`
	OwnedBy     string `json:"owned_by"`
	DisplayName string `json:"display_name,omitempty"`
}

// DefaultTextModel 是空模型字段和 Grok 文本别名的内置回退目标，
// 运维可通过设置键 grok_default_text_model 覆盖运行时默认值。
const DefaultTextModel = "grok-4.6"

// 以下为官方 Imagine 模型 ID。
const (
	DefaultImagineImageQualityModel = "grok-imagine-image-quality"
	DefaultImagineImageFastModel    = "grok-imagine-image"
	DefaultImagineImage20Model      = "grok-imagine-image-2.0"
	DefaultImagineVideoModel        = "grok-imagine-video"
	DefaultImagineVideo15Model      = "grok-imagine-video-1.5"
)

// ModelMappingOptions 保存允许省略模型的请求所用的默认文本型号。
type ModelMappingOptions struct {
	// DefaultText 仅用于空模型，空值使用 DefaultTextModel。
	DefaultText string
}

var defaultModels = []Model{
	// 文本模型。
	{ID: "grok-4.7", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.7"},
	{ID: "grok-4.6", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.6"},
	{ID: "grok-4.5", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.5"},
	{ID: "grok-4.3", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.3"},
	{ID: "grok-build-0.1", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Build 0.1"},
	{ID: "grok-composer-2.5-fast", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Composer 2.5 Fast"},
	{ID: "grok-4.20-0309-reasoning", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.20 Reasoning"},
	{ID: "grok-4.20-0309-non-reasoning", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.20 Non Reasoning"},
	{ID: "grok-4.20-multi-agent-0309", Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok 4.20 Multi Agent"},
	// Imagine 媒体模型。
	{ID: DefaultImagineImageQualityModel, Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Imagine Image Quality"},
	{ID: DefaultImagineImageFastModel, Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Imagine Image"},
	{ID: DefaultImagineImage20Model, Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Imagine Image 2.0"},
	{ID: DefaultImagineVideoModel, Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Imagine Video"},
	{ID: DefaultImagineVideo15Model, Object: "model", Type: "model", OwnedBy: "xai", DisplayName: "Grok Imagine Video 1.5"},
}

// grokTextResponsesModelAliases 登记 Responses 原生文本型号的精确身份。
var grokTextResponsesModelAliases = map[string]string{
	"grok-4.6": "grok-4.6",

	"grok-4.5": "grok-4.5",

	"grok-4.3": "grok-4.3",

	"grok-3-mini": "grok-3-mini",

	"grok-3-mini-fast": "grok-3-mini-fast",

	"grok-build-0.1": "grok-build-0.1",

	"grok-composer-2.5-fast": "grok-composer-2.5-fast",

	"grok-4.20-0309-reasoning": "grok-4.20-0309-reasoning",

	"grok-4.20-0309-non-reasoning": "grok-4.20-0309-non-reasoning",

	"grok-4.20-multi-agent-0309": "grok-4.20-multi-agent-0309",
}

func DefaultModels() []Model {
	out := make([]Model, len(defaultModels))
	copy(out, defaultModels)
	return out
}

func DefaultModelIDs() []string {
	models := DefaultModels()
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

// DefaultModelMapping 根据当前运行时配置构造 Grok 的内置模型映射。
func DefaultModelMapping() map[string]string {
	return ModelMappingWithOptions(RuntimeModelMappingOptions())
}

// ModelMappingWithOptions 仅提供原生完整型号的自映射。
func ModelMappingWithOptions(opts ModelMappingOptions) map[string]string {
	mapping := make(map[string]string, len(defaultModels))
	for _, model := range defaultModels {
		mapping[model.ID] = model.ID
	}
	return mapping
}

// NormalizeModelID 保留完整模型 ID，允许省略模型的入口使用协议默认值。
func NormalizeModelID(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return DefaultResponsesModel
	}
	return model
}

// grokReasoningModelID 只读识别推理能力，结果不得用于转发、查价或额度键。
func grokReasoningModelID(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range []string{"xai/", "x-ai/", "grok/"} {
		if native, found := strings.CutPrefix(model, prefix); found {
			return strings.TrimSpace(native)
		}
	}
	return model
}

// IsGrokModelID 判断模型是否为 Grok/xAI 原生 ID 或别名；Claude/OpenAI 模型返回 false。
func IsGrokModelID(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "grok") {
		return true
	}
	if strings.HasPrefix(normalized, "imagine") {
		return true
	}
	return false
}

// IsGrokTextResponsesModelID 判断模型是否为 Responses API 已知的 Grok 文本模型；
// Imagine 媒体模型和未知自定义 ID 返回 false。
func IsGrokTextResponsesModelID(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	_, ok := grokTextResponsesModelAliases[normalized]
	return ok
}

// ResolveGrokTextResponsesModelID 仅为空模型选用默认值，非空 ID 原样保留。
func ResolveGrokTextResponsesModelID(model string, defaultText ...string) string {
	return ResolveDefaultTextModel(model, defaultText...)
}

// ResolveDefaultTextModel 在模型为空时返回 defaultText，未提供则返回 DefaultTextModel。
func ResolveDefaultTextModel(model string, defaultText ...string) string {
	if trimmed := strings.TrimSpace(model); trimmed != "" {
		return trimmed
	}
	if len(defaultText) > 0 && strings.TrimSpace(defaultText[0]) != "" {
		return strings.TrimSpace(defaultText[0])
	}
	return DefaultTextModel
}

// CanonicalImagineVideoModel 使用完整视频型号查询价格，缺省请求使用默认型号。
func CanonicalImagineVideoModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return DefaultImagineVideoModel
	}
	return model
}
