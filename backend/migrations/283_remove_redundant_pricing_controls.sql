-- 同名模型在旧平台维度可能有不同价格，先拆分共享配置再取消平台列，禁止取高价合并。
CREATE TABLE IF NOT EXISTS byteseek_pricing_scope_migration (
    original_config_id BIGINT NOT NULL,
    platform TEXT NOT NULL,
    config_id BIGINT NOT NULL,
    PRIMARY KEY (original_config_id, platform)
);

DO $$
DECLARE
    source RECORD;
    platform_name TEXT;
    first_platform TEXT;
    new_config_id BIGINT;
    new_rule_id BIGINT;
    new_price_id BIGINT;
    rule RECORD;
    card RECORD;
    interval_row RECORD;
    config_json JSONB;
BEGIN
    -- 已迁移时只保留映射，不能重新以旧配置覆盖新管理员操作。
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema()
        AND table_name='pricing_config_model_pricing' AND column_name='platform') THEN RETURN; END IF;
    FOR source IN SELECT c.* FROM pricing_configs c
        WHERE NOT EXISTS (SELECT 1 FROM byteseek_pricing_scope_migration m WHERE m.original_config_id=c.id OR m.config_id=c.id)
        ORDER BY c.id LOOP
        first_platform := NULL;
        FOR platform_name IN
            SELECT platform FROM (
                SELECT p.platform FROM pricing_config_model_pricing p WHERE p.pricing_config_id=source.id
                UNION SELECT p.platform FROM pricing_config_account_stats_model_pricing p
                    JOIN pricing_config_account_stats_pricing_rules r ON r.id=p.rule_id WHERE r.pricing_config_id=source.id
                UNION SELECT g.platform FROM groups g JOIN pricing_config_groups cg ON cg.group_id=g.id
                    WHERE cg.pricing_config_id=source.id
            ) platforms WHERE platform IS NOT NULL ORDER BY platform LOOP
            IF first_platform IS NULL THEN
                first_platform := platform_name;
                new_config_id := source.id;
            ELSE
                new_config_id := nextval(pg_get_serial_sequence('pricing_configs','id'));
                config_json := to_jsonb(source) || jsonb_build_object('id',new_config_id,
                    'name',left(source.name,50)||' ['||source.id||'/'||platform_name||']');
                INSERT INTO pricing_configs SELECT * FROM jsonb_populate_record(NULL::pricing_configs,config_json);

                FOR card IN SELECT * FROM pricing_config_model_pricing
                    WHERE pricing_config_id=source.id AND platform=platform_name ORDER BY id LOOP
                    new_price_id := nextval(pg_get_serial_sequence('pricing_config_model_pricing','id'));
                    INSERT INTO pricing_config_model_pricing SELECT * FROM jsonb_populate_record(NULL::pricing_config_model_pricing,
                        to_jsonb(card)||jsonb_build_object('id',new_price_id,'pricing_config_id',new_config_id));
                    FOR interval_row IN SELECT * FROM pricing_config_pricing_intervals WHERE pricing_id=card.id ORDER BY sort_order,id LOOP
                        INSERT INTO pricing_config_pricing_intervals SELECT * FROM jsonb_populate_record(NULL::pricing_config_pricing_intervals,
                            to_jsonb(interval_row)||jsonb_build_object('id',nextval(pg_get_serial_sequence('pricing_config_pricing_intervals','id')),'pricing_id',new_price_id));
                    END LOOP;
                END LOOP;

                -- 成本统计规则也按相同平台拆分，保留匹配账号/分组与规则顺序。
                FOR rule IN SELECT * FROM pricing_config_account_stats_pricing_rules WHERE pricing_config_id=source.id ORDER BY sort_order,id LOOP
                    new_rule_id := nextval(pg_get_serial_sequence('pricing_config_account_stats_pricing_rules','id'));
                    INSERT INTO pricing_config_account_stats_pricing_rules SELECT * FROM jsonb_populate_record(NULL::pricing_config_account_stats_pricing_rules,
                        to_jsonb(rule)||jsonb_build_object('id',new_rule_id,'pricing_config_id',new_config_id));
                    FOR card IN SELECT * FROM pricing_config_account_stats_model_pricing WHERE rule_id=rule.id AND platform=platform_name ORDER BY id LOOP
                        new_price_id := nextval(pg_get_serial_sequence('pricing_config_account_stats_model_pricing','id'));
                        INSERT INTO pricing_config_account_stats_model_pricing SELECT * FROM jsonb_populate_record(NULL::pricing_config_account_stats_model_pricing,
                            to_jsonb(card)||jsonb_build_object('id',new_price_id,'rule_id',new_rule_id));
                        FOR interval_row IN SELECT * FROM pricing_config_account_stats_pricing_intervals WHERE pricing_id=card.id ORDER BY sort_order,id LOOP
                            INSERT INTO pricing_config_account_stats_pricing_intervals SELECT * FROM jsonb_populate_record(NULL::pricing_config_account_stats_pricing_intervals,
                                to_jsonb(interval_row)||jsonb_build_object('id',nextval(pg_get_serial_sequence('pricing_config_account_stats_pricing_intervals','id')),'pricing_id',new_price_id));
                        END LOOP;
                    END LOOP;
                END LOOP;
                UPDATE pricing_config_groups cg SET pricing_config_id=new_config_id FROM groups g
                WHERE cg.group_id=g.id AND cg.pricing_config_id=source.id AND g.platform=platform_name;
            END IF;
            INSERT INTO byteseek_pricing_scope_migration VALUES(source.id,platform_name,new_config_id);
        END LOOP;
        IF first_platform IS NOT NULL THEN
            DELETE FROM pricing_config_model_pricing WHERE pricing_config_id=source.id AND platform<>first_platform;
            DELETE FROM pricing_config_account_stats_model_pricing p USING pricing_config_account_stats_pricing_rules r
            WHERE p.rule_id=r.id AND r.pricing_config_id=source.id AND p.platform<>first_platform;
        END IF;
    END LOOP;
END $$;

-- 旧成本开关继续保留给兼容转换，不能在生成等价成本规则前删除。

-- 分组图片行为统一在协议控制中设置，保留账号覆盖和全局默认值。
UPDATE groups
SET routing_policy = routing_policy #- '{features_config,codex_image_generation_bridge}'
WHERE (routing_policy -> 'features_config') ? 'codex_image_generation_bridge';
