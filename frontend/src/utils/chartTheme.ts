/**
 * Chart.js 包豪斯主题
 *
 * 在任一图表组件中 import 本模块即生效（副作用式应用）：
 *   import { BH_CHART_PALETTE } from '@/utils/chartTheme'
 *
 * 设计语言：
 * - 直线段（tension 0）而非贝塞尔曲线 —— 构成主义的线条
 * - 方形数据点、方形图例块 —— 一切皆直角
 * - 墨色细网格、加粗轴文字
 * - 提示框：墨底纸字、直角、硬边框
 */
import {
  ArcElement,
  BarElement,
  Chart,
  Legend,
  LineElement,
  PointElement,
  Tooltip
} from 'chart.js'

// 兼容现有消费者的导出；响应式主题单独读取纯色板，避免加载副作用。
export { BH_CHART_PALETTE, BH_CHART_NEUTRAL } from '@/constants/chartTheme'

const BH_FONT_FAMILY =
  '"Plus Jakarta Sans Variable", system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif'

/**
 * 应用图表主题。
 *
 * 必须通过 defaults.set 写入嵌套配置：部分按需加载场景下，Chart.js 尚未创建
 * defaults.elements.line 等分支，直接读取后赋值会让整个页面在模块求值阶段崩溃。
 */
export function applyBauhausChartTheme(): void {
  // 注册是幂等的；先注册保证 Chart.js 合并元素自身默认值，再覆盖主题值。
  Chart.register(LineElement, PointElement, ArcElement, BarElement, Legend, Tooltip)

  // 字体
  Chart.defaults.font.family = BH_FONT_FAMILY
  Chart.defaults.font.weight = 600

  // 线条：直线段 + 加粗
  Chart.defaults.set('elements.line', {
    tension: 0,
    borderWidth: 2.5
  })

  // 数据点：方形
  Chart.defaults.set('elements.point', {
    pointStyle: 'rect',
    radius: 3,
    hoverRadius: 5,
    borderWidth: 0
  })

  // 饼图 / 环形图扇区：纸色描边分割
  Chart.defaults.set('elements.arc', {
    borderWidth: 2
  })

  // 图例：方块色标 + 粗体
  Chart.defaults.set('plugins.legend.labels', {
    boxWidth: 12,
    boxHeight: 12,
    font: { family: BH_FONT_FAMILY, weight: 700, size: 12 }
  })

  // 提示框：墨底纸字、直角、硬边
  Chart.defaults.set('plugins.tooltip', {
    cornerRadius: 0,
    backgroundColor: '#141414',
    titleColor: '#FFCC00',
    bodyColor: '#F4F0E6',
    borderColor: '#F4F0E6',
    borderWidth: 1,
    titleFont: { family: BH_FONT_FAMILY, weight: 800, size: 12 },
    bodyFont: { family: BH_FONT_FAMILY, weight: 600, size: 12 },
    padding: 10
  })
}

applyBauhausChartTheme()
