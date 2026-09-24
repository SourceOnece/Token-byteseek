# OpenCode 上游

本文记录 ByteSeek 对固定上游批次中 OpenCode Zen/Go 平台的适配，不描述 OpenCode 客户端本身，也不代表上游实时模型权益或价格保证。平台标识为 `opencode_go`，账号仅支持 API Key；Zen 与 Go 是同一平台下的账号模式，不是两个用户计费平台。

<a id="opencode_protocol_routing"></a>
## 模式与协议

`credentials.account_mode` 为 `zen` 或 `go`；缺失沿用 Go。`api_protocol` 可以固定 Chat Completions、Responses、Anthropic，或使用默认 `adaptive`。Zen 默认根为 `https://opencode.ai/zen/v1`，Go 为 `https://opencode.ai/zen/go/v1`；Messages 的默认根去掉末尾 `/v1`，拼接器识别版本段，避免 `/v1/v1/messages`。自定义端点保留其主机和路径前缀，仍受原 URL/代理/TLS 安全策略约束。

自适应协议按账号模型映射后的最终模型判断，与客户端的三种入站协议正交。`protocol_rules` 支持最多 64 条精确模型名或末尾 `*`，单模式最多 128 字节，按从上到下首次命中选择协议，未命中使用 Chat。缺失或 null 使用模式对应内置规则；显式 `[]` 意味全部使用 Chat，不等价于默认规则。Go 的 GPT/Grok/Muse 使用 Responses、MiniMax/Qwen 使用 Messages；Zen 的 GPT/Grok/Muse 使用 Responses、Claude/Qwen 使用 Messages，其余 Chat。管理员固定协议优先于规则；后台探测状态不覆盖这些规则。

三个公开入口复用现有转换服务、原模型恢复和计费路径。普通 OpenAI 的平台身份、OAuth、Codex 指纹模式、Compact 和调度资格不因新增平台改变。连接测试只检测管理员选中模型对应的实际原生协议，不把一个模型连续发送到三个端点。模型预设来自本批固定源码；实际账号可用模型以显式配置/同步结果和调度资格为准，不推断价格或上下文长度。

## 会话与用量

新平台请求中的 `X-OpenCode-Session` 由客户端显式会话头、`prompt_cache_key` 或可解析的 Claude 会话元数据取得；派生时隔离本平台 API Key、上游账号、凭据和原始会话。相同身份的同一对话稳定，换账号或换 Key 后不同。缺少会话线索时生成请求级随机值，同一次请求重试复用，但不承诺跨请求缓存命中；普通 `metadata.user_id` 不是会话，不能把该用户全部对话折叠到一个 ID。转换前只缓存候选字符串，不额外保存大请求体。

该会话规则仅作用于显式 OpenCode 平台，包括管理员配置的 OpenCode 中继。其他平台维持历史行为：仅向 HTTPS 官方 `opencode.ai` 原样转发客户端已有 OpenCode 会话头。不是撤回的 Codex Metadata 补齐功能，不给其他平台添加这一新规则。

Go 的手动用量查询复用本地 `UpstreamUsageService`，读取配置前缀下的 `/v1/usage`，归一化 rolling/weekly/monthly 到 5h/weekly/monthly；Zen 没有本批采用的订阅窗口查询，明确不支持。百分比不冒充货币余额，失败不清旧成功快照。周期监控仍默认关闭，启用后使用账号身份指纹、CAS 和现有 outbox。429 采用匹配身份的窗口重置或可信的未来响应重置时间，未知时沿用原兜底冷却；不改变用户账单。

## 配置与兼容

bh.061 使 OpenCode 平台及精确 HTTPS 官方 OpenCode/Command Code 端点使用规范 User-Agent；普通/透传 Responses、Chat、原生 Messages 与相应账号测试共用该规则，账号显式 Header 覆写随后生效。明确 Cloudflare 1010 边缘拦截不消耗 403 账号错误次数，真实权限错误照旧。Go 的 /zen/go 根地址兼容补齐，但不采用 sub2api 新增的独立窗口后台和同 Key 共享状态。

管理表单复用现有 Select、BaseDialog、规则列表和包豪斯蓝色重点。模式切换只替换仍等于旧默认的端点/规则，自定义值保留；保存空规则保留空数组。新平台接入现有分组、渠道、额度、错误规则和调度快照，不恢复已移除的 Composite 平台或另一套监控表。

迁移 `274_add_opencode_platform_quota.sql` 对应 sub2api 原 238，仅扩展本地用户平台额度 CHECK。新增平台额度缺失仍为无限，新注册的十一平台批量写入须与 Ent 校验一致。回滚二进制前应停用新平台账号/分组，否则旧实例没有对应平台处理器；迁移记录不得删除。

相关文档：[账号能力矩阵](upstream_account_matrix.md)、[模型目录](model_catalog_and_marketplace.md)、[上游用量](upstream_usage.md)、[网关策略](../domains/gateway_policy_controls.md)、[本批版本](../operations/versions/v0_1_278_bh_030.md)。
