<template>
  <div class="card min-w-0 space-y-4 p-4" :aria-busy="loading">
    <!-- 指标卡展示所选范围的合计与环比，点击后在下方折线图中查看该指标 -->
    <div
      class="grid grid-cols-2 gap-2"
      role="tablist"
      :aria-label="t('dashboard.usageChart.metricsLabel')"
    >
      <button
        v-for="item in metricTabs"
        :key="item.key"
        type="button"
        role="tab"
        :data-testid="`usage-metric-${item.key}`"
        :aria-selected="metric === item.key"
        class="metric-card btn flex min-w-0 items-center gap-2 p-3 text-left"
        :class="metric === item.key
          ? 'btn-warning'
          : 'btn-secondary'"
        @click="selectMetric(item.key)"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-1">
          <span class="text-xs" :class="metric === item.key ? 'text-gray-950' : 'text-gray-500 dark:text-dark-400'">{{ item.label }}</span>
          <Skeleton v-if="!loaded" :width="96" :height="28" />
          <span v-else class="truncate text-lg font-bold tabular-nums">{{ error ? '—' : item.value }}</span>
          <span v-if="loaded && !error" class="truncate text-xs" :class="metric === item.key ? 'text-gray-800' : 'text-gray-500 dark:text-dark-400'" :data-testid="`usage-delta-${item.key}`">
            <template v-if="item.delta">
              <span :class="deltaToneClasses[item.delta.tone]">{{ item.delta.text }}</span>
              {{ t('dashboard.usageChart.vsPrevious') }}
            </template>
            <template v-else>{{ t('dashboard.usageChart.noPrevious') }}</template>
          </span>
        </div>
        <!-- 只画走势线，选中的指标用品牌色 -->
        <UsageSparkline
          v-if="loaded && !error"
          class="metric-sparkline h-10 w-20 shrink-0"
          :class="metric === item.key ? 'text-primary-600 dark:text-primary-500' : 'text-gray-400 dark:text-dark-400'"
          :values="item.series"
          :data-testid="`usage-sparkline-${item.key}`"
        />
      </button>
    </div>

    <!-- 折线图独立成卡，标题跟随当前指标 -->
    <div>
      <div class="mb-4 flex min-h-7 flex-wrap items-center justify-between gap-2">
        <div class="flex items-baseline gap-2">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white" data-testid="usage-chart-title">{{ chartTitle }}</h3>
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ granularityHint }}</span>
        </div>
        <!-- Token 指标可以按分项开关堆叠面积 -->
        <div v-if="metric === 'tokens'" class="flex flex-wrap gap-2">
          <button
            v-for="series in tokenSeries"
            :key="series.key"
            type="button"
            class="btn btn-secondary btn-sm inline-flex items-center gap-2 text-xs"
            :class="hiddenTokenSeries.includes(series.key)
              ? 'text-gray-400 dark:text-dark-500'
              : 'text-gray-700 dark:text-dark-100'"
            :aria-pressed="!hiddenTokenSeries.includes(series.key)"
            :data-testid="`usage-series-${series.key}`"
            @click="toggleTokenSeries(series.key)"
          >
            <span
              class="h-2 w-2 rounded-full"
              :class="{ 'opacity-30': hiddenTokenSeries.includes(series.key) }"
              :style="{ backgroundColor: series.color }"
            ></span>
            {{ series.label }}
          </button>
        </div>
      </div>

      <div class="h-64 sm:h-80">
        <div v-if="error" role="alert" class="flex h-full flex-col items-center justify-center gap-3 text-red-600 dark:text-red-400"><span>{{ t('dashboard.usageChart.loadFailed') }}</span><button class="btn btn-secondary" @click="load">{{ t('common.retry') }}</button></div>
        <div v-else-if="!loaded" class="flex h-full items-center justify-center"><LoadingSpinner /></div>
        <div
          v-else-if="!hasUsage"
          data-testid="usage-chart-empty"
          class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t('dashboard.usageChart.empty') }}
        </div>
        <Line v-else :data="chartData" :options="chartOptions" :plugins="[crosshairPlugin]" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Filler,
  type Plugin,
  type ScriptableContext,
  type ScriptableLineSegmentContext,
  type TooltipItem,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import UsageSparkline from './UsageSparkline.vue'
