package provider

import (
	"context"
	"testing"
	"time"
)

type providerExpiryRepo struct {
	ExpiryRepository
	started chan struct{}
}

func (r *providerExpiryRepo) AutoPauseExpiredProviders(context.Context, time.Time) (int64, error) {
	close(r.started)
	return 0, nil
}

// 已停止的拥有者不能因重复 Start 再执行到期扫描。
func TestProviderExpiryCannotRestartAfterStop(t *testing.T) {
	repo := &providerExpiryRepo{started: make(chan struct{})}
	svc := NewExpiryService(repo, ExpiryOptions{Interval: time.Hour})
	svc.Stop()
	svc.Start()
	defer svc.Stop()
	select {
	case <-repo.started:
		t.Fatal("stopped expiry worker executed another scan")
	case <-time.After(30 * time.Millisecond):
	}
}
