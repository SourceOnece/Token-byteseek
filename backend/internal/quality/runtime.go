package quality

import (
	"context"
	"errors"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"io"
	"strings"
	"sync/atomic"
)

type Account = provider.Record
type TestEvent = provider.TestEvent
type accountTestContextKey string

const PlatformOpenAI = "openai"
const AccountTypeOAuth = "oauth"
const AccountTypeAPIKey = "apikey"

type AccountStore interface {
	GetByID(context.Context, int64) (*provider.Record, error)
	GetByIDs(context.Context, []int64) ([]*provider.Record, error)
}
type Execute func(context.Context, *provider.Record, *CodexQualityRequest) (string, error)

// 质量检测单独持有批次状态，运行时只接收共享账号存储和原生测试适配器。
type Service struct {
	accountRepo        AccountStore
	execute            Execute
	qualityBatchActive atomic.Bool
}

func New(store AccountStore, execute Execute) *Service {
	return &Service{accountRepo: store, execute: execute}
}

type qualitySink struct{ Text string }

func (s *Service) sendEvent(out *qualitySink, event TestEvent) {
	if event.Type == "content" {
		out.Text = event.Text
	}
}
func (s *Service) sendErrorAndEnd(_ *qualitySink, message string) error { return errors.New(message) }

// 严格解析完整的可见回答；思考内容和错误提示不计入关键词。
func Responses(body io.Reader) (string, error) {
	var core Service
	out := &qualitySink{}
	err := core.processCodexQualityStream(out, body)
	return out.Text, err
}
func Chat(body io.Reader) (string, error) {
	var core Service
	out := &qualitySink{}
	err := core.processCodexQualityChatStream(out, body)
	return out.Text, err
}

func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
