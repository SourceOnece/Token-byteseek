package postgres

import (
	"context"
	"errors"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	dbaccountgroup "github.com/TokenFlux/TokenRouter/ent/accountgroup"
	"github.com/TokenFlux/TokenRouter/internal/domain"
)

// SetSchedulableRecord 保留原Ent软删除和更新时间行为；提交后通知由外层原仓储执行。
func (s *Store) SetSchedulableRecord(ctx context.Context, id int64, enabled bool) error {
	_, err := s.client.Account.Update().Where(dbaccount.IDEQ(id)).SetSchedulable(enabled).Save(ctx)
	return err
}

func (s *Store) SetErrorRecord(ctx context.Context, id int64, message string) error {
	_, err := s.client.Account.Update().Where(dbaccount.IDEQ(id)).SetStatus(domain.StatusError).SetErrorMessage(message).SetSchedulable(false).Save(ctx)
	return err
}

func (s *Store) ClearErrorRecord(ctx context.Context, id int64) error {
	_, err := s.client.Account.Update().Where(dbaccount.IDEQ(id)).SetStatus(domain.StatusActive).SetErrorMessage("").Save(ctx)
	return err
}

// DeleteRecord 与原实现相同：关联/测试计划/软删除在同一事务，复用已有事务时不提前提交。
// 调度缓存与outbox仍由外层在此成功后处理，票据/质量历史不在这里擅自物理删除。
func (s *Store) DeleteRecord(ctx context.Context, id int64) error {
	tx, err := s.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return err
	}
	client := s.client
	if tx != nil {
		defer func() { _ = tx.Rollback() }()
		client = tx.Client()
	}
	if _, err := client.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := client.ExecContext(ctx, "DELETE FROM scheduled_test_plans WHERE account_id = $1", id); err != nil {
		return err
	}
	if _, err := client.Account.Delete().Where(dbaccount.IDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}
