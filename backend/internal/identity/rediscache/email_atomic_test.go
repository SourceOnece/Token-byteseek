package rediscache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type resetLinkSender struct{ link string }

func (*resetLinkSender) SendVerifyCodeMessage(context.Context, string, string, string, string) error {
	return nil
}

func (s *resetLinkSender) SendPasswordResetMessage(_ context.Context, _, _, link, _ string) error {
	s.link = link
	return nil
}

// 并发猜测不会把尝试次数回写覆盖，重发才重置次数且不延长原挑战有效期。
func TestEmailChallengeAtomicAttempts(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewEmailCache(client)
	ctx := context.Background()
	data := &identity.VerificationCodeData{Code: "123456", ExpiresAt: time.Now().Add(time.Minute)}
	require.NoError(t, cache.SetVerificationCode(ctx, "test@example.invalid", data, time.Minute))
	service := identity.NewEmailChallenges(cache, &resetLinkSender{})
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = service.VerifyCode(ctx, "test@example.invalid", "wrong") }()
	}
	wg.Wait()
	require.ErrorIs(t, service.VerifyCode(ctx, "test@example.invalid", "123456"), identity.ErrVerifyCodeMaxAttempts)
	mini.FastForward(time.Minute)
	require.ErrorIs(t, service.VerifyCode(ctx, "test@example.invalid", "123456"), identity.ErrInvalidVerifyCode)
	require.NoError(t, cache.SetVerificationCode(ctx, "test@example.invalid", data, time.Minute))
	require.NoError(t, service.VerifyCode(ctx, "test@example.invalid", "123456"))
}

// 邮件拿到原文，缓存仅保存摘要；再次发送撤销旧链接，并发消费只成功一次。
func TestResetTokenHashAndSingleUse(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewEmailCache(client)
	sender := &resetLinkSender{}
	service := identity.NewEmailChallenges(cache, sender)
	ctx := context.Background()
	const email = "test@example.invalid"
	require.NoError(t, service.SendPasswordResetEmail(ctx, email, "Test", "https://example.invalid/reset"))
	parsed, err := url.Parse(sender.link)
	require.NoError(t, err)
	oldToken := parsed.Query().Get("token")
	stored, err := cache.GetPasswordResetToken(ctx, email)
	require.NoError(t, err)
	sum := sha256.Sum256([]byte(oldToken))
	require.Equal(t, hex.EncodeToString(sum[:]), stored.Token)
	require.NotEqual(t, oldToken, stored.Token)
	require.NoError(t, service.VerifyPasswordResetToken(ctx, email, oldToken))
	require.NoError(t, service.SendPasswordResetEmail(ctx, email, "Test", "https://example.invalid/reset"))
	parsed, err = url.Parse(sender.link)
	require.NoError(t, err)
	newToken := parsed.Query().Get("token")
	require.Error(t, service.VerifyPasswordResetToken(ctx, email, oldToken))
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if service.ConsumePasswordResetToken(ctx, email, newToken) == nil {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), winners.Load())
}