import { useBalanceDisplay } from '@/composables/useBalanceDisplay'
import { useChartTheme, CHART_SERIES_COLORS, CHART_TICK_FONT_SIZE } from '@/composables/useChartTheme'
import { externalTooltipHandler, hideExternalTooltip } from '@/utils/chartExternalTooltip'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { cacheHitRateOf } from './usageChartData'
import { injectUsageChartState } from './usageChartState'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler)

onBeforeUnmount(hideExternalTooltip)

type TokenSeriesKey = 'input_tokens' | 'output_tokens' | 'cache_creation_tokens' | 'cache_read_tokens'

// 请求数和消费折线沿用包豪斯蓝，暗色用同源浅蓝保持对比。
const BRAND_LINE_COLOR = { light: '#1450A3', dark: '#97B7E8' }
// Token 堆叠从下到上的顺序：占比最大的缓存读取垫底，输出等小量贴在曲线顶部更容易看清。
const TOKEN_STACK_ORDER: TokenSeriesKey[] = ['cache_read_tokens', 'cache_creation_tokens', 'input_tokens', 'output_tokens']
// 最后一个时段尚未结束时，末段折线使用的虚线样式。
const OPEN_SEGMENT_DASH = [4, 4]

const { t } = useI18n()
const { formatBalanceAmount } = useBalanceDisplay()
const { colors: themeColors, isDark } = useChartTheme()
const reducedMotion = usePreferredReducedMotion()
const {
  metric,
  activeRange,
  loading,
  loaded,
  current,
  previousTotals,
  totals,
  selectMetric,
  error,
  load,
} = injectUsageChartState()

const hiddenTokenSeries = ref<TokenSeriesKey[]>([])

const chartTitle = computed(() => t('dashboard.usageChart.trendTitle', {
  metric: t(`dashboard.usageChart.metrics.${metric.value}`),
}))

const granularityHint = computed(() => (
  activeRange.value.granularity === 'hour' ? t('dashboard.usageChart.byHour') : t('dashboard.usageChart.byDay')
))

const deltaToneClasses = {
  up: 'text-emerald-600 dark:text-emerald-400',
  down: 'text-red-600 dark:text-red-400',
  flat: 'text-gray-500 dark:text-dark-400',
}

type Delta = { text: string; tone: keyof typeof deltaToneClasses }

// percentDelta 计算环比百分比；上一周期为 0 时无法比较。
const percentDelta = (currentValue: number, previousValue: number | undefined): Delta | null => {
  if (previousValue === undefined || previousValue <= 0) return null
  const change = ((currentValue - previousValue) / previousValue) * 100
  const percent = Math.round(change)
  if (percent === 0) return { text: '0%', tone: 'flat' }
  return { text: `${percent > 0 ? '+' : ''}${percent}%`, tone: percent > 0 ? 'up' : 'down' }
}

// pointDelta 计算命中率的百分点差值。
const pointDelta = (currentValue: number | null, previousValue: number | null | undefined): Delta | null => {
  if (currentValue === null || previousValue === null || previousValue === undefined) return null
  const change = Math.round((currentValue - previousValue) * 10) / 10
  if (change === 0) return { text: '0 pt', tone: 'flat' }
  return { text: `${change > 0 ? '+' : ''}${change.toFixed(1)} pt`, tone: change > 0 ? 'up' : 'down' }
}

const formatPercent = (value: number | null): string => (value === null ? '—' : `${value.toFixed(1)}%`)

// 小额消费保留 4 位小数，避免显示成 0.00。
const formatCostTotal = (value: number): string => formatBalanceAmount(value, { fractionDigits: value >= 1 ? 2 : 4 })

const metricTabs = computed(() => {
  const now = totals.value
  const prev = previousTotals.value
  return [
    {
      key: 'requests' as const,
      label: t('dashboard.usageChart.metrics.requests'),
      value: formatNumber(now.requests),
      delta: percentDelta(now.requests, prev?.requests),
      series: current.value.map((point) => point.requests),
    },
    {
      key: 'tokens' as const,
      label: t('dashboard.usageChart.metrics.tokens'),
      value: formatTokensK(now.tokens),
      delta: percentDelta(now.tokens, prev?.tokens),
      series: current.value.map((point) => point.total_tokens),
    },
    {
      key: 'cost' as const,
      label: t('dashboard.usageChart.metrics.cost'),
      value: formatCostTotal(now.cost),
      delta: percentDelta(now.cost, prev?.cost),
      series: current.value.map((point) => point.actual_cost),
    },
    {
      key: 'cacheHitRate' as const,
      label: t('dashboard.usageChart.metrics.cacheHitRate'),
      value: formatPercent(now.cacheHitRate),
      delta: pointDelta(now.cacheHitRate, prev?.cacheHitRate),
      series: current.value.map((point) => cacheHitRateOf(point.input_tokens, point.cache_creation_tokens, point.cache_read_tokens)),
    },
  ]
})

