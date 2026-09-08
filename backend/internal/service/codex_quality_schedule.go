package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/logger"
)

type CodexQualitySchedule struct {
	ID                int64               `json:"id"`
	Name              string              `json:"name"`
	IntervalMinutes   int                 `json:"interval_minutes"`
	KeepRuns          int                 `json:"keep_runs"`
	Enabled           bool                `json:"enabled"`
	Config            CodexQualityRequest `json:"config"`
	NextRunAt         time.Time           `json:"next_run_at"`
	ActiveRunID       *int64              `json:"active_run_id"`
	ManualRequestedAt *time.Time          `json:"manual_requested_at"`
}
type CodexQualityRun struct {
	ID            int64               `json:"id"`
	ScheduleID    int64               `json:"schedule_id"`
	ScheduleName  string              `json:"schedule_name"`
	Config        CodexQualityRequest `json:"config"`
	Status        string              `json:"status"`
	TriggerSource string              `json:"trigger_source"`
	StartedAt     time.Time           `json:"started_at"`
	FinishedAt    *time.Time          `json:"finished_at"`
	Counts        map[string]int      `json:"counts"`
}

func (p *CodexQualitySchedule) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len(p.Name) > 200 || strings.ContainsRune(p.Name, '\x00') {
		return errors.New("计划名称须为 1–200 字节")
	}
	if p.IntervalMinutes < 1 || p.IntervalMinutes > 43200 {
		return errors.New("间隔须为 1–43200 分钟")
	}
	if p.KeepRuns == 0 {
		p.KeepRuns = 30
	}
	if p.KeepRuns < 1 || p.KeepRuns > 100 {
		return errors.New("历史保留须为 1–100 轮")
	}
	return p.Config.Normalize()
}

type CodexQualityScheduleRepository interface {
	SaveQualitySchedule(context.Context, *CodexQualitySchedule) error
	ListQualitySchedules(context.Context) ([]*CodexQualitySchedule, error)
	SetQualityScheduleEnabled(context.Context, int64, bool) error
	TriggerQualitySchedule(context.Context, int64) error
	ClaimQualitySchedule(context.Context) (*CodexQualityRun, error)
	RenewQualitySchedule(context.Context, *CodexQualityRun) (bool, error)
	SaveQualityRunResult(context.Context, int64, *CodexQualityResult) error
	FinishQualityRun(context.Context, *CodexQualityRun, string) error
	ListQualityRuns(context.Context, int64) ([]*CodexQualityRun, error)
	GetQualityRun(context.Context, int64) (*CodexQualityRun, error)
	ListQualityRunResults(context.Context, int64, string, int, int) ([]*CodexQualityResult, int, error)
}

type qualityScheduledRunKey struct{}

var ErrQualityScheduleBusy = errors.New("计划不存在或正在执行")

func QualityScheduledRunID(ctx context.Context) int64 {
	id, _ := ctx.Value(qualityScheduledRunKey{}).(int64)
	return id
}
func (s *AccountTestService) QualityScheduleRepository() (CodexQualityScheduleRepository, bool) {
	if s == nil {
		return nil, false
	}
	r, ok := s.accountRepo.(CodexQualityScheduleRepository)
	return r, ok
}

// ValidateQualityScheduleAccounts 一次读取固定集合，避免保存大计划产生逐账号查询。
func (s *AccountTestService) ValidateQualityScheduleAccounts(ctx context.Context, ids []int64) error {
	if s == nil || s.accountRepo == nil {
		return errors.New("账号服务不可用")
	}
	accounts, err := s.accountRepo.GetByIDs(ctx, ids)
	if err != nil {
		return errors.New("读取账号失败")
	}
	if len(accounts) != len(ids) {
		return errors.New("所选账号不存在或已删除")
	}
	for _, account := range accounts {
		if !IsOpenAIQualityTestable(account) {
			return errors.New("计划只能包含独立 OpenAI OAuth 或 API Key 上游")
		}
	}
	return nil
}

// RunDueQualitySchedule 由现有 runner 的独立轮询调用，不依赖浏览器打开。
// @project-doc docs/operations/account_maintenance.md#codex_quality_schedules
func (s *AccountTestService) RunDueQualitySchedule(ctx context.Context) {
	repo, ok := s.QualityScheduleRepository()
	if !ok || !s.BeginCodexQualityBatch() {
		return
	}
	defer s.EndCodexQualityBatch()
	claimCtx, claimStop := context.WithTimeout(ctx, 5*time.Second)
	run, err := repo.ClaimQualitySchedule(claimCtx)
	claimStop()
	if err != nil {
		logger.LegacyPrintf("service.quality_schedule", "claim failed: %v", err)
		return
	}
	if run == nil {
		return
	}
	// 数据库旧配置也须重新校验，避免损坏的并发值让 worker 永久等待。
	if err = run.Config.Normalize(); err != nil {
		_ = repo.FinishQualityRun(ctx, run, "failed")
		return
	}
	// 整轮期限随账号数、并发和自定义单号超时扩展，不再被固定 20 小时截断。
	waves := (len(run.Config.AccountIDs) + run.Config.Concurrency - 1) / run.Config.Concurrency
	budget := time.Duration(waves*(run.Config.TimeoutSeconds+30)+60) * time.Second
	runCtx, cancel := context.WithTimeout(context.WithValue(ctx, qualityScheduledRunKey{}, run.ID), budget)
	defer cancel()
	var heartbeat sync.WaitGroup
	heartbeat.Add(1)
	go func() {
		defer heartbeat.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				leaseCtx, stop := context.WithTimeout(runCtx, 5*time.Second)
				valid, err := repo.RenewQualitySchedule(leaseCtx, run)
				stop()
				if err != nil || !valid {
					cancel()
					return
				}
			}
		}
	}()
	jobs := make(chan int64)
	var workers sync.WaitGroup
	var saveMu sync.Mutex
	saveFailed := false
	for i := 0; i < run.Config.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for id := range jobs {
				if runCtx.Err() != nil {
					return
				}
				// 领取每个账号前复核计划，暂停后不等待下一次心跳才停止派发。
				guardCtx, guardStop := context.WithTimeout(runCtx, 5*time.Second)
				valid, guardErr := repo.RenewQualitySchedule(guardCtx, run)
				guardStop()
				if guardErr != nil || !valid {
					cancel()
					return
				}
				result := s.RunCodexQualityTest(runCtx, id, &run.Config)
				saveCtx, stop := context.WithTimeout(context.WithoutCancel(runCtx), 10*time.Second)
				err := repo.SaveQualityRunResult(saveCtx, run.ID, result)
				stop()
				if err != nil {
					saveMu.Lock()
					saveFailed = true
					saveMu.Unlock()
					cancel()
					return
				}
			}
		}()
	}
loop:
	for _, id := range run.Config.AccountIDs {
		select {
		case jobs <- id:
		case <-runCtx.Done():
			break loop
		}
	}
	close(jobs)
	workers.Wait()
	status := "completed"
	if runCtx.Err() != nil {
		status = "interrupted"
	}
	if saveFailed {
		status = "failed"
	}
	cancel()
	heartbeat.Wait()
	finishCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer stop()
	if err = repo.FinishQualityRun(finishCtx, run, status); err != nil {
		logger.LegacyPrintf("service.quality_schedule", "finish run=%d failed: %v", run.ID, err)
	}
}
