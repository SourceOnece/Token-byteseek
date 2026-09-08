# 账号维护

本文描述上游账号凭据刷新、健康测试、配额与能力探测、临时不可调度和自动恢复的后台流程。它不定义创建表单字段或请求内调度评分；这些由平台专题与调度架构拥有。

## 章节导航

- [凭据刷新](#凭据刷新)：修改 refresh 候选、并发、重试或状态同步时读取。
- [状态与临时不可调度](#状态与临时不可调度)：修改错误、限流或恢复时间时读取。
- [账号测试与自动恢复](#账号测试与自动恢复)：修改定时测试或恢复条件时读取。
- [额度与能力探测](#额度与能力探测)：修改上游 usage、quota 或 endpoint capability 时读取。
- [运维诊断](#运维诊断)：排查刷新堆积、误封禁或账号抖动时读取。

<a id="account_credential_refresh"></a>
## 凭据刷新

`TokenRefreshService` 分页读取需要维护的账号，按平台 refresher 判断资格，并对每个 provider 应用独立并发/QPS 门槛、单次 attempt 超时、周期总超时和有界退避。OAuth、Setup Token 和 Qoder COSY 的候选规则不同；API Key、Bedrock 和 Service Account 通常由各自请求路径或签名 provider 管理，不应统一假设有 refresh token。

刷新成功后要原子更新凭据/过期时间，清理可恢复错误，并同步账号缓存与调度快照。OpenAI/Antigravity 还可在刷新后确保 privacy 状态。刷新失败按失败阈值记录，不立即把一次瞬时网络错误等同于永久禁用；凭据明确撤销或账号归属失效时才进入需要重新授权的状态。

请求路径 token provider 仍会在使用前检查过期偏移，并用账号级锁避免并发刷新。后台刷新降低热路径延迟，但不是唯一正确性来源；两条路径必须使用相同的凭据版本/CAS 保护，避免旧请求覆盖新 token。

## 状态与临时不可调度

账号长期状态、`schedulable`、全账号限流、模型限流和临时不可调度规则是不同层次：

- 凭据/配置错误可以记录 recoverable error 并要求人工重新授权。
- 429、明确 reset time 或短期网络/供应商故障使用恢复时间，在到期前过滤账号或模型。
- 管理员策略和代理过期可能临时移出调度，但不删除账号。
- 账号到期可由维护任务自动暂停；重新启用前仍需验证凭据和关联资源。

状态写入要携带凭据快照或版本条件。较早请求不能在新凭据生效后再次设置旧错误；恢复同样不能清除另一个请求刚确认的永久错误。

## 账号测试与自动恢复

管理端即时测试和 `scheduled-test-plans` 使用平台测试服务调用真实凭据/模型，并保存测试结果。计划使用分钟级 cron 表达式；每个计划可配置自动恢复。成功测试可以清除符合条件的 error、rate limit、temporary unschedulable 和模型限流，但不能绕过管理员禁用、账号过期或类型不匹配。

测试本身应使用受控超时、代理/TLS 路由和脱敏日志。一个模型测试成功只证明该路径当时可用，不证明所有 endpoint capability 或媒体资格。失败结果需区分认证、模型、配额、代理、TLS 和上游容量，以免自动恢复形成启停抖动。

Kimi、Zhipu、DeepSeek 的连接测试按账号 `api_protocol` 选择原生 Anthropic Messages、OpenAI Chat Completions 或 DeepSeek Responses 端点，不得始终假设 Chat 形状。测试复用账号自定义 Base URL、代理、TLS 指纹和受保护 Header Override；Anthropic 协议的自定义中继在模型同步等 OpenAI 格式请求中只移除末尾 `/anthropic`，不能改回官方 host 或丢弃此前的路径前缀。

管理端连接测试请求必须显式选择 `test_type=text|image` 并传入同一字段的自定义 `prompt`。文字测试不再因为模型名称包含图片标记而切换端点；图片测试也不再依赖模型名称命中规则，而是由 OpenAI、Gemini 或 Grok 账号的平台图片端点执行。OpenAI 的 `compact` 与 `legacy_compact` 仅执行固定载荷的能力探测，不显示或使用自定义提示词。未携带 `test_type` 的历史调用才允许回退到旧模型名判断。图片和文字的结果分别通过 SSE 图片事件和内容事件返回；不支持图片端点的平台应直接返回可诊断的错误，不得静默改成文字测试。

<a id="codex_quality_testing"></a>
## Codex 题目测试与调度联动

管理员可选中独立 OpenAI OAuth 账号执行批量题目测试。使用指定模型、reasoning.effort、题目和单一大小写敏感的包含关键词，仅完整可见回答参与判定：命中为“满血”，未命中为“降智”，网络/认证/空回答/不完整响应为“测试失败”。这些标签是管理员自定义关键词结果，不是模型能力的客观认证。管理员必须明确确认调度影响；满血设置 schedulable=true，降智或失败设置 false，不清除 inactive/error、到期或额度门禁。未选、跳过、取消和配置已变化的账号不改调度，已完成的结果不会因取消余下任务自动回滚。

结果独立保存到 codex_quality_tests，每账号只保留最近一条，不把长题目/回答加入账号 extra、调度缓存或用户接口。数据库租约阻止多实例同时测试同账号，结果和 schedulable/outbox 原子提交；输入快照发生变化则不应用旧结果。新批次每实例单批、并发 1–5（默认 3）、最多 500 个账号、每账号 120 秒；测试消耗上游额度。前端经管理员 SSE 获取结果并支持停止未完成测试，满血率分母是满血+降智+失败，跳过/取消/过时结果不计入。原单账号连接测试及定时恢复逻辑保持不变，本功能不添加永久禁止管理员手工改调度的门禁。

专用上下文只在此入口存在：抑制原探针额外写用量快照、401 永久错误和 429 状态恢复，调度修改统一交给结果存储。普通连接测试、定时测试和网关请求仍走原分支。完整回答支持 Responses delta 或 completed 中的最终 assistant output_text；最多读取 8 MiB SSE、单事件 1 MiB、可见回答 128 KiB，不完整/空回答/超限均按失败停调。题目不超过 16000 字符、关键词 200 字符，普通审计只记动作不复制题目。管理员列表默认读取摘要，单账号 detail=true 才读取完整输入输出。

账号 updated_at 会被正常用量更新刷新，因此先比较凭据、代理、账号状态/开关、TLS/客户端规则等实际配置，只有这些保持相同才采用最新 updated_at 做最终 SQL CAS；CAS 冲突保留 stale 结果，不改调度。运行期间外部策略或管理员仍可能改变资格，满血只证明该账号本次模型/题目/档位的关键词结果，不保证所有模型或 transport 均可用。关闭浏览器/弹窗会取消在途测试而不回滚已完成结果；应用异常退出后租约 150 秒到期，重启不自动继续未完成批次。

## 额度与能力探测

平台可维护独立的上游额度快照：OpenAI/Codex 窗口、Gemini tier/model quota、Antigravity credits、Grok 计费/媒体资格、Qoder Credits，以及 Kimi/Zhipu/DeepSeek 的统一用量监控快照等。快照用于调度、容量展示和诊断，不是 TokenRouter 用户余额或订阅账本。

OpenAI 重置次数查询把带到期时间的完整结果保存为账号展示快照；上游只返回正数次数却缺少到期明细时，实时结果仍返回给调用方，但旧快照必须保留。直接调用重置 API 成功消费次数后，服务先在脱离客户端取消信号的有界上下文中恢复账号 error、限流和临时不可调度状态，再回读额度快照与最新账号投影；恢复不修改人工 `schedulable` 开关。后续步骤部分失败时响应使用 `cache_refreshed`、`account_state_recovered` 和 `warning_code` 明确区分，调用方不得把已消费的次数当作可重试失败。

OpenAI API Key 的 Responses 探测只维护 `extra.openai_responses_probe_status`，取值为 `supported`、`unsupported`、`unknown`；管理员路由策略 `extra.openai_text_route_mode` 和 HTTP continuation 能力开关 `extra.openai_responses_continuation_supported` 不属于探测服务，任何探测结果都不得覆盖它们。2xx 响应若仍因 `max_output_tokens` 未完成，或响应状态为 `failed`，应保持 `unknown`；完成但没有 `function_call` 的响应判定为 `unsupported`。网络错误、响应读取失败和其它结论不足的结果保留最近状态。账号默认连接测试以 Responses 为首选协议，路由策略显式强制 Chat 时才使用 Chat 测试路径。HTTP continuation 缺失时按关闭处理；嵌套 Sub2API 账号应显式关闭，直连且确认支持的 API Key 账号才开启。

账号创建、编辑、批量更新和导入会把旧 OpenAI 配置规范化为新字段；复制账号保留 `credentials.openai_workload_capabilities`、`openai_text_route_mode` 与 continuation 能力开关，丢弃已有探测状态并重新探测。调度投影必须同时包含工作负载集合、路由策略、探测状态和 continuation 能力开关，探测状态变化要触发投影失效。Ollama Cloud 等兼容 API Key 上游可以保存其管理会话和用量快照，但只有明确匹配的账号才进入探测，不能把探测协议推广到所有 `apikey`。Grok 计费与媒体资格、各平台额度探测也继续使用各自独立协议。

通用的上游声明倍率探测已移除，不再有定时任务、手动操作、快照或公开账单自省接口。账号创建、编辑、批量更新、复制、CRS 同步和仓储写入都会丢弃历史 `upstream_billing_probe` 与 `upstream_billing_probe_enabled` 键；这项清理不得影响 Ollama Cloud 会话/用量、endpoint capability 或其它额度状态。

实时探测失败时保留最近成功快照并同时暴露当前错误，不把旧数据标为实时。任何配额耗尽或 capability 变化都要触发相关调度投影失效。

## API Key 上游用量查询

API Key 上游用量由独立的 `UpstreamUsageService` 提供，和 OAuth/Setup Token 的 `AccountUsageService` 语义分离。它只服务管理员展示，不参与调度、自动暂停、倍率、本地配额或结算；列表加载、滚动和自动刷新都不会产生上游流量。管理员手动查询时，服务按账号和规范化配置指纹合并并发请求，单次约 60 秒超时、512 KiB 响应体上限、禁止重定向，并复用代理、TLS 指纹、Header Override 和既有 `HTTPUpstream`。

配置缺失时，普通 API Key 默认启用 Sub2API；New API 和 Zivv 必须显式选择对应适配器。CN API Key 则按平台/mode 自动选择 Kimi 余额、Kimi Coding、Zhipu Coding 或 DeepSeek 多币种余额适配器，Zhipu payg 明确不支持。Sub2API 的钱包负余额可以展示，`remaining=-1` 只在适配器内部转换为 `unlimited=true`。New API 用 `/api/usage/token/` 读取 Key 配额，再通过固定的钱包端点或受保护的用户访问令牌查询用户钱包；只有钱包结果进入 `balance`，Token 配额进入 `limits`/`subscription`。Zivv 用 `/v1/user/balance` 同时读取钱包、累计用量、Key 限额和套餐，`key_limit=0` 显示为不限量。`unlimited_quota=true` 时忽略上游可能溢出的额度字段，状态接口失败时使用默认单位比例；官方实例未配置用户访问令牌且不提供 API Key 钱包端点时返回 `UPSTREAM_USAGE_WALLET_UNAVAILABLE`，不把 Token quota 当钱包余额。手动查询失败不修改账号状态或运行快照，也不自动回退到另一个协议。

浏览器只缓存成功的归一化结果五分钟，缓存键隔离管理员身份、账号 `updated_at`、代理/Base URL 和配置；失败不缓存，账号凭据、代理或配置变化立即失效。审计仅记录管理员动作和脱敏元数据，不记录 API Key 或上游原始响应。该功能与已移除的 `upstream_billing_probe` 完全不同，不恢复旧的自动倍率探测。

CN 周期监控默认关闭；启用后只把统一快照写入 `extra.cn_usage_monitor_snapshot`，用身份 hash 和账号 `updated_at` CAS 防止旧探测覆盖新凭据。失败保留最近成功结果，余额低于阈值只写带同一身份 hash 的临时不可调度原因，恢复也只清理由该身份创建的状态。多实例同轮由 leader lock 串行化，自定义中继必须命中启用的 URL allowlist。详细字段、适配器和超时语义见[API Key 上游用量查询](../interfaces/upstream_usage.md)。

## 运维诊断

- 观察每 provider 的候选数、刷新成功/失败、节流、超时和最长积压，而不只看总成功率。
- 关联账号测试、刷新、quota probe、代理健康和调度过滤原因，区分凭据故障与出站网络故障。
- 检查账号数据库状态、当前进程投影和跨实例失效是否一致；手工改数据库后等待周期重建不等于即时生效。
- 自动恢复或批量导入后抽查实际协议，避免仅凭 token endpoint 成功误判推理可用。
- OpenAI Chat 排障应同时核对入站协议、`openai_text_route_mode` 和 Usage Log 的 `upstream_endpoint`；默认模式应记录 `/v1/chat/completions`，不能因探测为 `supported` 而变成 `/v1/responses`。

相关文档：[上游账号能力矩阵](../interfaces/upstream_account_matrix.md)、[账号调度与缓存一致性](../architecture/account_scheduling_and_cache.md)、[上游传输安全](upstream_transport_security.md)。