const tokenSeries = computed(() => [
  { key: 'input_tokens' as const, label: t('dashboard.usageChart.series.input'), color: CHART_SERIES_COLORS.input },
  { key: 'output_tokens' as const, label: t('dashboard.usageChart.series.output'), color: CHART_SERIES_COLORS.output },
  { key: 'cache_creation_tokens' as const, label: t('dashboard.usageChart.series.cacheCreation'), color: CHART_SERIES_COLORS.cacheCreation },
  { key: 'cache_read_tokens' as const, label: t('dashboard.usageChart.series.cacheRead'), color: CHART_SERIES_COLORS.cacheRead },
])

const hasUsage = computed(() => totals.value.requests > 0 || totals.value.tokens > 0 || totals.value.cost > 0)

// 横轴按粒度只显示时分或月日，完整时间留给提示标题。
const axisLabels = computed(() => current.value.map((point) => (
  activeRange.value.granularity === 'hour' ? point.date.slice(11, 16) : point.date.slice(5, 10)
)))

const brandColor = computed(() => (isDark.value ? BRAND_LINE_COLOR.dark : BRAND_LINE_COLOR.light))

// 窗口包含当前时刻时，最后一个时段（今天或当前小时）还没结束，数据不完整。
const openBucketIndex = computed(() => (
  activeRange.value.endAt.getTime() > Date.now() && current.value.length > 1 ? current.value.length - 1 : null
))

// 未结束的末段画成虚线，避免把还在累计的数据误读成下跌。
const segmentStyle = {
  borderDash: (context: ScriptableLineSegmentContext) => (
    context.p1DataIndex === openBucketIndex.value ? OPEN_SEGMENT_DASH : undefined
  ),
}

// gradientFill 生成从折线向下逐渐变淡的填充；图表尚未完成布局时先不填充。
const gradientFill = (color: string) => (context: ScriptableContext<'line'>) => {
  const { ctx, chartArea } = context.chart
  if (!chartArea) return 'transparent'
  const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
  gradient.addColorStop(0, `${color}40`)
  gradient.addColorStop(1, `${color}00`)
  return gradient
}

// lineDataset 生成统一样式的折线数据集；monotone 插值保证曲线不越过真实数据点。
const lineDataset = (
  label: string,
  data: Array<number | null>,
  color: string,
  fill: boolean | string,
  background: string | ((context: ScriptableContext<'line'>) => CanvasGradient | string),
) => ({
  label,
  data,
  borderColor: color,
  backgroundColor: background,
  borderWidth: 2,
  fill,
  cubicInterpolationMode: 'monotone' as const,
  segment: segmentStyle,
  pointRadius: 0,
  pointHoverRadius: 4,
  pointHoverBorderWidth: 2,
  pointHoverBorderColor: isDark.value ? '#0F0F10' : '#FFFFFF',
  pointHoverBackgroundColor: color,
  spanGaps: true,
})

const chartData = computed(() => {
  const points = current.value
  if (metric.value === 'tokens') {
    // 堆叠面积：第一层填到底，后续每层填到前一层，最上沿即可见分项之和。
    const visible = TOKEN_STACK_ORDER
      .filter((key) => !hiddenTokenSeries.value.includes(key))
      .map((key) => tokenSeries.value.find((series) => series.key === key)!)
    return {
      labels: axisLabels.value,
      datasets: visible.map((series, index) => ({
        ...lineDataset(series.label, points.map((point) => point[series.key]), series.color, index === 0 ? 'origin' : '-1', `${series.color}40`),
        borderWidth: 1.5,
      })),
    }
  }
  if (metric.value === 'cacheHitRate') {
    return {
      labels: axisLabels.value,
      datasets: [lineDataset(
        t('dashboard.usageChart.metrics.cacheHitRate'),
        points.map((point) => cacheHitRateOf(point.input_tokens, point.cache_creation_tokens, point.cache_read_tokens)),
        CHART_SERIES_COLORS.cacheHitRate,
        'origin',
        gradientFill(CHART_SERIES_COLORS.cacheHitRate),
      )],
    }
  }
  const isCost = metric.value === 'cost'
  return {
    labels: axisLabels.value,
    datasets: [lineDataset(
      isCost ? t('dashboard.usageChart.series.actualCost') : t('dashboard.usageChart.metrics.requests'),
      points.map((point) => (isCost ? point.actual_cost : point.requests)),
      brandColor.value,
      'origin',
      gradientFill(brandColor.value),
    )],
  }
})

