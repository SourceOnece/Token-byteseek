# 前端 UI 规范

> 上级目录：[架构文档目录](index.md)

本文记录前端设计 token 与组件样式的强制约定：圆角层级、间距网格、控件尺寸、菜单与浮层、弹窗、层级、断点、加载反馈、图标与动画时长、行列表编辑器、表格密度、深色配色角色、图表主题和字号下限。覆盖 `frontend/tailwind.config.js`、`frontend/src/style.css` 与全部 Vue 组件；不覆盖浅色配色主题和业务组件的局部布局。修改前端组件、样式或这两个文件前先读本文。

本文的 TokenFlux 基线样式作为默认皮肤。ByteSeek 从 0.2.0-bh.002 提供包豪斯皮肤；两者共用组件、页面布局和业务状态，下文固定颜色/尺寸指默认 TokenFlux，包豪斯通过限定到根 data-visual-theme 的规则覆盖，不能全局覆盖默认皮肤。

## ByteSeek 视觉皮肤

- 管理员设置页使用自研 Select 保存站点皮肤 TokenFlux/包豪斯；公开设置与 HTML 注入统一下发，普通用户、访客和认证页不显示皮肤选择器。useVisualTheme 只接受站点配置，缺失/非法值默认 TokenFlux；该状态与 useTheme 的 light/dark/system 独立。站点更新只改变根属性及响应式图表色板，不重建路由或表单。
- style.css 跟随上游组件/动画配方；styles/visual-palette.css 给 Tailwind 工具类提供可切换颜色和阴影；styles/bauhaus.css 的皮肤规则限定于包豪斯根属性；styles/byteseek-components.css 给原有专属组件提供公共外观。图表 useChartTheme 和 Chart.js 默认值跟随皮肤恢复，不能无条件套包豪斯默认值。
- 包豪斯保留三原色、纸色、直角硬阴影、按压、三元素背景、满血绿/降智红/失败黄、标准价格绿/Fast 黄及红黄蓝前三名；深色上游去阴影规则由更精确的包豪斯规则覆盖。默认 TokenFlux 保持原生排版、颜色和动效。
- 用户仪表盘按上游使用四指标、用量趋势、Top 5、热力图、公告与快捷入口；没有额外旧八卡片与模型分布圆环，模型分布仍在使用记录及管理端。保留查询失败提示/重试，不把失败伪装为零用量。
- 定制业务保持票据工作台/账号规则/导入模板/勾选批改、质量检测与调度、邮箱列、管理员响应模型、代理、分组白名单、批量 Key/订阅/用户操作、订阅显示策略。导入默认值的模型和映射继续双列，使用上游稳定行编辑器；票据随原保存动作提交。
- BaseDialog 保留 showCloseButton，供批量任务锁定时隐藏关闭入口；上游 MotionTransition、嵌套 Esc、滚动锁、退出 inert、下拉定位和减少动画共同生效。已有分页数字输入、查询竞态、注册确认密码、导出条件快照和退款金额判断修复不得随整页迁移丢失。

## 章节导航

