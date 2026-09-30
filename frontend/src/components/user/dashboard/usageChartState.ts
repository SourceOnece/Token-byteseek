import { computed, inject, provide, ref, onScopeDispose, type InjectionKey } from 'vue'
import { usageAPI } from '@/api/usage'
import type { TrendDataPoint } from '@/types'
import {
  bucketKeys,
  fillTrendBuckets,
  formatDayKey,
  formatQueryTime,
  previousRange,
  resolveCustomRange,
  resolvePresetRange,
  summarizeTrend,
  type UsageRange,
  type UsageRangePreset,
  type UsageTotals,
} from './usageChartData'
import { useUsageChartFilters } from './useUsageChartFilters'

// UsageMetric 是指标卡和趋势图可以切换的指标。
export type UsageMetric = 'requests' | 'tokens' | 'cost' | 'cacheHitRate'

// PresetRange 是工具栏上的快捷时间范围。
export type PresetRange = Exclude<UsageRangePreset, 'custom'>

export const USAGE_METRICS: UsageMetric[] = ['requests', 'tokens', 'cost', 'cacheHitRate']
export const USAGE_RANGE_PRESETS: PresetRange[] = ['24h', '7d', '30d', '90d']

const STORAGE_KEY = 'dashboard-usage-chart'
// 日期选择器的快捷项与工具栏快捷范围重合时，选中后回到对应的快捷按钮。
const PICKER_PRESET_MAP: Record<string, PresetRange> = { '7days': '7d', '30days': '30d' }

// readSavedState 读取上次选择的指标与快捷范围，存储不可用时使用默认值。
const readSavedState = (): { metric: UsageMetric; range: PresetRange } => {
  const fallback = { metric: 'tokens' as UsageMetric, range: '7d' as PresetRange }
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    return {
      metric: USAGE_METRICS.includes(saved.metric) ? saved.metric : fallback.metric,
      range: USAGE_RANGE_PRESETS.includes(saved.range) ? saved.range : fallback.range,
    }
  } catch {
    return fallback
  }
}

// createUsageChartState 创建仪表盘用量区的共享状态：工具栏负责修改条件，指标卡和趋势图负责展示。
export function createUsageChartState() {
  const saved = readSavedState()
  const metric = ref<UsageMetric>(saved.metric)
  const rangePreset = ref<UsageRangePreset>(saved.range)
  const customRange = ref<UsageRange | null>(null)
  const activeRange = ref<UsageRange>(resolvePresetRange(saved.range))

  const loading = ref(false)
  const loaded = ref(false)
  const error = ref(false)
  const current = ref<TrendDataPoint[]>([])
  const previousTotals = ref<UsageTotals | null>(null)
  const totals = computed(() => summarizeTrend(current.value))

  const filterState = useUsageChartFilters()

  // 日期选择器显示当前窗口覆盖的日期，结束时间是排他边界，所以回退一毫秒。
  const pickerStart = computed(() => formatDayKey(activeRange.value.startAt))
  const pickerEnd = computed(() => formatDayKey(new Date(activeRange.value.endAt.getTime() - 1)))

  // saveState 只记住指标和快捷范围，自定义日期与筛选条件下次打开时回到默认值。
  const saveState = () => {
    try {
      const range = rangePreset.value === 'custom' ? readSavedState().range : rangePreset.value
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ metric: metric.value, range }))
    } catch {
      // 存储不可用时只在当前页面生效。
    }
  }

  // 用递增序号丢弃过期响应，快速切换范围或筛选时只保留最后一次结果。
  let requestSeq = 0
  onScopeDispose(() => { requestSeq++ })

  // fetchRange 拉取一个窗口的趋势数据，带上当前的筛选条件。
  const fetchRange = (range: UsageRange) => usageAPI.getDashboardTrend({
    start_date: formatQueryTime(range.startAt),
    end_date: formatQueryTime(range.endAt),
    granularity: range.granularity,
    ...filterState.queryParams.value,
  })

  // loadTrend 并发拉取当前窗口和上一周期；上一周期失败只隐藏环比，不影响主图。
  const loadTrend = async (range: UsageRange) => {
    const seq = ++requestSeq
    loading.value = true
    error.value = false
    const [currentResult, previousResult] = await Promise.allSettled([
      fetchRange(range),
      fetchRange(previousRange(range)),
    ])
    if (seq !== requestSeq) return
    if (currentResult.status === 'fulfilled') {
      current.value = fillTrendBuckets(currentResult.value.trend || [], bucketKeys(range))
    } else {
      console.error('Failed to load usage trend:', currentResult.reason)
      // 失败显式提示，不能把接口错误画成零消费。
      current.value = []
      error.value = true
    }
    previousTotals.value = previousResult.status === 'fulfilled'
      ? summarizeTrend(previousResult.value.trend || [])
      : null
    loading.value = false
    loaded.value = true
  }

  // load 按当前范围重新拉取趋势和模型候选项。
  const load = async () => {
    // 只有自定义范围时 customRange 才有值，其余情况按快捷范围实时换算。
    const range = customRange.value ?? resolvePresetRange(rangePreset.value as PresetRange)
    activeRange.value = range
    await Promise.all([loadTrend(range), filterState.loadModelOptions(range)])
  }

  // 筛选只影响趋势数据，候选项沿用当前范围的结果。
  const applyFilters = () => {
    void loadTrend(activeRange.value)
  }

  const resetFilters = () => {
    filterState.reset()
    applyFilters()
  }

  const selectMetric = (value: UsageMetric) => {
    metric.value = value
    saveState()
  }

  const selectPreset = (value: PresetRange) => {
    rangePreset.value = value
    customRange.value = null
    saveState()
    void load()
  }

  const selectCustomRange = (range: { startDate: string; endDate: string; preset: string | null }) => {
    const mapped = range.preset ? PICKER_PRESET_MAP[range.preset] : undefined
    if (mapped) {
      selectPreset(mapped)
      return
    }
    const resolved = resolveCustomRange(range.startDate, range.endDate)
    if (!resolved) return
    rangePreset.value = 'custom'
    customRange.value = resolved
    void load()
  }

  return {
    metric,
    rangePreset,
    activeRange,
    loading,
    loaded,
    error,
    current,
    previousTotals,
    totals,
    pickerStart,
    pickerEnd,
    filterState,
    load,
    loadFilterOptions: filterState.loadKeyAndGroupOptions,
    applyFilters,
    resetFilters,
    selectMetric,
    selectPreset,
    selectCustomRange,
  }
}

export type UsageChartState = ReturnType<typeof createUsageChartState>

const USAGE_CHART_STATE_KEY: InjectionKey<UsageChartState> = Symbol('usageChartState')

// provideUsageChartState 在页面上创建共享状态，供标题行的工具栏和正文里的图表共同使用。
export const provideUsageChartState = (): UsageChartState => {
  const state = createUsageChartState()
  provide(USAGE_CHART_STATE_KEY, state)
  return state
}

// injectUsageChartState 读取页面提供的共享状态。
export const injectUsageChartState = (): UsageChartState => {
  const state = inject(USAGE_CHART_STATE_KEY)
  if (!state) throw new Error('usage chart state is not provided')
  return state
}
