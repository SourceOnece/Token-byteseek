//go:build integration

package migrations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
)

// pricingMigrationPreview 返回可复核的变价与冲突；不包含提供商凭据或 Key 字符串。
type pricingMigrationPreview struct {
	Scopes        []pricingMigrationScope `json:"scopes"`
	UnboundKeyIDs []int64                 `json:"unbound_key_ids"`
	Blocked       bool                    `json:"blocked"`
}

type pricingMigrationScope struct {
	Kind      string                      `json:"kind"`
	ID        int64                       `json:"id"`
	Before    json.RawMessage             `json:"before"`
	After     []pricing.ModelPricingEntry `json:"after"`
	Conflicts []pricingMergeConflict      `json:"conflicts,omitempty"`
}

// previewPricingMigration 读取迁移 276 之前的 schema，历史 accounts 表名在此保留。
// 预检使用只读一致性事务，失败不修改持久数据。
func previewPricingMigration(ctx context.Context, db *sql.DB) (pricingMigrationPreview, error) {
	report := pricingMigrationPreview{Scopes: []pricingMigrationScope{}, UnboundKeyIDs: []int64{}}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
SELECT 'config', pricing_config_id, jsonb_agg(to_jsonb(p) || jsonb_build_object('intervals', COALESCE((
 SELECT jsonb_agg(to_jsonb(i) ORDER BY i.sort_order, i.id) FROM pricing_config_pricing_intervals i WHERE i.pricing_id = p.id
), '[]')) ORDER BY p.id) FROM pricing_config_model_pricing p GROUP BY pricing_config_id
UNION ALL
SELECT 'account_rule', rule_id, jsonb_agg(to_jsonb(p) || jsonb_build_object('intervals', COALESCE((
 SELECT jsonb_agg(to_jsonb(i) ORDER BY i.sort_order, i.id) FROM pricing_config_account_stats_pricing_intervals i WHERE i.pricing_id = p.id
), '[]')) ORDER BY p.id) FROM pricing_config_account_stats_model_pricing p GROUP BY rule_id
UNION ALL
SELECT 'group', id, model_pricing FROM groups WHERE model_pricing IS NOT NULL
ORDER BY 1, 2`)
	if err != nil {
		return report, fmt.Errorf("read migration price cards: %w", err)
	}
	for rows.Next() {
		var scope pricingMigrationScope
		var raw []byte
		if err := rows.Scan(&scope.Kind, &scope.ID, &raw); err != nil {
			_ = rows.Close()
			return report, err
		}
		var original []pricing.ModelPricingEntry
		if err := json.Unmarshal(raw, &original); err != nil {
			_ = rows.Close()
			return report, fmt.Errorf("decode %s %d: %w", scope.Kind, scope.ID, err)
		}
		// 预检保留旧平台标签与行 ID，管理员可以定位待合并条目；运行时价卡不再持有平台。
		scope.Before = append(json.RawMessage(nil), raw...)
		scope.After, scope.Conflicts = mergePriceCards(original)
		report.Blocked = report.Blocked || len(scope.Conflicts) > 0
		report.Scopes = append(report.Scopes, scope)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return report, err
	}
	keys, err := tx.QueryContext(ctx, `SELECT id FROM api_keys WHERE group_id IS NULL AND NOT is_composite AND deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return report, err
	}
	for keys.Next() {
		var id int64
		if err := keys.Scan(&id); err != nil {
			_ = keys.Close()
			return report, err
		}
		report.UnboundKeyIDs = append(report.UnboundKeyIDs, id)
	}
	err = keys.Err()
	_ = keys.Close()
	if err != nil {
		return report, err
	}
	return report, tx.Commit()
}