- [圆角层级](#圆角层级)、[间距约定](#间距约定)、[控件尺寸](#控件尺寸)、[开关](#开关)：调整基础组件时读取。
- [菜单与浮层](#菜单与浮层)、[层级 z-index](#层级-z-index)、[弹窗](#弹窗)：调整浮层及遮罩时读取。
- [通用图标](#通用图标)：选择图标、调整悬停动画、迁移内联 SVG 时读取。
- [断点](#断点)、[加载反馈](#loading_feedback)、[动画与时长](#动画与时长)、[表格密度](#表格密度)：调整响应式布局和交互时读取。
- [行列表编辑器](#rule_list_editor)：新增或修改逐条添加的映射、规则列表时读取。
- [深色配色](#dark_colors)、[图表主题](#图表主题)、[字号](#字号)：调整颜色和文字时读取。
- [合法例外](#合法例外)、[校验](#校验)：确认局部例外及验证入口时读取。

## 圆角层级

圆角只允许使用语义 token，数值的唯一来源是 `style.css` `:root` 的 `--radius-*` 变量，`tailwind.config.js` 的 `borderRadius` 只做 var() 引用：

| token | 值 | 用途 |
|---|---|---|
| `rounded-compact` | 6px | 徽章、chip、tab 项、骨架屏、行内代码、小图标块 |
| `rounded-control` | 8px | 按钮、输入框、下拉框、侧栏链接、浮层面板 |
| `rounded-surface` | 12px | 卡片、表格容器、toast、代码块 |
| `rounded-dialog` | 16px | 桌面端弹窗（移动端弹窗仍用 surface） |
| `rounded-full` / `rounded-none` | — | 胶囊、进度条、开关；需要直角时的覆盖 |

系统设置页的吸顶导航采用胶囊分段样式：外壳、一级标签和网关二级标签都用 `rounded-full`。选中态只保留淡品牌青底、边框和图标文字着色，不再叠加底部渐变线；网关页多出二级标签行时，外壳改用 `rounded-dialog`。

旧尺度名（`rounded-sm/md/lg/xl/2xl/3xl`）、裸 `rounded` 和 `rounded-[...]` 任意值一律禁用——旧 key 已从配置删除，写旧类名不会生成任何样式。裸 CSS 里的 `border-radius` 只允许 `var(--radius-*)`、`0` 或 `9999px`。

唯一例外：边长 ≤16px 的微型装饰元素（如用量热力图的 12px 格子），全局最小档 compact（6px）已达边长一半、视觉上近似椭圆，允许用组件级局部变量保持更小半径（如 `.heatmap-cell` 的 `--radius-cell: 4px`），不新增全局档位。

<a id="layout_spacing"></a>
## 间距约定

- 全部间距落在 Tailwind 4px 网格上，禁止 `mt-[2px]`、`padding-left: 17px` 这类任意值。
- 页面布局使用两档间距：独立大卡片、统计卡、图表卡和移动端数据卡之间，以及工具栏、控件组与卡片之间为 16px（`gap-4` / `space-y-4`）；工具栏和控件组内部仍为 8px（`gap-2` / `space-y-2`）。横向网格与纵向堆叠遵守同一档位，加载骨架与实际内容一致。`TablePageLayout` 中相邻工具组保持 8px，最后一组工具到表格使用 16px；移动端数据卡与分页器之间也是 16px。`RuleListEditor` 的标题操作区到卡片列表为 16px，卡片行之间为 16px，线形行及其标题操作区之间保持 8px。
- 列表的搜索、筛选、刷新和批量操作工具栏统一使用 `gap-2`（8px），覆盖工具栏外层、左侧筛选组和右侧操作组；共用组件与加载骨架遵守同一间距。窄屏换行仍使用 8px 行间距。筛选弹层内带标签的字段按表单间距排列，分页摘要、正文信息组和卡片区块保留各自间距。
- 页面或卡片的标题带说明文字时，同一行的操作区与整块标题说明底部对齐。`AppLayout` 页头使用 `items-end`，自绘页头和带说明的卡片标题行遵守同一约定。页头补充信息放在可选的 `page-heading-meta` 插槽，位于说明下方 8px，参与同一底部对齐，例如用户仪表盘的实时状态条；先纵向堆叠、宽屏再横排的布局只在横排断点启用底部对齐，窄屏控件仍正常换行。
- 卡片 padding 只有两档：独立卡片 `p-6`，嵌套面板、网格卡和统计卡 `p-4`。不再使用 `p-5`。
- 布局水平 padding 链在 header 与 main 之间完全一致：`px-4 md:px-6 lg:px-8`，保证两侧边缘在所有断点对齐。
- 默认首页和控制台共用 `AppHeader`，统一品牌、工具按钮、余额和用户菜单。首页通过 `public-page` 隐藏侧栏开关，保留模型广场与访客登录入口，并为固定顶栏预留高度；自定义 HTML 或 iframe 首页继续使用全页模式。操作台的返回仪表盘图标位于品牌右侧。
- 布局尺寸 token 只在 `style.css` 的 `:root` 定义一份：`--header-h`（3.5rem，顶栏高度）、`--sidebar-w`（14rem，侧栏展开宽）、`--sidebar-w-collapsed`（4.5rem，侧栏折叠宽）。顶栏高度、主区 `padding-top`、侧栏遮罩 `top`、侧栏宽度与主区 `lg:ml-*` 偏移一律引用变量（如 `h-[var(--header-h)]`），不写 `h-14`、`top-14`、`w-56` 这类平行字面量。吸顶偏移与锚点 `scroll-margin-top` 同样以 `calc(var(--header-h) + 余量)` 组合（参考 SettingsView 的 tabs 吸顶），余量写构成注释。
- 垂直空间由 AppLayout 的 flex 链统一分配：wrapper（`flex-col`，普通模式 `min-h-screen` / 锁定模式 `h-full min-h-0`）→ `.app-main`（`flex-1 flex-col`）→ 页头（自然高度）+ 页面内容。需要撑满剩余高度的页面容器（如 `TablePageLayout`、`CustomPageView` 根元素）自取 `flex-1 min-h-0`，禁止手写 `calc(100vh - …)` 视口差值、禁止负 margin 抵消父级内边距；不引入 `--main-pad-*`、`--page-heading-space` 这类与布局重复的尺寸变量。
- 系统设置页的内容容器使用 `w-full min-w-0` 填满主区，各页签的卡片保持同宽；不设置居中外边距或最大宽度，避免在纵向 flex 布局中按内容收缩。
- 全屏工作区（`full-viewport`）模式下 `.app-main` 无内边距，页面天然满幅。宽屏锁定（`fit-viewport`）模式只在 `lg` 及以上把外壳锁定为视口高度，保留页头和标准内边距链，页面根容器以 `lg:flex-1 lg:min-h-0` 承接剩余高度，由卡片内部区域滚动（参考兑换页的历史列表与分页器）；窄屏仍随内容自然滚动。
- 自定义页面使用 `fit-viewport="all"`，在所有屏幕尺寸下按动态视口分配高度，保留页头和标准内边距。iframe 与 Markdown 正文在卡片内滚动。只设 `min-h-screen` 不能为后代的 `height: 100%` 提供明确高度，会使 iframe 回落到默认的 150px；需要百分比高度的页面必须接入完整的高度链。
- 表单内 `space-y-2/3/4/6` 按上下文自选，不归一。

## 控件尺寸

- 按钮、输入框、下拉触发器共用 36px 基线（`.btn` / `.input` 均为 `min-h-9`），分页器控件同为 36px——表格页脚不再压缩分页尺寸。基线之上再写 `h-9` 属冗余（门禁拦截）；紧凑档要 36px 时用 `btn-sm/md/lg + h-9` 显式提挡。
- 主操作使用 `.btn-primary`；普通编辑、查询、筛选和链接操作使用 `primary-*` 品牌色，不单独指定 `blue-*`。状态提示、业务分类与第三方品牌保留各自的语义配色。radio 和 range 通过全局 `accent-color` 使用 `primary-600`。checkbox 保留原生 input 的勾选、半选、禁用和键盘语义，由 `style.css` 统一外观；浅色填充为 `primary-700`，深色为 `primary-600`，白色勾选标记复用通用 check 路径与 1.75 描边。选中、取消采用 `--motion-fast` 的缩放和透明度过渡，半选显示横线；系统强制配色时回到原生外观。
- 原生按钮、`role="button"` 和 `.btn` 禁止文字选取，按钮内图片禁止浏览器拖拽；正文、表格数据和输入内容仍可选取复制。
- 图标按钮两档：`.btn-icon`（h-9 w-9）与 `.btn-icon-sm`（h-8 w-8），自带 `rounded-control` 与居中布局，站点只补 hover/颜色类；`.btn-sm` 用于表格行内等紧凑场景。
- 下拉触发器（Select、DateRangePicker）模板组合 `input input-trigger` + 各自状态类，不复制基线配方。
- 分段切换（两到五个互斥选项，如指标、时间范围、数据来源）统一用 `style.css` 的 `.segmented` 轨道、`.segmented-item` 选项和 `.segmented-item-active` 选中态，不再手写灰底白块。轨道加 `v-segmented`（`directives/segmented.ts`），共用一个选中背景；内边距、字号和高度由调用方补工具类；放进 36px 工具栏时给轨道加 `h-9 items-stretch`。选中项必须保留 1px 描边，浅色下只靠阴影和白底分不清边界。页面级大页签仍用 `.tabs`。
- 输入框图标/字符前后缀统一走 `input-icon-*` 机制（`style.css`）：容器 `input-icon-wrap`，图标位 `input-icon` / `input-icon-right`（可点击内容加 `input-icon-action`），输入框按侧加 `input-has-icon` / `input-has-icon-right`；文本留白由变量推导（`留白 = inset + slot`）。档位：默认（inset 0.75rem、留白 2.5rem）、`input-icon-lg`（auth 表单，inset 0.875rem、留白 2.75rem）、`input-icon-text`（`$` 等窄字符前缀，留白 2rem），紧凑搜索框内联 `--input-icon-slot:1.5rem`（留白 2.25rem）。
- 价格管理和属性管理的页签栏与下方工具栏之间使用 16px 间距。搜索框使用同一图标布局，`sm` 及以上固定为 `w-64`，更窄时随工具栏剩余宽度伸缩，提示写明配置名称或模型名称。工具栏相邻控件统一使用 `gap-2`；配置页的状态筛选框使用 `w-32 shrink-0`，默认目录页的两个筛选条件收纳到 `FilterDropdown`（规则见[菜单与浮层](#菜单与浮层)）。两页的默认目录信息共用 `ModelCatalogInfo`，以辅助字号展示来源、短版本号和更新时间，窄屏自动换行。

## 开关

全站开关只有 `components/common/Toggle.vue` 一个实现，禁止手写轨道/滑块（门禁拦截 `h-6 w-11` / `h-5 w-9` 组合）。

- 几何单一来源是 Toggle  scoped 样式里的 CSS 变量（`--toggle-track-w/h`、`--toggle-thumb`、`--toggle-inset`），开态位移由 `calc(轨道宽 − 滑块 − 2×边距)` 推导，改档位只调变量。
- 档位：`size="md"`（44×24）/ `size="sm"`（36×20），`variant="inset"`（滑块内嵌，默认）/ `variant="flush"`（大滑块贴边，原 Headless 手写风）。
- 配色：开态默认 `toggle-active`（≡ `bg-primary-600`）；关态由 `off-tone` 选档——`default`（gray-300）/ `soft`（gray-200，手写迁移站点的原色）。个别站点的亮色开态（`bg-primary-500`）或 hover 配色用 `on-class` / `off-class` 整串透传，不新增档位。
- 异步保存场景用 `:model-value` + `@update:model-value` 受控写法，值由处理器写回（参考 ProvidersView 的可调度开关）。

## 菜单与浮层

- 菜单和搜索建议列表统一用 `.dropdown` 容器配方，复用圆角、边框、阴影和深色背景。默认绝对定位、上下留白 `py-1`；需要固定定位时补 `fixed`，内容自带留白时补 `py-0`。定位偏移、尺寸和箭头由调用点维护，展开和收起使用公共动效配方。
- 深底信息提示（`HelpTooltip`、表格悬停明细、状态说明等）统一用 `.tooltip-panel` 面板配方：浅色为 `gray-900` 深底白字，深色使用 `dark-900` 底色、`dark-100` 文字和 `dark-600` 描边。箭头用 `.tooltip-caret` 旋转方块，半嵌入面板边缘；调用点补定位和朝外的两条边（向下 `border-b border-r`、向上 `border-l border-t`、向左 `border-b border-l`、向右 `border-r border-t`）。不再手写 `dark:bg-gray-*` 面板或边框三角箭头，否则深色配色调整时会漏改。浮层内的分隔线深色用 `dark-600`。
- 菜单项两档：`.dropdown-item`（px-4）与紧凑档 `.dropdown-item-sm`（px-3），配色、hover、过渡都在配方里，站点只补 `gap-*`、`rounded-control` 这类布局增量。配方基线是中性色。顶栏用户菜单、联系客服这类导航型浮层用内缩菜单配方：面板 `.dropdown py-0`，内容按 `.menu-section` 分区（`p-1.5` 留白，相邻分区自动加分隔线），分区标题 `.menu-heading`，菜单项 `.menu-item` 内缩并带 `rounded-control` 悬停底色，悬停配色与 `.sidebar-link` 一致，图标继承文字色；危险操作叠加 `.menu-item-danger`。顶栏工具区图标按钮统一用全局 `.header-status-icon-button`。
- 列表工具栏的漏斗筛选只用 `FilterDropdown`，字段用 `FilterField` 包裹，不再手写按钮、面板和点击外部关闭逻辑。面板外观与 `DateRangePicker` 同源：`rounded-surface`、淡描边和柔和阴影；头部、已选条件和字段区之间不画分割线，只靠留白分层。头部是标题、条件数和「重置」，没有生效条件时「重置」置灰；有生效条件时，头部下方列出已选条件标签（字段名 + 当前取值 + ×），顺序与字段一致，点 × 只移除这一项；再往下是可滚动的字段栅格。`FilterField` 里的 `Select` 会自动上报当前选项：第一项取值为空或 `'all'` 时当作「全部」，其余情况按空值判断，默认值不是第一项时（如运维看板时间范围的 `1h`）传 `empty-value`。文本输入、远程搜索这类非 `Select` 字段，由调用方传 `value-text` 并监听 `clear`。字段标签用中性灰 `text-xs`，不用表单的 `.input-label`；占满整行的字段传 `full`。`columns` 取 1/2/3，对应面板宽 18/34/48rem，窄屏都退回单列：两三个条件用 1，四个上下用 2，更多用 3。面板默认左缘对齐触发按钮，右侧放不下时向左平移，水平夹取复用 `getFloatingPanelPosition`。有生效条件时按钮换品牌色描边并显示数量角标。面板里有输入框状态，或测试需要直接访问字段时传 `keep-mounted`；运维看板这类自定义按钮样式用 `trigger-class` 覆盖。
- 日期范围只用 `DateRangePicker`：左侧是分组的快捷范围，右侧是自绘的单月日历，不再用原生 `type="date"` 输入。点两次日历确定起止日期，反向点选时自动对调；起止日期和主按钮同色，中间的日期用淡品牌青色带连起来，今天用小圆点标出。最晚可选到明天，用来兼容时区差异。没点应用就取消、点外部或按 Esc 关闭时，改动全部丢弃，触发器只显示已生效的范围。弹层由 `getFloatingPanelPosition` 定位：触发器在视口右半侧时右对齐，在左半侧时左对齐。
- 表格行内操作菜单（4 个 `*ActionMenu`）的浮层容器统一 `.action-menu` 类（fixed 定位 + 层级 + 面板样式），宽度类（w-48/w-52）与 `action-menu-content` 钩子类留在调用点。
- 遮罩透明度只有两档，唯一来源是 CSS 变量：浅色在 `:root` 定义常规 `--overlay-bg`(black/50)、媒体灯箱等强遮罩 `--overlay-bg-strong`(black/70)。深色模式在 `html.dark` 中整体加深为 black/70 和 black/85。模板写 `bg-[var(--overlay-bg)]`，禁止手写 `bg-black/50` 这类字面值——原 /55、/60 漂移值已就近归并入标准档。
- 浮层面板最大高度三档，唯一来源是 `:root` 的 `--max-h-menu-sm`(15rem)、`--max-h-menu`(20rem)、`--max-h-panel`(26.25rem)，模板对应 `max-h-menu-sm/menu/panel`；像素任意值 `max-h-[Npx]` 由门禁拦截，局部特例（如告警表 520px）加 `check-ui-allow` 说明。vh/dvh/calc 等视口相对值语义不同，不入档也不拦截。
- 挂载到 body 的浮层定位只有一份实现：`utils/floatingPanel.ts` 的 `getFloatingPanelPosition`（翻转、对齐、夹取、窄屏行为全部由 options 表达：固定高菜单用 `fixedHeight`，左对齐面板用 `align: 'left'`，菜单类传 `pinLeftOnMobile: false`)。禁止在组件里重写 rect/spaceBelow 翻转几何。JS 侧面板尺寸常量在 `constants/overlay.ts`(`SELECT_PANEL_MAX_HEIGHT`、`MIN_COMFORTABLE_PANEL_HEIGHT`)，与样式档位同源。

## 断点

- 断点数值的唯一来源是 `constants/layout.ts`:`BREAKPOINT_SM/MD/LG`(640/768/1024，与 Tailwind 默认 screens 对齐，tailwind.config.js 不得自定义 screens)、`MEDIA_MIN_*`/`MEDIA_MAX_*` 媒体查询串、`TABLE_DESKTOP_MEDIA_QUERY`(lg 别名，表格与分页器共用切换点）。契约测试 `breakpointTheme.spec.ts` 锁定对齐。
- max 变体约定 `max = min - 1px`(与 Tailwind max-* 语义一致）,min/max 区间互斥、没有 1px 重叠带；JS 侧写 `MEDIA_MAX_SM` 而不是 `<= 640`。
- JS 断点判断（useMediaQuery/matchMedia/innerWidth 比较）一律引用常量，字面量由门禁拦截；CSS @media 不支持 var()，唯一例外落点（如 CustomPageView 目录抽屉的 639px）写字面量 + 推导注释，由契约测试锁定。
- Ops 五个表格组件曾用 768 并自称「与 DataTable 一致」，实际 DataTable 是 1024——已统一修正为 `TABLE_DESKTOP_MEDIA_QUERY`，修掉 768–1023px 区间桌面表格配移动分页器的混排。

## 层级 z-index

层级梯队三轨同源：`style.css` `:root` 的 `--z-*` 变量承载唯一数值 → `tailwind.config.js` 的 `zIndex` 扩展只做 var() 引用（模板用 `z-modal`、`z-toast` 这类语义工具类）→ `constants/overlay.ts` 的 `Z_INDEX` 常量供 JS 内联。契约测试 `src/__tests__/zIndexTheme.spec.ts` 锁定三轨一致；任意值 `z-[...]` 与数字字面量由门禁拦截。梯队（只命名不调层级，数值即现状）:

| 值 | token | 用途 |
|---|---|---|
| 30 | `chart-tooltip` / `sidebar-overlay` | 图表 tooltip（有意低于导航）/ 侧栏遮罩 |
| 40 | `sidebar` | 侧栏 |
| 50 | `header` / `modal` | 顶栏 / 弹窗遮罩（同层靠 DOM 顺序与 teleport 决胜） |
| 60 | `modal-nested` | 嵌套弹窗、筛选面板 |
| 100 | `tooltip` / `announcement` | 弹窗内 tooltip、下拉面板、灯箱 / 公告底层 |
| 120 / 140 | `announcement-raised` / `announcement-top` | 公告梯队（递增有意） |
| 9998 | `menu-overlay` | ActionMenu 点击捕获层 |
| 9999 | `toast` / `action-menu` / `teleport-tooltip` | 通知 / 行内操作菜单 / teleported 提示浮层 |
| 99999 | `help-tooltip` | HelpTooltip(teleported) |
| 100000000 | `tour` | driver.js 引导层，外部约束，仅登记不暴露工具类；onboarding.css 保持字面值 !important |
| 100000020 | `teleport-dropdown` | Select 等 teleported 下拉（必须压过引导层） |

局部堆叠上下文不入阶梯、不加全局 token:DataTable 内部（0/20/200/210/220)、CreativeCanvas 画布内、CustomPageView 目录抽屉、弹窗内 sticky 表头（z-[1])，均加 `check-ui-allow` 豁免。表内局部 dropdown 用普通 `z-50` 即可（局部上下文，不占语义档）。

## 弹窗

- 默认入口是 `BaseDialog`：宽度档位 narrow/normal/wide/extra-wide/full，Escape 关闭、点击外部关闭、焦点管理与背景滚动锁定全部内置，新弹窗不要再手写 `fixed inset-0` 外壳。
- 分页表单可设置 `BaseDialog` 的 `bodyScroll=false`，由表单内部管理滚动；标题、页签和底部操作区保持可见。默认仍由弹窗内容区滚动。分组创建与编辑共用 `GroupSettingsForm`，分类和控件布局在共享表单内维护。
- 安全凭证流程（TOTP 设置/禁用/登录验证/提权）走 `AuthCardDialog`：居中图标头、无右上角关闭按钮、整卡 p-6，是与 BaseDialog 并存的独立风格族。它不 teleport、保持内联渲染，嵌套层级由 `z-index` prop 决胜。
- 分诊标准：结构同构（标题头 + 内容 + 按钮行）的手写弹窗迁 BaseDialog；有定制视觉结构的保留并登记在下面的例外清单。

<a id="dark_colors"></a>
## 深色配色

深色文字和表面色阶的唯一来源是 `tailwind.config.js` 的 `colors.dark`，边框另用同文件的 `darkEdges`。按组件角色选择 token，保留现有 `dark-*` 档位名称：

| 档位 | 值 | 角色 |
|---|---|---|
| `dark-50` | `#FAFAFA` | 标题与强调文字 |
| `dark-100` | `#DEE0E2` | 正文文字、侧栏导航与顶栏图标默认色 |
| `dark-200` | `#D4D4D8` | 次强文字、占位文字底色 |
| `dark-300` | `#A1A1AA` | 次要文字 |
| `dark-400` | `#8B8B94` | 辅助文字、表头文字 |
| `dark-500` | `#5F5F67` | 图标、禁用文字 |
| `dark-600` | `#3D3D42` | 较强中性填充 |
| `dark-700` | `#27272A` | 中性填充：chip、禁用控件 |
| `dark-800` | `#17171A` | 弱填充：表格行 hover、嵌套面板 |
| `dark-900` | `#0F0F10` | 卡片、弹窗、下拉面板 |
| `dark-950` | `#141416` | 控件底：输入框、次级按钮、Tab 轨道、行内代码 |

页面底色统一引用 `--page-bg`：浅色为 `#FCFCFE`，深色在 `html.dark` 覆盖为 `#0A0A0B`。`html`、`body`、主题外壳与背景层共用纯色背景，深色页面滚动条轨道也引用该变量，不使用页面背景渐变。

深色侧栏和顶栏同样使用不透明的 `--page-bg`，与页面保持同色。控制台、首页和公开模型广场的顶栏共用 `.site-header`；深色模式关闭顶栏背景模糊，滚动内容和移动侧栏遮罩不会改变其底色。浅色顶栏保留白色 80% 透明度与背景模糊，浅色侧栏仍为白色。

深色分层使用接近底色的表面和低透明度边线。`borderColor.dark`、`divideColor.dark`、`ringColor.dark` 共用 `darkEdges`，不从 `colors.dark` 取实色，避免边框调整牵动文字和背景。

| 边线档位 | 灰白 `#FCFCFE` 的透明度 | 用途 |
|---|---|---|
| `dark-400` | 30% | 控件焦点 |
| `dark-500` | 12.5% | 控件 hover、较强边线 |
| `dark-600` | 7.8% | 卡片、输入框、弹层的默认边框 |
| `dark-700` | 4% | 表内分隔线 |
| `dark-800` | 3.1% | 顶栏、侧栏与页面的分割线，其他弱分隔线 |
| `dark-900` | 2% | 最弱边线 |

边线强度与 `/70` 等透明度修饰符相乘，例如 `dark:border-dark-600/70` 的最终透明度为 5.46%，不能覆盖成 70%。裸 CSS 用 `theme('borderColor.dark.600')` 等引用同一来源。深色中性边框统一使用 `dark-*` 边线档位，避免残留 `gray` / `slate` 实色描边。

顶栏底边和侧栏外侧边共用 `dark-800` 弱分割线，由 `.site-header` / `.sidebar` 的深色配方维护。控件与卡片保留 `dark-600` 默认边框，使可操作区域的边界比页面分区更清楚。

表头与卡片同为 `dark-900`，表格行 hover 用 `dark-800`。控件底 `dark-950` 略亮于卡片。

导航、分段控件和列表选中行使用 `primary-500/8` 淡品牌青底与 `primary-500` 文字，带边框的选中控件使用 `primary-500/15`。侧栏和 Select 的未选中项 hover 使用 `dark-800` 弱填充与品牌青文字，已选项 hover 或获得键盘焦点时保留选中底色。主操作、链接、开关和图表继续使用品牌青；状态色和徽章配色保持各自语义。

输入框焦点、次级按钮焦点与 Select 展开态使用 `dark-400` 边线（灰白 30%）和 `white/6` 外圈。按钮的深色焦点环 offset 使用 `dark-900`。深色遮罩由 `html.dark` 将 `--overlay-bg` / `--overlay-bg-strong` 加深为 0.7 / 0.85。

## 图表主题

- 图表主题的唯一入口是 `composables/useChartTheme.ts`：响应式 `colors`（text/muted/grid 三档语义，zinc 体系）+ `onThemeChange` 重绘钩子。禁止 `document.documentElement.classList.contains('dark')` 快照判断（门禁拦截）——它没有响应式依赖，切主题不重算，曾导致 8 处图表切主题不换色。vue-chartjs 场景 colors 变响应式即自动重绘；Stripe Elements 等命令式场景用 watch + `elements.update({ appearance })` 重应用。
- 分布图调色板只有一份 `CHART_PALETTE`（12 色，按切片排名取色），"Others" 聚合切片用 `CHART_OTHER_COLOR`;token 趋势序列色用 `CHART_SERIES_COLORS`。刻度字号 `CHART_TICK_FONT_SIZE`(10)、图例字号 `CHART_LEGEND_FONT_SIZE`(11)。
- 业务色例外留在本地：TeamMemberUsageCharts 成员固定配色（跨图表按成员稳定取色）、OpsSwitchRateTrendChart 与 DashboardView 的本地图表主题（深色刻度 `#D4D4D8`、网格 `#27272A`，浅色保留品牌调字面值）、DailyRevenueChart 的线/填充色对。
- 用户仪表盘趋势图（`UserDashboardUsageChart`）只画线，不填充线下区域，也不加辉光阴影，保持扁平风格。单指标叠加上一周期对比线，使用 `colors.muted` 虚线，按下标与本期对齐。峰值环、均值虚线、标签胶囊和末端呼吸点画在同一个本地插件里，颜色取自 `useChartTheme` 与卡片底色。未结束的时段仍画虚线。
- token 数量格式化统一 `utils/format.ts` 的 `formatTokens`（两位小数 + 千分位）与 `formatTokensK`（一位小数），语义不同不混用；ProviderTodayStatsCell 的 K1/M2 混合精度是有意的本地变体。

<a id="loading_feedback"></a>
## 加载反馈

路由切换统一使用 `NavigationProgress`：页面顶部 2px 品牌青细线，`pointer-events: none`，不遮挡点击。`router/navigationLoading.ts` 在鉴权守卫之前注册导航反馈，覆盖异步页面加载、重定向、取消和异常。每次导航都有编号，旧导航或旧完成动画不能结束新导航。

导航开始立即显示；成功、取消或异常后，进度线铺满并淡出，由 `animationend` 通知状态层隐藏。这里只表示导航仍在进行，不报告请求完成百分比。减少动画模式下保持静态细线，并缩短结束动画。页面内的数据请求和轮询不接入全局导航指示器，由各自的数据区域反馈加载状态。

未取得数据时，在内容将要出现的位置显示骨架，保留页头、筛选工具栏、卡片外框和表格列结构。不要在加载期间展示业务零值或“暂无数据”。通用骨架使用 `Skeleton.vue` / `.skeleton` 的中性色与轻微脉动，减少动画模式下关闭脉动；装饰块用 `aria-hidden`，区域用加载标签和 `aria-busy` 表明状态。

当前入口包括管理员仪表盘的 `DashboardSkeleton`、设置页的 `SettingsSkeleton`、模型广场的 `ModelMarketplaceSkeleton`、图表的 `ChartSkeleton`，以及 DataTable 的表格行和移动卡片。公告按时间线条目占位，热力图直接用日期格子占位。用户仪表盘的用量指标卡、趋势图和模型排行只在首次取数时显示骨架，分别占位数值、绘图区和 5 行排行；刷新、切换范围或筛选时保留已有数据。加载成功或失败后退出占位状态，沿用页面原有的数据、错误或空状态分支。标题下方的实时状态条首次取数时用行内骨架占位数值；之后轮询失败只把数值显示为“—”，不弹出错误。

页面、弹窗和局部数据区共用这套规则。列表、表单、字段详情和文档正文可使用 `ContentSkeleton`；统计页按实际卡片网格组合 `Skeleton` 与 `ChartSkeleton`。原生表格使用 `TableSkeletonBody`，保留真实表头并传入当前可见列数，条件列变化时同步调整占位列。加载更多只在列表末尾追加占位，已有内容保留。

骨架表示内容尚未取得。提交、刷新按钮、授权跳转、连接测试、任务执行和支付处理中继续使用操作反馈。支付页面首次读取订单和初始化表单时显示骨架；渠道已受理或等待外部支付时保留状态提示。加载样式不得改变请求、轮询、支付 SDK 或 iframe 的挂载时机。

## 通用图标

通用界面图标统一使用 `components/icons/Icon.vue`，名称与 `IconName` 类型由同目录的 `registry.ts` 管理。图形采用 Lucide 风格，官方逐元素动效移植自 Lucide Animated，以 `motion-v` 运行；缺少官方动画的图形使用一次 400ms 的轻微缩放。源码版本和许可见图标目录的 README 与 LICENSE 文件。

- 保留 `name`、`size`、`strokeWidth` 接口；默认描边为 1.75，颜色继承 `currentColor`。根节点只有一个 SVG，调用点的样式、事件、标签和显式尺寸透传。
- 尺寸档位为 `xs` 12px、`sm` 16px、`md` 18px、`lg` 24px、`xl` 32px。侧栏一级导航和顶栏工具区使用 `md`（18px）；普通按钮、纯图标操作按钮、分页、选择框箭头、弹窗关闭按钮及侧栏子项统一使用 `sm`（16px），刷新、创建、编辑等操作保持同尺寸。表格内微型操作可保留 `xs`，快捷入口卡片的主图和媒体灯箱关闭图标保留 `lg`。侧栏自定义 SVG 与一级导航同尺寸。图标尺寸不改变按钮的点击区域。
- 英文导航使用简短名称，省去上下文已说明的 Management、Records 等词；页面标题和说明仍可保留完整名称。
- `animateOnHover` 默认开启。动画绑定最近的按钮、链接、菜单项等控件，非标准交互容器加 `data-icon-trigger`；独立图标响应自身悬停。键盘聚焦使用同一反馈，每次进入只播放一次，鼠标和焦点都离开后复位，初次挂载不自动播放。
- 演示时间轴通过 `animationActive` 触发图标已有的动画序列，变为 `true` 时播放一次，变回 `false` 时复位；该入口独立于 `animateOnHover`。Key 重定向演示使用节点的 `animationstart` 事件触发，重播时重置，继续遵守减少动态效果、禁用和卸载清理规则。
- `disabled`、`aria-disabled`、父级 `inert`、加载转圈和系统减少动态效果均抑制装饰动画，状态变化立即生效。动画序列在离开、换图形或卸载时取消，禁止用定时器猜测完成时间。
- 加载转圈继续由业务状态控制，`animate-spin` 和 `.spinner` 在减少动态效果模式下静止。展开、排序和选中指示图标设置 `:animate-on-hover="false"`，外层 CSS 旋转仍表达原有状态。
- 图标默认 `aria-hidden`，不新增焦点。根 SVG 默认设置 `tabindex="-1"` 与 `focusable="false"`，内部图形在挂载和更新后补齐相同属性，避免焦点监听让浏览器将装饰节点加入 Tab 顺序。鼠标仍可能聚焦这些 SVG 节点，`useIconAnimation` 会将焦点交回外层控件或 label 关联的表单控件，避免装饰图形出现浏览器默认方框；键盘焦点提示继续由外层控件提供。具备独立语义时传入无障碍标签；图标按钮仍由按钮提供名称和点击区域。
- 通用图标与模型品牌图标使用 `select-none`，包括模型图标的字母占位，避免拖选文字时出现图标选中高亮；保留图标的点击和悬停事件。
- 新增通用图标必须进入统一映射。品牌标志、用户上传的 SVG、图表和业务插画保留专用实现，自定义 SVG 继续经过既有净化流程。

<a id="ui_motion"></a>
## 动画与时长

普通交互共用 `style.css` 的动效变量，Tailwind 的 `duration-fast/normal/layout` 只引用变量。按钮、提示和内容淡入使用 `--motion-fast`（150ms），菜单、开关与短列表使用 `--motion-normal`（200ms），折叠、侧栏及弹窗进入使用 `--motion-layout`（220ms）；浮层退出共用 `--motion-exit`（150ms）。缓动为 `--motion-ease`，浮层退出使用 `--motion-ease-exit`，菜单位移为 `--motion-shift`（4px）。不在调用方复制普通动效时长。

- Vue 在运行时添加过渡类，这些配方写在 Tailwind `@layer` 外，避免静态类名扫描将它们裁剪。全局显隐配方包括 `fade` / `fade-slow`（纯透明度）、`dropdown-fade` / `pop-float`（透明度与短距离位移）、`modal` / `pop-fade`（遮罩与面板，面板从 0.98 缩放进入）。`fade-slow` 保留已有调用名，使用内容淡入档。向上打开的菜单用 `--dropdown-shift` 调整方向。定位使用的 top、left、bottom 不参与浮层过渡；带定位 transform 的提示只使用 fade。调用方不保留 scoped 副本。
- 显隐入口使用 `MotionTransition`，它将 Vue Transition 的属性和事件原样透传，并在退出时设置 inert 防止旧节点响应操作；只有当前校验不通过的表单控件会被暂时禁用，以免阻止原生校验，其余控件保持原样，退出动画里不会闪出禁用态底色。`v-show` 调用点显式传 `persisted`。菜单的点击捕获层随关闭立即移除，面板保留到退出结束。Select、HelpTooltip 等通过 `useFloatingMotion` 跟随正在折叠的触发器，祖先变为 inert 时同步关闭 Teleport 浮层。
- 纵向展开使用 `Collapse`：`open` 控制显隐，默认保留内容；`unmountOnHide` 让原本按需挂载的内容在退出后卸载，退出期间保留上一帧的 slot 数据。`animate=false` 跳过动效，`appear` 控制首次可见挂载，`after-enter` / `after-leave` 通知完成。Grid 行高自动适配内容，过渡时裁剪，展开结束恢复正常溢出；收起内容设置 inert。表格明细使用 `ExpandableTableRow`，保持 tr/td 结构，关闭完成后不留空行。
- 原生 details 的交互迁入 `Disclosure`，折叠头使用按钮并关联 `aria-expanded` / `aria-controls`。校验和引导通过 `form-field-reveal` / `onboarding-reveal` 展开字段时，Collapse 跳过动画，保证下一次 DOM 更新后即可定位。
- 分段切换使用 `vSegmented` 定位轨道的 `::before` 背景，以 `--motion-normal`（200ms）和公共缓动过渡位置、宽度及高度。选中项仍由调用方的 `.segmented-item-active` 标记，原有点击、键盘和 ARIA 语义不变；不增加可聚焦节点。首屏和隐藏后重现直接定位，没有选中项时移除背景。指令在 Vue 更新和 ResizeObserver 通知时重新测量，覆盖不等宽文案、换行、字体及尺寸变化，卸载时释放观察器。快速切换由 CSS 从当前位置继续过渡，减少动态效果时沿用公共时长变量立即完成。
- 页签和页面内容使用 `vContentReveal`：默认对已有元素执行 150ms 透明度动画，不增加包装层或重新挂载组件。AppLayout 仅处理 main，AuthLayout 仅处理内容区，公开页面在自身内容容器接入。路由以 `route.path` 触发，同路径 query/hash 更新不重播；快速换页取消旧动画，导航进度条继续独立表示路由加载。保留页签原有的 v-if/v-show 策略。购买页用 `MotionTransition` 在同一视窗内横向切换，面板仅以 `activeTab` 为 key：进入 Subscribe 向左翻页，返回 Pay-as-you-go 向右翻页，持续 `--motion-layout`（220ms）。退出面板绝对定位并由公共生命周期隔离交互，动画期间裁剪视窗，结束后恢复摘要吸顶和浮层布局；同一页签内填写表单或推进支付流程不重新挂载面板。首次加载骨架不播放，减少动态效果时沿用公共时长变量直接切换。
- 两类弹窗共用 `useDialogLifecycle`，滚动锁保留到实际退出结束，快速重开不会重复计数，旧弹窗不覆盖新弹窗的焦点，Escape 仅关闭最上层 BaseDialog。父级按需挂载的安全凭证弹窗使用 `useLeavingPresence`，关闭后等待外壳 `after-leave` 再卸载。
- Toast 和用户直接增删的短列表使用 `motion-list`，以稳定业务 key 识别进入、退出与位置变化。分页表格、虚拟列表和轮询结果不逐行播放动效。
- 系统减少动态效果时，普通过渡变量缩短至 1ms、位移归零，Collapse 和内容淡入直接完成状态切换；动态改变偏好会取消内容淡入。完成和清理继续通过 Vue 生命周期、动画完成事件执行，不用业务定时器猜测结束时间。
- 专用动画保留自身几何：CreativeCanvas 工具条扩展、CreativeRunHistory 详情及 CustomPageView 目录抽屉共用普通时长和缓动；通用图标、加载反馈、计时进度和公开页数字滚动保留专用节奏与各自的减少动态效果处理。主题切换沿用临时关闭过渡的规则。
- 用户兑换成功使用卡片内的 `RedeemCelebration`：统计区保留占位，显示成功图标与权益信息，彩纸使用独立的 2 秒节奏并通过动画完成事件移除；提示保留 3 秒后按公共 fade 配方退出。每次成功以独立序号触发，再次提交或卸载时清理旧效果。减少动态效果时只展示静态成功信息，播放期间修改偏好也立即取消装饰；成功消息通过礼貌播报区域通知辅助技术。
- 用户仪表盘的专用动效时长集中在 `components/user/dashboard/dashboardMotion.ts`：脚本直接引用常量，样式读取页面根节点注入的 `--dash-*` 变量。动效包括指标数字滚动（`useCountUp`，四次缓出，读屏只读取最终值）、趋势图在切换指标或重新取数时按指标和数据版本重建，与首次打开一样从左到右裁剪描线、迷你走势线裁剪展开、热力图首次取数时按列错峰入场、模型占比条依次伸展，以及区块首次挂载时依次上移淡入（结束后不保留 transform）。减少动态效果时，以上动效全部直接显示最终状态。
- 同一次动效的 JS/CSS 时长必须同源。KeyUsageView 圆环和数字滚动共用 `RING_ANIMATION_MS`。`constants/ui.ts` 的 `COPY_FEEDBACK_MS`（2000）和 `SEARCH_DEBOUNCE_MS`（300）属于反馈保留与防抖，不并入过渡档位。公告连播仍由 after-leave 推进队列，组件途中卸载时由卸载回调完成清理。

<a id="rule_list_editor"></a>
## 行列表编辑器

逐条添加的映射和规则列表统一使用 `components/common/RuleListEditor.vue`。来源到目标的模型映射使用基于它的 `ModelMappingEditor.vue`，提供商弹窗再包一层 `components/provider/ProviderModelMappingEditor.vue`，默认带上提供商映射的说明、占位和预设。

- 外壳只负责展示，增删和排序以 `add`、`remove(index)`、`move(from, to)` 事件交给父级执行，所以能接入向上 emit 的组件和不可变更新的组件。
- 头部左侧是标题和说明，右侧是带加号的 `btn btn-secondary` 添加按钮，达到 `max` 时禁用。列表没有自己的标题时（标题和开关在外层），改用 `add-placement="footer"`，按钮放在列表下方，不再使用整宽虚线按钮。
- 标题旁的帮助入口放入 `title-suffix` 插槽，`ModelMappingEditor` 透传该插槽。Key 模型重定向使用点击展开的 `HelpTooltip`，请求卡片横向经过客户端、密钥规则和后续路由，在密钥节点展示模型名替换。关闭时卸载演示，减少动态效果时显示最终结果。
- 空态是 `rounded-control` 虚线框加居中说明，只在传入 `empty-text` 时显示。
- `line` 形态用分隔线隔开各行，适合一两个字段的短行，操作按钮使用 `btn-icon`，与 36px 输入框对齐。`card` 形态是 `rounded-surface` 浅底卡片，适合多字段规则；传入 `item-label` 时卡片头显示“规则 #n”和紧凑的 `btn-icon-sm` 操作按钮。
- 行操作只有上移 `arrowUp`、下移 `arrowDown` 和删除 `trash` 三种图标，默认灰色，删除悬停变红，不使用文字删除按钮。首行不能上移，末行不能下移，行数不超过 `min` 时不能删除。
- 行 key 默认使用对象身份（`createStableObjectKeyResolver`），不要用下标。行内容是基本类型、只能按下标区分时（如协议回退目标），传 `:animated="false"` 关闭列表动效，避免退出动画落在错误的行上。
- 点击添加按钮后，新行的第一个输入框或选择框获得焦点；预设和 JSON 导入追加的行不抢焦点。
- 删除时，后面还有行的退出行沿用 `motion-list` 的绝对定位，由后续行补位；末尾的退出行留在文档流中淡出，避免下方内容立即上移并与之重叠。空态等所有退出动画结束后才出现。
- 行对象原地编辑以保持身份，结构变化（增删、排序）时发出新数组。需要保留不可变更新契约的站点，应让克隆后的行继承原行的展示 key；例如公告定向编辑器用 WeakMap 保存组和条件的 key，避免编辑字段时重新挂载。展示 key 不写入提交数据。
- 字段错误使用 `.input-error` 与 `.input-error-text`，列表级汇总错误通过 `error` 以 `role="alert"` 显示。映射校验规则在 `utils/modelMappingRules.ts`，默认不开启，由站点按需传入 `KEY_REDIRECT_RULES`、`WILDCARD_ONLY_RULES` 等选项。统一样式时保持各站点原有的校验范围。
- 编辑器不改变提交给后端的数据形状：API Key 和分组仍提交映射对象，提供商仍经 `buildModelMappingObject` 构造。

已接入的编辑器：

- API Key 模型重定向与复合分组绑定；分组的模型映射、模型路由、推理强度映射和协议回退。
- 提供商的模型映射、紧凑模型映射、请求头覆写、临时不可调度规则，以及 TLS 指纹路由规则。
- 系统设置中的整流器签名模式、Beta 模型模式、OpenAI Fast 规则及模型模式、默认与认证来源订阅、提示词替换、搜索服务商、自定义端点、菜单、首页推荐模型、页脚分组及链接、登录协议文档、创作模型和配额通知邮箱。
- 定价页的模型定价条目、提供商统计规则及其定价条目；定价卡片的上下文区间、按次与媒体计费层级、时间段。
- EasyPay 自定义支付方式、公告定向的 OR 组与 AND 条件、运维告警静默条目、用户属性的下拉选项。

标签式输入（例如 `ModelTagInput`、`ModelWhitelistSelector` 和收件人 chip），以及带搜索与分页的倍率表格，继续使用各自的组件。价格配置和属性配置的模型规则共用 `ModelTagInput`，回车添加标签、粘贴批量导入、逐项删除，保存时仍提交模型数组。属性配置中的空模型规则需要补齐或删除后才能保存。

## 表格密度

全站只有一套密度：表头 `px-4 py-2 text-xs font-medium tracking-wider`，数据单元格 `px-4 py-3 text-sm`。`.table` 组件类、`TablePageLayout` 深度样式和 `DataTable` 必须保持一致。

分页表格的表体和 `Pagination` 共用一个 `rounded-surface` 外框，外框使用 `overflow-hidden` 裁剪底部圆角。分页放在表体滚动区域之外，两者之间不加 `space-y-*` 或外边距；固定高度的表格以 flex 分配表体高度，分页不收缩。`TablePageLayout` 已包含该结构；独立卡片和弹窗也遵守同一约定，嵌入 `UsageTable` 时传入 `flat`，避免重复边框。桌面分页统一使用上下 8px 留白、浅色 `gray-50/80` 和深色 `dark-900` 底色，控件保持 36px；窄屏继续显示上一页、下一页和当前页数。

两个合法例外：

- `DataTable` 按列数自适应横向 padding（`px-2/3/4/6`），宽表格不至于溢出；
- 选择列通过 `DataTable` 的 `.table-selection-cell` 统一使用 `w-11 min-w-11 px-3 text-center`，复选框为 `h-4 w-4`，内置行选择和自定义 `select` 列使用相同布局，外层页面的通用单元格样式不得覆盖。

启用左侧固定列时，选择列随横向滚动移出，首个数据列到达左边缘后固定；右侧操作列仍按 `stickyActionsColumn` 固定。自定义 `select` 列应放在列定义首位，不单独设置宽度或固定偏移。

`DataTable` 通过 `columnOrderStorageKey` 启用列顺序调整。每种表格使用稳定、独立的存储键；同一组件承载不同类型的表格时，键中包含类型。用户拖动表头手柄换列，手柄获得焦点后也可按左右方向键移动。手柄沿用排序指示器的 `h-5 w-4` 尺寸，表头高度不变；拖到横向滚动区域边缘时自动滚动。

选择列和操作列保留在原有位置，其余数据列可调整，左侧固定区域跟随调整后的首个数据列。表头、单元格、加载占位和移动端卡片使用同一列顺序。顺序保存在当前浏览器，隐藏列保留位置，新列追加到数据列末尾；存储不可用时仍允许在当前页面调整。拖拽手柄与数据升降序、表头内的筛选按钮各自处理事件。

## 字号

- 下限 `text-xs`（12px），禁止 `text-[9px]/[10px]/[11px]`；徽章和辅助数字也不例外。
- 正文与表格 `text-sm`，说明文字 `text-xs`，页面标题统一 `.page-title`（`text-2xl font-bold`），不散装 `h1` 字号。

## 合法例外

- 营销与落地页（`HomeView`、`KeyUsageView` 等公开页）的 hero 标题可用展示级字号。
- `onboarding.css` 覆盖 driver.js 第三方样式时的 `!important`。
- i18n 文案中内嵌的导览 HTML（`src/i18n/**`）属于内容字符串，其 inline style 不参与 token 校验。
- 测试文件里的负断言（断言某类名不存在）会命中扫描，行尾加 `check-ui-allow` 豁免。
- 弹窗分诊保留的手写外壳：BackupView R2Guide 与 SubscriptionsView 指南弹窗（max-w-2xl 无 BaseDialog 对应档位）、AnnouncementPopup 与 AnnouncementBell 弹窗（独立层级梯队 + 定制过渡）、两个 ProviderTestModal 的图片灯箱（媒体覆盖层，用强遮罩档）。新增弹窗默认走 BaseDialog，不复刻这些结构。
- RiskControlView 搜索框的 `pl-9` 图标留白（与 `input-icon-*` 档位值都不重合，局部保留）。
- ProvidersView 的鼠标跟随操作菜单（定位语义独特，不走 `getFloatingPanelPosition`）。
- textarea 内容驱动高度、GroupBadge 方角造型、OpsDashboard 的 250ms 路由同步防抖（语义不同于搜索防抖）、CreativeCanvas 工具条与 CreativeRunHistory 条目详情的结构性展开动画，均属局部语义，不强行入档。

## 校验

`frontend/` 下运行 `npm run check:ui`（`scripts/check-ui-tokens.mjs`）扫描上述规则，随 `lint:check` 一起作为提交前检查；具体拦截规则与正则见脚本头部注释（旧圆角、任意圆角/字号/z 值、36px 冗余、手写开关、max-h 任意像素、非响应式暗色判断、z-index 字面量、JS 断点字面量、tailwind.config 键约束）。新增组件样式前对照本文；确需偏离时在 PR 中说明理由，并考虑补充为例外条款。

## 相关文档

- [系统架构](system_architecture.md)：前端在整体部署中的组成与静态资源交付。
- [开发、验证与上游同步](../operations/development_workflow.md)：前端工具链与验证分层。