// formatAxisValue 格式化纵轴刻度，消费刻度不带货币符号以免挤占宽度。
const formatAxisValue = (value: number): string => {
  if (metric.value === 'tokens') return formatTokensK(value)
  if (metric.value === 'cacheHitRate') return `${value}%`
  if (metric.value === 'cost') return String(Math.round(value * 10000) / 10000)
  return formatNumber(value)
}

// formatTooltipValue 格式化提示中的单个数值。
const formatTooltipValue = (value: number | null): string => {
  if (value === null) return '—'
  if (metric.value === 'tokens') return formatTokensK(value)
  if (metric.value === 'cacheHitRate') return formatPercent(value)
  if (metric.value === 'cost') return formatBalanceAmount(value, { fractionDigits: 4 })
  return formatNumber(value)
}

const chartOptions = computed(() => ({
  animation: reducedMotion.value === 'reduce' ? false as const : undefined,
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { display: false },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      // 堆叠时按从上到下的顺序列出分项，与图形层次一致。
      itemSort: (a: TooltipItem<'line'>, b: TooltipItem<'line'>) => b.datasetIndex - a.datasetIndex,
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => current.value[items[0]?.dataIndex]?.date ?? '',
        label: (context: TooltipItem<'line'>) =>
          `${context.dataset.label}: ${formatTooltipValue(context.raw as number | null)}`,
        footer: (items: TooltipItem<'line'>[]) => {
          if (metric.value !== 'tokens' || items.length < 2) return ''
          const sum = items.reduce((total, item) => total + ((item.raw as number | null) ?? 0), 0)
          return `${t('dashboard.usageChart.series.total')}: ${formatTokensK(sum)}`
        },
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: {
        color: themeColors.value.text,
        autoSkip: true,
        maxTicksLimit: 8,
        maxRotation: 0,
        font: { size: CHART_TICK_FONT_SIZE },
      },
    },
    y: {
      stacked: metric.value === 'tokens',
      beginAtZero: true,
      // 顶部留白，最高点不贴着图表上沿；命中率固定 0-100%。
      ...(metric.value === 'cacheHitRate' ? { max: 100 } : { grace: '10%' }),
      grid: { color: themeColors.value.grid, drawTicks: false },
      border: { display: false, dash: [3, 3] },
      ticks: {
        color: themeColors.value.text,
        padding: 8,
        maxTicksLimit: 5,
        font: { size: CHART_TICK_FONT_SIZE },
        callback: (value: string | number) => formatAxisValue(Number(value)),
      },
    },
  },
}))

// crosshairPlugin 在悬停的时间点画一条贯穿绘图区的竖线，画在折线下层。
const crosshairPlugin: Plugin<'line'> = {
  id: 'usageCrosshair',
  beforeDatasetsDraw(chart) {
    const active = chart.getActiveElements()
    if (!active.length) return
    const { ctx, chartArea } = chart
    const x = active[0].element.x
    ctx.save()
    ctx.strokeStyle = themeColors.value.muted
    ctx.globalAlpha = 0.5
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(x, chartArea.top)
    ctx.lineTo(x, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  },
}

const toggleTokenSeries = (key: TokenSeriesKey) => {
  hiddenTokenSeries.value = hiddenTokenSeries.value.includes(key)
    ? hiddenTokenSeries.value.filter((item) => item !== key)
    : [...hiddenTokenSeries.value, key]
}
</script>

<style scoped>
/* 卡片内容宽度够放下数字和走势线时才显示走势线，窄卡片优先保证数字完整。 */
.metric-card {
  container-type: inline-size;
}

.metric-sparkline {
  display: none;
}

@container (min-width: 200px) {
  .metric-sparkline {
    display: block;
  }
}
</style>
