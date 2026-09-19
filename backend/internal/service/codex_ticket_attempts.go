package service

import "context"

// 计数与配置/凭据/模型同键，自动短轮次跨实例续计；成功票据单独保留最终轮数。
type CodexTicketAttemptCache interface {
	NextTicketAttempt(context.Context, string, int) (int, error)
	ResetTicketAttempts(context.Context, string) error
}

func (s *CodexTicketService) nextTicketAttempt(ctx context.Context, key string, max, fallback int) (int, error) {
	if c, ok := s.cache.(CodexTicketAttemptCache); ok {
		return c.NextTicketAttempt(ctx, key, max)
	}
	return fallback, nil
}
func (s *CodexTicketService) resetTicketAttempts(ctx context.Context, key string) {
	if c, ok := s.cache.(CodexTicketAttemptCache); ok {
		_ = c.ResetTicketAttempts(ctx, key)
	}
}
