# 网关请求生命周期

本文描述 AI 网关请求从 Gin 路由进入到认证、模型归一化、提供商调度、上游转发、故障转移和用量结算的共同阶段。它用于修改跨协议热路径时保持顺序和失败语义，不枚举全部端点、供应商字段或某个平台的模型能力。

## 章节导航

- [入口族与处理器](#入口族与处理器)：确定请求由哪个协议分支拥有。
- [共同处理管线](#共同处理管线)：核对不可随意交换的阶段顺序。
- [认证与准入](#认证与准入)：修改 API Key、权益或请求上下文时读取。
- [模型名称链](#模型名称链)：修改模型映射、列表或响应恢复时读取。
- [提供商选择与故障转移](#提供商选择与故障转移)：修改调度、并发、粘性或重试时读取。
- [转发与流式边界](#转发与流式边界)：修改上游调用和错误返回时读取。
- [用量与结算](#用量与结算)：修改记录、价格或扣费时读取。
- [审核与搜索协作](#moderation_search_boundaries)：修改裁决、搜索额度与工具模拟时读取。
- [扩展约束](#扩展约束)：新增入口或平台时检查。
- [Qoder Chat 请求编排](#qoder_gateway_execution)：修改 Qoder Chat 的准入、尝试和完成顺序时读取。
- [平台执行与资源拥有权](#upstream_attempt_ownership)：修改单次执行、流输出及关闭边界时读取。

## 入口族与处理器

`RegisterGatewayRoutes` 在面板 `/api/v1` 之外注册客户端协议入口。核心入口族为：

| 入口族 | 主要用途 | 处理器分派 |
| --- | --- | --- |
| `/v1`、裸 `/models`/`responses` 等兼容别名 | Anthropic Messages、OpenAI Responses/Chat/Embeddings、图片、视频、模型与用量 | 进入 app 固定绑定的统一文本、媒体与模型入口，选号后按实际提供商执行 |
| `/v1beta` | Gemini 原生模型、生成、流式生成、token 统计 | Google 形状认证后进入 `GeminiNativeHandler` 或 `ModelsHandler` |
| `/antigravity/v1`、`/antigravity/v1beta` | 强制 Antigravity 平台的 Claude/Gemini 专用入口 | 在上下文写入 force platform，再复用通用 handler 与调度 |
| `/backend-api/codex` | Codex/ChatGPT 风格 Responses、Realtime 和 sideband | `OpenAITextHandler`、`ResponsesWSHandler` 与 `LiveHandler`；部分路径有专用认证/路由限制 |
| 批量图片管理 | 提交、查询、下载、取消和清理任务 | 专用 handler/service；查询类入口只按任务归属认证，不重新选择模型提供商 |

普通 `/v1/messages`、`/v1/responses` 和 `/v1/chat/completions` 共用文本提供商循环。HTTP 路由不判断分组平台；`UnifiedTextExecutor` 在选号后复用实际提供商的平台单次执行器。媒体、WS 和 Live 仍保留各自的资源生命周期。专用 `/antigravity/*` 入口附加强制提供商平台过滤，该约束在分组回退后仍生效。

公开入口不再注册 `/v1/sub2api/billing`。它不是模型、用量或结算管线的别名，访问时直接得到普通 `404`，也不会进入 API Key 非消费请求分支。

<a id="gateway_pipeline"></a>
## 共同处理管线

HTTP 入口由 app 固定构造。组合根分别提供原生执行器、会话和资源，路由直接接收需要的端口。OpenAI Responses、Chat 与 Messages 的 HTTP 绑定直接接收唯一用户/图片槽资源、Cyber、审核、归属读取及资金端口；`gateway/httpapi/openaiattempt.Runtime` 直接绑定平台单次能力、同一调度反馈、槽位和完成器，每次 Open 创建独立尝试状态；Responses、Chat、Messages 通过 `UnifiedTextExecutor` 调用所选提供商的固定单次执行器。

WS 的入站与每轮单步端口由 `gateway/httpapi/wsentry` 直接装配，与文本运行时共享同一份尝试绑定；图片、视频、音频、Embeddings 与 Alpha Search 的 HTTP 请求适配和完成捕获由 `gateway/httpapi/mediaentry` 直接装配，也共享原生失败输出与资源释放。Wire 显式绑定执行器与完成器，契约测试使用相同绑定与函数句柄夹具；平台单次交换与 WS relay 直接绑定原生执行器的固定端口。

图片单次执行直接绑定 OpenAIImagesExecutor，共用 OpenAIRequests、OpenAIResponseOutput 和应用活动屏障；图片工具冷却通过提供商端口写入。图片意图提示由 HTTP 按尝试保存，分组改写后重新判断，不把请求级提示误用于下一提供商尝试。`gateway/text` 拥有文本提供商循环与计数预检的独立预算，`gateway/requeststate` 拥有报文副本、引导规范化和请求内模型替换缓存；`gateway/modeltrace` 维护响应恢复链。

`forward` 组织通用请求准备和转换推进，技术 provider/HTTP Adapter 执行交换、读写与 Flush。平台单次执行由 upstream 与 gateway/provider 提供，不创建第二套提供商切换循环。

WS 执行使用明确的静态选项和请求、输出、会话及选择端口，核心不读取完整应用配置。连接池仍在首次使用时启动，关闭屏障使它在退出后不能被重新创建。Grok、Live 和 WS共用拨号器，Agent Identity 凭据失效也作用于同一池。入站、池化、透传及HTTP桥接继续使用各自原有恢复和取消规则，测试直接验证原生帧执行与共享状态。

Responses 的固定执行器拥有请求准备、转换和 HTTP 单次执行，复用已有 OpenAIRequests、OpenAITextExecutor 和输出实例。协议转换和 Compact 错误恢复不重新运行全局提供商循环。失效密文读写在 HTTP 与 WS 间共享原会话存储及 TTL，转入 WS 时不再重做模型映射或请求变换。

文本入口把 `requeststate.ExecutionHints` 和 `RoutingState` 显式传给执行器，分别携带客户端识别、图片意图、粘性预取等执行提示，以及原生分组、路由计划和客户端协议。分组在写入和读取边界复制，后续 attempt 重新绑定变更后的分组，不能修改先前请求快照。各执行 Adapter 读取同一显式状态；telemetry 只保存观测关联信息。

普通三个文本入口使用 `gateway/httpapi/openaiattempt.Runtime`；强制 Antigravity 和 Gemini 原生入口继续使用 `textattempt.Runtime`。app 固定绑定平台单次调用、选择反馈与原生资源，每次 Open 只创建请求和尝试状态。Messages 在选号前解析 Claude Code、版本和 Thinking 身份；选号后才执行平台专有策略、原生 warmup 拦截与串行队列。

app 绑定平台单次执行端口；提供商循环与完成规则仍由原生 text/completion 拥有。执行边界采用 `gateway/provider.SelectionResult`，保留实际提供商目标、等待计划和本次反馈参数；调度核心继续只读取无凭据候选。

OpenAI HTTP 的并发 helper 与本地图片限制器由 app 构造为唯一 `OpenAIHTTPResources`，文本、媒体、WS 和剩余兼容入口共用同一实例。资源对象只接收静态图片限制参数；用户槽仍按进入等待前的请求 context 绑定取消释放，图片等待、拒绝和独立作用域保持。

`gateway/searchtools` 组织工具模拟，`gateway/moderationflow` 固化审核完成输入；`completion.Recorder` 消费独立资金与用量快照。`ws`、`live` 各自管理连接/turn 状态；摘要、隔离和归属值由 `session` 提供，Redis 协议由 `rediscache` 适配。错误规则与不可变发布快照位于 `errorpolicy`，不承担调度健康或重试决策。

```text
请求体/连接限制、request ID、Ops 采集
                 |
             API Key 认证
                 |
       复合 Key 选组 -> Key 级模型重定向
                 |
       用户/团队/分组/IP/权益准入
                 |
    RequireGroup + 客户端协议准入门禁
                 |
             协议 handler 分派
                 |
  解析与归一化 -> 内容策略 -> 用户并发槽
                 |
          等待后的权益二次检查
                 |
  会话/分组策略/能力解析 -> 提供商选择 -> 提供商并发槽
                 |
       请求转换、凭据/代理和上游转发
                 |
      可重试错误 -> 受限故障转移循环
                 |
  成功响应/流 -> 用量解析 -> 幂等结算 -> 记录
                 |
      响应模型元数据恢复与 Ops 完成采集
```

这条管线有几个不能交换的约束：

- 复合 Key 必须先用客户端的完整 `前缀/模型` 选择分组，Key 级模型重定向再处理去前缀后的模型。
- 客户端协议准入必须使用普通 Key 的绑定分组或复合 Key 最终选中的分组；拒绝发生在协议 handler、提供商选择和计费之前。
- 用户并发等待可能跨越余额、订阅或额度变化，获取用户槽后必须通过 BillingCache 再检查一次权益。
- 模型权限、分组白名单和提供商资格必须基于逐层解析后的对应模型，不能用客户端别名直接替代最终路由模型。
- 上游成功后才增加相应 RPM 软计数并安排正常用量结算；本地拦截、内容拒绝和上游失败使用各自独立的审计/运维记录语义。
- HTTP 200 中的失败事件使用语义状态执行提供商策略。WS 桥已执行的副作用通过请求自己的 `ResponseFailureEffects` 交给输出端消费一次，避免重复处理；HTTP 提交、重试窗口和语义输出继续分开记录。
- 响应别名恢复只改协议元数据字段，不能替换正文中恰好相同的字符串。 工具恢复状态由 `requeststate.ResponseTools` 按请求和 turn 持有，WS 会话更新不能改写仍在输出的 turn；协议算法与 HTTP 输出适配分别位于 `protocol/bridge` 和 `gateway/httpapi`。

<a id="apikey_authentication"></a>
## 认证与准入

凭据提取和认证错误展示由 `apikey/httpapi` 承接，Key、用户、团队和 IP 校验进入 `apikey.Authenticate`，返回区分 owner/payer/actor/team 的 `AccessSnapshot`。`gateway/httpapi` 的通用/Google 认证入口组合复合选组、模型改写与 `gateway/admission` 的资金准入，HTTP 适配负责请求上下文与观测投影。普通协议门禁不提前读取请求体，Google 与通用入口仍各自保留原错误顺序。

认证缓存使用 v41、原 Redis key、TTL 和失效协议；来源与请求中的嵌套 map、slice、指针分别复制，复合选组不能污染共享快照，分组显式 Fast 策略也必须完整往返。

通用认证入口在最终选组授权后绑定原生 `AccessSnapshot` 和 Fast 策略，付款用户仍取该请求的付款主体；Google 分支保留原先独立的绑定时机。认证失败时供 Ops 使用的已加载 Key 信息与已认证快照分开，加载到记录并不代表授权成功。

通用 API Key 认证依次执行：

1. 对无效认证滥用和过大 header 做入口限制；拒绝通用网关的 query API Key，接受 `Authorization: Bearer`、`x-api-key`，并为 Gemini 兼容 `x-goog-api-key`。
2. 从认证缓存/仓储加载 Key 以及必要的 User、Group、Team 和复合映射。加载失败按不存在、过载、团队生命周期或内部错误区分。
3. 始终校验 Key 禁用状态、团队 Key 生命周期、成员限额、IP 规则、用户存在与启用状态。
4. 普通 Key 必须明确绑定分组，历史未绑定 Key 的模型调用返回 `GROUP_REQUIRED`；若为复合 Key，从请求模型选中映射并得到本次请求的普通 Key 视图。随后校验最终分组可用性、用户权限，并应用 Key 级模型重定向。
5. 鉴权要求明确分组及提供商成员关系，按 Key 的 `auto`、`subscription` 或 `balance` 策略解析资金来源，并对消费入口检查 Key 过期/配额、订阅窗口限额或余额；指定订阅不可用、额度不足或不覆盖最终分组时直接拒绝。
6. 写入 API Key、认证主体、角色、分组和可选订阅上下文。路由随后以最终分组的 `allowed_protocols` 执行协议准入，再进入 handler；`last_used_at` 更新失败不阻断已认证请求。

`/v1/usage` 及部分批任务管理会跳过消费准入，使额度耗尽或 Key 过期后仍可取回或清理自己的数据；它们仍执行身份、用户、团队、IP 和资源归属检查。用量查询为指定订阅保留其失效状态，不把显示来源改成余额。模型列表虽然对复合 Key 不需要选中一个分组，但仍要执行适用的 Key 额度、余额和订阅检查。

不存在自动默认组。不可用组、客户端限制和无效请求只能回退到管理员明确配置的目标；目标必须重新通过用户、团队、协议、模型、订阅和资金检查。运行时回退重新发布有效 Key、分组、RoutePlan 和粘性预取，不修改认证缓存；每个请求只允许既定的有限回退，已开始输出或取消后不能重放。

身份读取直接使用 authctx 的唯一记录；已认证 Key 与 Ops 的失败加载投影由 apikey HTTP 分开读取，失败加载投影不构成授权状态。

普通 Key 的协议门禁不会读取请求体；复合 Key 的认证阶段会先读取并恢复请求体，以模型前缀确定最终分组，然后才执行同一门禁。禁用协议返回客户端协议原生的 `403`，记录 `LocalPolicyDenied`，不进入提供商选择、重试、fallback 或结算。进入协议 handler 后，请求体按端点限制读取并做宽容 JSON/Multipart 处理，再完成用户提示词替换、协议解析、客户端识别、内容审查和 Ops 元数据设置。HTTP 与 WS 直接使用 app 注入的同一个 promptpolicy 实例，规则回源和替换仍发生在这些原始调用位置。用户并发槽位在提供商选择前获取，避免为已经超过用户并发的请求消耗调度资源。

## 模型名称链

一个请求可能同时存在以下名称：

```text
client_model
  -> composite_actual_model
  -> api_key_redirected_model
  -> group_mapped_model
  -> provider/upstream_model
```

- `client_model` 是客户端原始模型；复合 Key 时保留分组前缀。
- `composite_actual_model` 是选组后去掉前缀的模型。
- Key 级重定向是一跳匹配，发生在选组之后、分组与提供商映射之前。
- 分组映射形成白名单、计费和用量映射链的模型候选；白名单检查阶段与计费来源分别配置；提供商映射得到最终供应商路由键。
- `requested_model`、`upstream_model` 和去重后的 `model_mapping_chain` 分别保存客户端意图、实际发送模型和变换路径。

协议 handler 可以在每次 failover attempt 重新基于所选提供商构造请求，但不得对已经解析过的一跳映射再次递归。模型列表需要从当前可请求目标反推可展示别名；保存映射时允许目标暂时不可路由，真正请求仍使用标准无提供商/无模型错误。

`routing.RoutePlan` 保存当次最终分组、入口协议和 Key→分组模型链，handler 把它交给后续候选解析。模型链快照同时保留分组白名单的 `RestrictModels`、`RestrictionModelSource` 和独立的计费模型来源，投影为 `GroupMappingResult` 时完整传递。候选仍按当前提供商快照和最终分组复核协议；提供商映射在原使用时点读取，不提前固定提供商或把 attempt 结果写入共享缓存。分组发生回退时旧计划不替代重新授权，平台请求改写与响应模型恢复仍由原执行链负责。

`requeststate.AttemptRoute` 固化本次候选结果和协议，`RoutingState.ResolveAttempt` 在 fresh/DB 复核后重新解析。提供商持久记录不承载该状态；协议相关地址和 CN 适配规则由 `provider.ProtocolTarget` 组合显式协议与记录计算。执行 Adapter 使用只组合原生 Record 与路线的 ExecutionProvider；读取模型映射仍发生在原调用时点，不能把上一次尝试结果写回共享提供商缓存。

候选协议快照、协议准入与模型冷却判断统一由 gateway/provider.ModelPolicy 组合原生提供商记录和本次 AttemptRoute。选择与诊断保留“协议 → 提供商状态 → 模型窗口”的检查顺序；原始窗口是否冷却、允许 overages 时的可调度性和剩余时间分别计算，不能互相替代。模型目录与候选协议复核使用同一快照投影，不因此提前读取动态模型映射。

提供商缺失或空模型范围使用平台及认证类型的默认目录；非空清单定义显式范围，精确映射可引入自定义模型。末尾通配符参与匹配但不作为模型名展示，显式 `*` 仍不能绕过协议、认证与端点硬能力。`protocol_fallbacks` 对每个入口保存有序目标数组：缺省自动匹配，空数组仅原生，非空数组限制转换目标；每个候选先原生、再尝试已有单步转换。

计费模型与实际上游模型仍分别解析；OpenAI 普通/Compact 映射和 Bedrock 区域路由直接使用原生 ModelPolicy。Lite 标记由入口确认后，平台适配按提供商契约选择完整 OAuth 工具规范化或 API Key 的并行工具限制，保持原入口范围。

协议地址读取直接使用 provider.ProtocolTarget，RPM、会话数量、串行队列及窗口配置直接使用 provider.RuntimeConfig，执行边界按调用时点读取字段。Codex 图片桥接覆盖、显式工具策略与指纹模式由 provider 持有，仍保留平台资格、顶层/嵌套设置优先级和旧值兼容。提供商 Header 覆写由 provider/provider 在原调用点应用 egress 策略，绑定成方法回调时仍在执行时读取提供商最新字段，不提前固定凭据或覆写表。

Codex 身份和指纹的请求内状态只由 HTTP Adapter 持有；提供商 provider 组合明确输入，平台库只执行纯派生和报文修改。状态发布保持原同步方式，Header 与请求体使用同一组本次 IDs，failover 覆盖顺序不变；异步完成不会持有该 HTTP 状态。

<a id="account_selection_and_failover"></a>
## 提供商选择与故障转移

提供商选择的输入包含本次显式分组、请求模型、分组映射结果、会话 hash、失败提供商集合、端点能力和可选强制平台。先解析客户端限制等显式回退，再以最终分组的 `scheduler_type` 选择基础或高级调度器。候选覆盖组内全部提供商平台，不扫描组外提供商。选择器综合以下约束：

- 组内成员关系、专用入口 force platform、分组/提供商模型范围与一跳映射。
- 提供商启用、过期、代理、凭据、上游资格、临时不可调度、模型/提供商限流及配额状态。
- 调度快照的可用性、粘性会话、优先级/负载、最近使用、并发槽和可等待队列。
- 特殊端点能力，例如图片、Realtime/WS、Grok 付费媒体资格或站点特定模型能力。

候选排序与高级评分不读取上游声明倍率，也没有 `upstream_cost` 权重或 OAuth 参考倍率。提供商本地 `rate_multiplier` 与价格配置的上游计费模型来源仍在提供商选定和模型映射完成后参与结算，不作为候选资格或排序信号。

Messages 的 `count_tokens` 由 app 直接构造原生 HTTP Handler。HTTP 只持有受控计数目标与无凭据提供商快照；原有选择和平台执行原语由组合根连接。资金预检先于无槽选择，按实际提供商平台执行计数，每次尝试从原报文重建分组映射。Anthropic/OpenAI 使用已有执行器，Gemini 保留 OAuth scope 回退，Grok/CN 使用本地计数；Qoder、Antigravity 和 Bedrock 返回不支持。Responses `input_tokens` 同样不获取提供商槽。计数不提交费用或完成任务。文本与计数入口共用原有兼容指标采样计数器。

OpenAI 兼容计数、Grok 本地估算和 Responses 输入 token 预检由 app 独立构造 OpenAITokensHandler，直接绑定同一个请求生命周期屏障。计数保持分组路由规划后检查资金，再执行单次无槽选择；Responses 预检保持资金检查先于分组路由规划，并释放选择器交付的每个提供商槽。Grok 本地估算不增加资金检查、选提供商或上游请求。这些入口没有生成请求的完成提交端口。 计数执行现在直接绑定 OpenAIAuxiliary，路由计划和选择分别使用原生 RoutePlanner 与 Compatible；提供商目标只在受控转发方法内携带凭据。

AlphaSearch 与 Embeddings 同样复用固定请求和响应实例，搜索授权元数据由同一 provider.OpenAIAuthorization 提供。

Live 与 sideband 的 HTTP 入口也由 app 直接构造，原生 LivePorts 共享原审核、资金准入和并发服务。分组协议与功能门禁先于读取请求体；审核先于资金检查，再取得即时用户槽。会话创建、身份归属及 relay 直接绑定 OpenAILiveExecutor；纯会话编排继续由 gateway/live 拥有，存储、租约、拨号器与现有入口共享。observer 的取消表和等待计数由该执行器唯一持有，app 直接登记原关闭阶段；Live 仍只记录零费用用量。非报文模型重定向由 HTTP Adapter 的唯一函数提供，Live 与 WS 保持相同的一跳映射与追踪语义。

客户端会话 Header、Grok 实际选号/强制平台/显式 Header 判定及 OpenAI 会话哈希的请求绑定由 gateway/httpapi 负责；内容种子、Grok 模型隔离种子和 Gemini 摘要格式由 gateway/session 提供。显式信号、内容回退、无状态图片入口和用量日志仍使用各自原有优先级，不把只适用于日志的 Header 扩大为通用选号信号。新旧哈希同时从同一种子派生，旧哈希通过 requeststate 传给既有粘性读取逻辑。Qoder 兼容入口直接调用唯一的通用请求哈希函数。

`basic` 保留历史选择路径。`advanced` 在上述硬约束完成后调用通用评分核心，按 Top-K 加权顺序尝试候选并在每次尝试前复核并发槽。有效 Top-K、权重和粘性开关按最终高级分组逐字段合并：分组 `advanced_scheduler_overrides` 优先于网关运行时设置，缺失字段继续使用全局值；空对象等于全部继承。OpenAI/Grok 在这一核心上附加 previous response、订阅、transport、Compact 与额度能力；其它平台只提供各自已存在的候选与硬过滤。

运行时只对本次实际走高级模式的选择回写错误率、TTFT 和切换统计，基础请求不会污染高级评分。`count_tokens`、可用性探测等仅选提供商入口同样按最终分组决定模式，但使用无槽选择，不占用提供商并发槽或会话数量。

提供商选择由 `scheduler` 的通用、兼容平台和 Gemini 选择器执行，`gateway/provider/selection` 提供平台资格与受控执行目标投影；Messages、文本、媒体、WS/Live、计数及任务消费者由 app 绑定原生选择实例，平台执行由已绑定的原生单次端口承担。`SelectionInput` 使用最终 RoutePlan 和独立提供商候选；每个 attempt、fresh 和 DB 复核重新解析候选，不把模型/协议结果写回共享缓存。

`AcquireUser` 返回请求 Lease 与带计数所有权的 WaitResult；`Lease.Select` 返回当前 AttemptLease。选择结果也可能携带 WaitPlan，由 scheduler 执行等待循环、HTTP 同步观察并输出原心跳。只释放确认取得的等待计数；完整提供商补全失败等后续准备错误立即归还已登记槽位。请求和尝试的组合释放幂等，成功/部分结果的会话保留由 Finish 决定。用户等待完成后仍在原位置复查权益。

故障转移只处理适配器明确包装为 `forward.UpstreamFailoverError` 的可切换错误。该值保留错误阶段、归属、原始响应值及重试投影；平台特有的 OpenAI 容量与请求大小识别留在 gateway/provider，通用契约不导入具体平台。`failover.FailoverState` 记录切换次数、失败提供商和最后错误，并根据提供商 pool-mode 重试次数决定同提供商重试、排除后选择下一个提供商、短暂等待或耗尽。

普通同提供商重试固定等待 500ms；被标记为请求级瞬时故障的容量错误按 500ms、1s、2s、4s 指数退避，后续单次等待封顶 8s，客户端取消会立即打断等待。提供商健康命令按错误分类写入临时不可调度标记。Messages 的重试耗尽冷却会跳过请求级瞬时故障及池模式提供商；HTTP 非 2xx 本身不构成停调条件。

粘性会话已经绑定提供商时，切换提供商可能要求把普通输入按缓存读取计费，以反映缓存不再命中的成本语义。选择耗尽后的单提供商重试和等待有严格上限；客户端 Context 取消必须立即终止，不继续选择或休眠。

<a id="protocol_conversion_boundary"></a>
## 转发与流式边界

Messages、Claude 的 Chat/Responses 转换及 `count_tokens` 使用 `gateway/provider/messageforward.Runtime`。app 固定绑定凭据来源、HTTP 池、健康反馈、TLS、分组策略和搜索实例；普通、API Key 透传、Vertex 与 Bedrock 分支继续调用各自的 upstream 执行器。每次尝试独立持有 Beta 过滤结果、工具名称映射和错误诊断，HTTP Adapter 负责响应提交、Header、Flush 与错误报文。

Messages、计数和 Qoder 的路由计划由 `gateway/provider.RoutePlanner` 连接分组策略读取与 routing，摘要和隔离直接使用 gateway/session；重试耗尽后的兼容冷却由 `provider.RetryCooldown` 读取最新池模式后决定。完成器直接绑定 app 的原生记录器。调试输出由同一个 `requestdebug.Trace` 持有文件句柄，请求和后台工作结束后再关闭。

Gemini 与 Antigravity 的凭据来源、传输和动态读取端口由 app 注入 `gateway/provider/googleforward`。平台准备器不持有 Gin 或完整配置；`gateway/httpapi` 保留三种客户端协议的错误形状、规则覆盖和 Ops 写入顺序。图片计数与工具名恢复状态按 attempt 创建，图片仍取单个响应片段的最大内联图片数；没有观测到图片时才使用原模型名回退。

OpenAI/Grok 共享响应的回合状态头由 `gateway/httpapi.CodexTurnStateHeaders` 处理：首输出暂存阶段不登记来源，实际提交后才写入会话组件的提供商来源表。请求回带值只有在已知来自另一提供商时才移除，未知、同提供商或过期来源继续透传；来源按 API Key 与客户端原始会话隔离。配额头和回合状态的强制透传同样位于 HTTP Adapter，原流读取器仍决定何时调用。

上游非流响应的有界读取由 infra/httpclient 执行，默认仍为 128 MiB，并保留多读一个字节判断超限及原错误链。gateway/httpapi 在原位置记录 Ops 并输出 Anthropic/OpenAI 形状的 502；读取函数不接管响应体关闭，重试与取消仍由执行链决定。

每次 attempt 都以原始/规范化请求和本次提供商重新构造供应商请求，注入凭据、代理、TLS 指纹、客户端标识、Thinking/工具配置及上游模型。平台适配器负责协议转换、上游响应限制和供应商错误解析，handler 负责在客户端协议中返回最终结果。

通用报文与转换算法位于 `protocol/{anthropic,openai,gemini,google,bridge}`。采样/Max effort 选项由 `gateway/forward` 的型号策略投影提供；平台适配继续选择 schema、thinking、签名与工具选项，并注入时刻和 ID 生成器；每请求/attempt 创建独立转换状态。Gemini 的 Messages 与 OpenAI 兼容流保留各自的 thinking、index 和 usage 观测顺序，以逐事件迭代返回输出；HTTP Adapter 提供同步写入与 Flush，upstream 执行器保持首次输出判定、取消、失败后排水和提供商内重试；外层网关仍决定换号。不得因提取纯状态机而整流缓冲或改变真实输出后的重试边界。

流式响应有不可逆边界：在调用上游前记录 `ResponseWriter` 已写字节数；如果 attempt 已向客户端写出真实业务输出，就不能再选择提供商，否则会把两个上游响应拼接为损坏的单流。旧版 Compact 桥接心跳、Responses 的 `response.created` / `response.in_progress` 前导事件，以及等待终态判定的可重试 `error` 帧不算业务输出，可以留在 attempt 缓冲中为 pre-output failover 保留空间；不可重试错误仍按事件边界及时转发。真实输出开始后，错误只能按当前协议追加允许的流错误事件或结束连接。非流式且尚未写响应时，才可以安全地进入下一次 failover。

错误分为本地准入、业务能力不足、调度容量不足、上游可切换错误和不可切换转发错误。协议准入拒绝分别使用 Anthropic `permission_error`、OpenAI `protocol_not_allowed` 和 Google `PERMISSION_DENIED`，且没有所选提供商。Ops 采集会记录归属、endpoint、实际提供商平台、模型和所选提供商；未选提供商时平台记为 unknown，但返回客户端的错误不能泄露凭据、内部代理或数据库错误。

Qoder 流式已经进入上游后使用完成释放：客户端断开停止下游输出，原预算内的尾部 usage 收集结束后再释放槽位。等待和非流请求仍按原取消策略释放。WS 入站连接与 Live 租约各自续租、丢失取消，不能因请求 Lease 引入而合并。

## 用量与结算

统一文本执行保留原生 usage 的缓存创建 5 分钟/1 小时分桶与 Speed，完成器据此选择原生计费语义，避免按 OpenAI 用量再次扣除缓存读取。用户售价与提供商成本分别计算；价卡缓存只按分组和模型索引，继承顺序仍是分组价、共享价、内置价。使用记录平台在写入时取实际提供商，不再从分组或入口推断。

上游转发产生可计量 usage 后，handler 把解析出的 token/图片/视频用量、客户端与上游模型、endpoint、提供商、订阅快照、请求标识和分组映射交给有界 UsageRecord worker pool。Anthropic 网关与 OpenAI 兼容的 Messages、Responses、Chat 三条链在终止事件前中断时，只要 service 随错误返回了部分结果，handler 仍提交其中已观测的 usage；无结果不生成记录，`UpstreamFailoverError` 不携带部分结果，避免重试成功后双重计费。

国产供应商原生 Anthropic 转 Responses 的流在客户端写失败后停止下游输出，但继续排水上游并推进状态机，直到读到末尾 `message_delta` 的最终 token 或达到有界读超时。OpenAI OAuth 图片响应在 HTTP 成功后若发生上游 body 传输中断，仅在尚未向客户端写出真实图片内容时按 502 进入提供商策略和 failover；JSON keepalive 空白不算真实输出，客户端取消、deadline、响应体超限以及首字节后的中断不会换号。

worker 使用脱离已结束请求取消信号但受自身超时约束的 Context；队列策略可以同步回退或丢弃，并通过指标/日志暴露压力，不能为每个请求创建无界 goroutine。

完成执行器的唯一实现位于 `gateway/completion`，配置由 app 投影。停止同时等待排队任务和已经接受的同步溢出任务；扩缩容与停止共用屏障，停止后不能重开。显式 drop/sample/sync 与 mandatory 兜底仍保留原入口语义，完成记录与结算编排也由 `gateway/completion.Recorder` 唯一实现。同步提交点使用 `gateway/provider.CaptureMessages/CaptureOpenAI/CaptureCyber` 读取原生提供商、Key、付款主体和用量，再生成独立的 `completion.Input`。

app 直接从原生价格、资金、用量和提交后端口构造 Forward/OpenAI 完成器，再将同一实例绑定到各执行入口。两条链各自保留独立倍率缓存，由 app 的原一分钟时间轮任务直接清理；后台副作用继续由唯一 ApplicationBackgroundTasks 拥有。媒体与 WS turn 同样在入队前取得快照，保留各自原计费时刻、请求 ID 和额度更新标记；creative 与 batchimage 独立拥有任务完成资格、预占/捕获/释放及恢复，不进入网关完成队列。

创作执行目标由 gateway/provider.CreativeTargets 按本次提供商构造，复用唯一凭据、传输和活动屏障；任务执行由 app 绑定具体生成能力。任务生成阶段通过 scheduler.Lease 管理用户与提供商槽，供应商返回后即释放，结果交付与结算不占槽。

结算的共同顺序为：

1. 归一化不同协议的 token 桶、媒体尺寸/时长、缓存和长上下文语义。
2. 根据计费模型来源、共享价格配置、提供商成本、用户/分组/订阅倍率、高峰倍率及共享价格配置分时倍率计算费用；WebSocket 多轮请求使用当前 turn 的开始时刻冻结时间相关价格。
3. 使用 `request_id + api_key_id` 认领结算幂等键，并用请求指纹检测 ID 被不同 payload 复用。
4. 在一个 PostgreSQL 事务中锁定付款用户，按结算模式分配订阅或余额，并同步累计团队成员、Key 配额/速率和适用的上游提供商额度；已通过准入并完成上游调用的普通请求若跨过指定订阅剩余额度，订阅用量封顶，溢出部分按余额倍率扣入付款主体余额并允许形成欠费。该回退只结算已放行或并发在途的请求，后续绑定已耗尽订阅的新请求仍直接拒绝；批量图片提交前的额度预占仍要求指定订阅完整覆盖。
5. 结算成功后尽力写已结算 Usage Log，并更新缓存和最后使用时间。结算失败时仍写入包含计算成本的待对账 Usage Log，但将 `actual_cost` 置零，随后返回 worker 错误且不伪造成功结算；Usage Log 写失败不得导致同一请求重复扣费。

请求热路径已经把响应交给客户端时，后台结算失败无法改写该响应，因此相关错误必须可观测并由幂等重试/对账流程处理。

<a id="moderation_search_boundaries"></a>
## 审核与搜索协作

网关通过 moderation.Check 取得本地裁决；请求解析只提取原有范围内的当前轮内容。实际行为用户与付款用户分开传入，供应商错误识别由 upstream 提供，审核模块不接管推理重试。创作台继续明确选择无媒体留存；普通审核和 Cyber warning 的事务保证见[内容审核](../domains/content_moderation.md#content_moderation_decision_pipeline)。

Brave/Tavily 搜索由 search 选择供应商并预占额度。失败释放自己已确认的预占；请求取消后不尝试其他供应商、不标记代理故障，额度回滚使用独立的最多三秒清理预算。Redis 结果不明确时继续原有故障放行，不猜测已取得计数。配置替换后已进入的请求保留原代次快照，所有代次在停机时共同等待。额度窗口与故障边界见[搜索编排](../domains/search_orchestration.md)。

gateway/searchtools 拥有工具识别、提供商/分组启用裁决、协议事件和合成 usage。app 通过 gateway/provider 直接绑定同一个 search.ConfigService、Registry 和分组策略读取实例，配置更换仍发布到该注册表。重试与完成处理继续由请求编排拥有。Grok 原生搜索及 OpenAI AlphaSearch 仍由对应 upstream 拥有；不能将通知成功、搜索配额或审核记录当作资金提交证明。

独立 Web/X 搜索由 app 直接构造 `gateway/httpapi.SearchHandler` 和固定的 `SearchPorts`。HTTP 继续按解析、认证、资金、审核、选号的顺序执行；平台报文及单次交换由 `gateway/provider` 组合原 Grok 原语与共享传输。选择端口只返回当前请求的受控目标，完成前同步取得提供商快照；异步记录不持有 Gin Context。相同查询的每次调用仍生成独立资金请求 ID，完成快照提交后再释放提供商资源。

## 扩展约束

新增网关入口或平台适配器时至少核对：

- 是否应用正确的 body/header 限制、request ID、Ops error logger 和 API Key 错误形状。
- 是否支持普通/复合 Key，模型从何处读取，哪些无模型管理入口只校验资源归属。
- 选组、Key 重定向、分组映射和提供商映射是否保持一跳顺序，模型列表与响应恢复是否同步。
- 用户/提供商并发、会话隔离、粘性和取消路径是否能完整释放槽位。
- 哪些错误允许同提供商重试或换提供商，流开始后是否会错误进入 failover。
- 用量能否得到稳定 request ID、请求指纹、requested/upstream model 和正确平台归属。
- HTTP 适配、业务用例、存储实现、前端调用方与 API contract/协议测试是否一起更新。

<a id="qoder_gateway_execution"></a>
## Qoder Chat 请求编排

Qoder 提供商参与普通文本入口的同一个提供商循环，`UnifiedTextExecutor` 复用 `QoderChatExecutor` 和兼容单次执行器。站点、模型及认证能力仍由 Qoder 适配器检查，已有单步协议转换不会扩展为任意会话恢复。

选号前保留资金预检，等待后复查资金不重复累计 RPM；已提交的真实输出禁止重放。Qoder 流式上游沿用独立执行预算，客户端取消不提前释放正在执行的提供商槽，完成捕获后再释放。部分结果只进入一次完成处理，不因记录或结算失败重新调用供应商。

平台尝试继续在 app 的 `QoderRequestsAndAttempts` 中登记，统一 HTTP 入口同时受网关请求屏障保护。停止时先禁止新请求并等待在途，随后才停止完成队列和共享连接；超过退出预算必须报告未完成。

<a id="upstream_attempt_ownership"></a>
## 平台执行与资源拥有权

各平台的供应商交换、请求构造与原生读取位于 upstream；gateway/provider 在调用点投影提供商、出站策略与错误观察接口，平台执行不接收 Gin 或完整 config。OpenAI 的响应读取、图片与辅助查询保留各自取消与终态差异，WS relay 和连接池独立于完整入站 WS 编排。续接报文和失效密文剥离使用平台纯实现。HTTP Responses 归属校验由 gateway/session 执行，HTTP 只保存已认证的 user/key 标识；同用户跨 Key 与历史仅 Key 归属的兼容规则保持，失败读取不授权。

app 构造唯一 OpenAI 会话状态存储供 HTTP/WS 共用，不另建归属缓存或提前打开连接；每轮价格快照仍由入站持有。缺失 usage 的低频诊断由 gateway/telemetry 持有原单份采样状态，日志不再读取 Gin 或完整提供商。

OpenAI 和 Messages 的转发结果由 `gateway/forward` 拥有，WS ingress hook 与 turn capture 由 `gateway/ws` 拥有。WS 重放输入继续保持私有，不随结果 JSON 输出；各协议结果仍保留原字段差异。每次 attempt／turn 的 `forward.ResponseObserver` 独立记录模型和实际服务档位，Gin 存取由 HTTP Adapter 负责；终态优先与冲突回退不改变，出站档位在完成计费时才与观测值结合。

Grok 文本、图片/视频和 Voice 共用固定装配的 GrokExecutor，沿用原请求取消策略及应用活动屏障。Chat→Responses 不适用时回到同一次派发的原生 Chat 路径；档位规则通过 ExecutionFastPolicy 在原位置读取设置和价格，WS 仍复用该 turn 的设置快照。运行时不新建提供商切换循环、资金记录器或缓存。

OpenAI 兼容文本的单次执行由 `OpenAITextExecutor` 组合现有协议执行器，目标、凭据头、TLS 和客户端策略通过 `OpenAIRequests` 按原顺序取得。HTTP 适配仍同步处理输出；Raw Chat、原生 Anthropic、Messages 和 passthrough 不各自创建提供商切换循环。CompatResponses 保留原提供商/Key/提示缓存隔离键、TTL 和续接禁用规则；Codex 额度观察复用原节流间隔，写回进入应用管理的后台任务。

上游风控警告统一用 `gateway/forward.UpstreamWarning` 传递，HTTP、WS 和 Grok 适配共用值类型及错误链契约。警告本身不证明请求可结算，完成资格、失败状态和通知仍按原入口规则决定。

Compact/SSE 注释心跳、非流式图片 JSON 空白心跳及扣除心跳字节后的输出判定由 gateway/httpapi 管理。图片首拍仍会提交 200，迟到错误继续以合法 JSON 写回；这些空白不会关闭原有安全重试窗口。TTFT 的语义/可见输出策略由 gateway/provider 组合平台事件解析，保持其与 HTTP 提交和重试窗口的区别。写入包装器只在 HTTP 内部使用，停止及写入互斥保持原语义。

OpenAI `cyber_policy` 的原生事件识别位于 gateway/provider，当前 HTTP/WS turn 的首个标记及清除由 gateway/httpapi 管理，直接使用 moderationflow.Mark。标记键、正文截断和已观测用量保持原值；清除后下一 turn 才能接受新证据。已透传错误使用 forward 的唯一哨兵通知收尾，不增加换号、响应写入或扣费资格。

GatewayRequestsAndAttempts 由 app 构造一次，HTTP 入口与原生平台尝试引用同一进入屏障；停止后拒绝新进入，并等待请求尾部完成快照入队及在途尝试，先于完成队列和共享存储关闭。它与 HTTPRequests 是并列等待屏障；额度恢复操作先取消，再等待请求结束，不能把 HTTPRequests 的完成时间当作停止监听时间。提供商授权会话及底层配额服务随后停止，按需 Live/WS 资源不因构造应用而提前开启。

相关文档：[系统架构](system_architecture.md)、[提供商调度与缓存一致性](provider_scheduling_and_cache.md)、[网关策略控制](../domains/gateway_policy_controls.md)、[上游提供商能力矩阵](../interfaces/upstream_provider_matrix.md)、[网关错误响应策略](../interfaces/gateway_error_policy.md)、[领域目录](../domains/index.md)、[接口目录](../interfaces/index.md)。
