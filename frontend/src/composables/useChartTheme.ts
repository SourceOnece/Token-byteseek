import { computed, watch } from 'vue'
import { useTheme } from './useTheme'
import { BH_CHART_PALETTE, BH_CHART_NEUTRAL } from '@/constants/chartTheme'

/**
 * 图表主题与共享色板的唯一来源。
 *
 * 刻度/网格色只分三档语义:text(刻度与图例正文,强对比)、muted(次要说明,弱化)、
 * grid(网格线)。保留响应式公共入口，分类色复用 ByteSeek 既有色板；业务专用色
 * (成员固定配色、品牌强调色)不从这里取,继续留在各图表本地。
 */

/** 分布图(doughnut)的 12 色调色板,按切片排名依次取色;全站唯一一份。 */
export const CHART_PALETTE = [...BH_CHART_PALETTE]

/** 分布图 “Others” 聚合切片的固定灰。 */
export const CHART_OTHER_COLOR = BH_CHART_NEUTRAL

/** token 用量趋势序列的语义色(输入/输出/缓存创建/缓存读取/缓存命中率)。 */
export const CHART_SERIES_COLORS = {
  input: '#1450A3',
  output: '#E1251B',
  cacheCreation: '#E0A800',
  cacheRead: '#0F7B4D',
  cacheHitRate: '#141414'
} as const

/** 坐标刻度字号(全站图表统一)。 */
export const CHART_TICK_FONT_SIZE = 10

/** 图例字号(全站图表统一;此前 10/11 两值漂移,归一为 11)。 */
export const CHART_LEGEND_FONT_SIZE = 11

export function useChartTheme() {
  // 响应式主题状态:切换主题后依赖 colors 的图表配置会同步刷新,
  // 不要再用 document.documentElement.classList 手写非响应式判断。
  const { isDark } = useTheme()

  const colors = computed(() =>
    isDark.value
      ? { text: '#EAE5D8', muted: '#B7B1A4', grid: '#514D43' }
      : { text: '#403D36', muted: '#6B655A', grid: '#D8D1C2' }
  )

  /**
   * 主题切换后触发重绘。Chart.js 不会自动跟随 CSS 变量,
   * 回调里应重建配置或调用 chart.update('none')(禁动画避免切换抖动)。
   * 返回 watch 的停止函数,组件卸载时可显式断开。
   */
  const onThemeChange = (callback: (dark: boolean) => void) => {
    return watch(isDark, (dark) => callback(dark))
  }

  return { isDark, colors, onThemeChange }
}
