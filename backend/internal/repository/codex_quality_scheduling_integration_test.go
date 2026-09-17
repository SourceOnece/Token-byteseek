//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

// 上游只用合成响应；实际 service、账号租约、结果 SQL 和定时 runner 都运行。
type qualitySchedulingUpstream struct {
	status      int
	body        string
	beforeReply func()
}

func (u *qualitySchedulingUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if u.beforeReply != nil {
		u.beforeReply()
	}
	return &http.Response{StatusCode: u.status, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(u.body))}, nil
}
func (u *qualitySchedulingUpstream) DoWithTLS(r *http.Request, p string, id int64, c int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(r, p, id, c)
}

func TestQualitySchedulingFailedPreservesStateAndHistory(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newAccountRepositoryWithSQL(client, integrationDB, nil)
	for _, origin := range []string{"batch", "periodic", "manual-plan"} {
		for _, enabled := range []bool{true, false} {
			for _, resultStatus := range []string{"full", "degraded", "failed"} {
				t.Run(fmt.Sprintf("%s/%t/%s", origin, enabled, resultStatus), func(t *testing.T) {
					account := mustCreateAccount(t, client, &service.Account{Name: "quality-state", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Credentials: map[string]any{"api_key": "fake-key", "base_url": "https://upstream.example/v1"}})
					// fixture 默认开启调度，显式写出要验证的初始状态。
					_, err := client.Account.UpdateOneID(account.ID).SetSchedulable(enabled).Save(ctx)
					require.NoError(t, err)
					before, err := repo.GetByID(ctx, account.ID)
					require.NoError(t, err)
					countOutbox := func() int {
						var n int
						require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox WHERE account_id=$1", account.ID).Scan(&n))
						return n
					}
					countBefore := countOutbox()
					answer := "PASS"
					if resultStatus == "degraded" {
						answer = "NO MATCH"
					}
					event, _ := json.Marshal(map[string]any{"type": "response.output_text.delta", "delta": answer})
					up := &qualitySchedulingUpstream{status: 200, body: "data: " + string(event) + "\n\ndata: {\"type\":\"response.completed\"}\n\n"}
					if resultStatus == "failed" {
						up.status = 429
						up.body = `{"error":{"message":"overloaded"}}`
					}
					svc := service.NewAccountTestService(repo, nil, nil, nil, nil, &service.OpenAIGatewayService{}, up, &config.Config{}, nil)
					options := service.CodexQualityRequest{AccountIDs: []int64{account.ID}, Model: "gpt-6-astra", Prompt: "test", Keyword: "PASS", ConfirmScheduling: true, APIProtocol: "responses", Concurrency: 1, TimeoutSeconds: 120}
					var result *service.CodexQualityResult
					var plan service.CodexQualitySchedule
					if origin == "batch" {
						result = svc.RunCodexQualityTest(ctx, account.ID, &options)
					} else {
						plan = service.CodexQualitySchedule{Name: "quality-plan", IntervalMinutes: 60, KeepRuns: 30, Config: options, Enabled: origin == "periodic"}
						require.NoError(t, repo.SaveQualitySchedule(ctx, &plan))
						t.Cleanup(func() { _, _ = repo.DeleteQualitySchedule(ctx, plan.ID) })
						if origin == "periodic" {
							_, err = integrationDB.ExecContext(ctx, "UPDATE codex_quality_schedules SET next_run_at=NOW() WHERE id=$1", plan.ID)
							require.NoError(t, err)
						} else {
							require.NoError(t, repo.TriggerQualitySchedule(ctx, plan.ID))
						}
						svc.RunDueQualitySchedule(ctx)
						runs, err := repo.ListQualityRuns(ctx, plan.ID)
						require.NoError(t, err)
						require.Len(t, runs, 1)
						require.Equal(t, "completed", runs[0].Status)
						history, total, err := repo.ListQualityRunResults(ctx, runs[0].ID, "", 1, 20)
						require.NoError(t, err)
						require.Equal(t, 1, total)
						require.Len(t, history, 1)
						result = history[0]
					}
					require.Equal(t, resultStatus, result.Status, result.Error)
					require.Equal(t, resultStatus != "failed", result.SchedulingApplied)
					expected := resultStatus == "full"
					if resultStatus == "failed" {
						expected = enabled
					}
					require.Equal(t, expected, result.Schedulable)
					after, err := repo.GetByID(ctx, account.ID)
					require.NoError(t, err)
					require.Equal(t, expected, after.Schedulable)
					require.Equal(t, before.Status, after.Status)
					if resultStatus == "failed" {
						require.True(t, before.UpdatedAt.Equal(after.UpdatedAt))
						require.Equal(t, countBefore, countOutbox())
					} else {
						require.Greater(t, countOutbox(), countBefore)
					}
					latest, err := repo.ListCodexQualityResults(ctx, []int64{account.ID}, true)
					require.NoError(t, err)
					require.Len(t, latest, 1)
					require.Equal(t, resultStatus, latest[0].Status)
					require.Equal(t, result.SchedulingApplied, latest[0].SchedulingApplied)
				})
			}
		}
	}
}

func TestQualitySchedulingFailedRejectsChangedAccountAndRevokedPlan(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newAccountRepositoryWithSQL(client, integrationDB, nil)
	for _, revoke := range []string{"account", "pause", "delete"} {
		t.Run(revoke, func(t *testing.T) {
			account := mustCreateAccount(t, client, &service.Account{Name: "guard", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Credentials: map[string]any{"api_key": "fake", "base_url": "https://upstream.example/v1"}})
			options := service.CodexQualityRequest{AccountIDs: []int64{account.ID}, Model: "test", Prompt: "test", Keyword: "PASS", ConfirmScheduling: true, Concurrency: 1, TimeoutSeconds: 120, APIProtocol: "responses"}
			plan := service.CodexQualitySchedule{Name: "guard", IntervalMinutes: 60, KeepRuns: 30, Config: options}
			if revoke != "account" {
				require.NoError(t, repo.SaveQualitySchedule(ctx, &plan))
				require.NoError(t, repo.TriggerQualitySchedule(ctx, plan.ID))
				t.Cleanup(func() { _, _ = repo.DeleteQualitySchedule(ctx, plan.ID) })
			}
			up := &qualitySchedulingUpstream{status: 401, body: `{"error":{"message":"failed"}}`, beforeReply: func() {
				if revoke == "account" {
					_, err := client.Account.UpdateOneID(account.ID).SetSchedulable(false).Save(ctx)
					require.NoError(t, err)
				}
				if revoke == "pause" {
					require.NoError(t, repo.SetQualityScheduleEnabled(ctx, plan.ID, false))
				}
				if revoke == "delete" {
					ok, err := repo.DeleteQualitySchedule(ctx, plan.ID)
					require.NoError(t, err)
					require.True(t, ok)
				}
			}}
			svc := service.NewAccountTestService(repo, nil, nil, nil, nil, &service.OpenAIGatewayService{}, up, &config.Config{}, nil)
			if revoke == "account" {
				r := svc.RunCodexQualityTest(ctx, account.ID, &options)
				require.Equal(t, "stale", r.Status)
				require.False(t, r.SchedulingApplied)
			} else {
				svc.RunDueQualitySchedule(ctx)
			}
			a, err := repo.GetByID(ctx, account.ID)
			require.NoError(t, err)
			require.Equal(t, revoke != "account", a.Schedulable)
			latest, err := repo.ListCodexQualityResults(ctx, []int64{account.ID}, true)
			require.NoError(t, err)
			if revoke != "account" {
				require.Empty(t, latest)
			}
		})
	}
}
