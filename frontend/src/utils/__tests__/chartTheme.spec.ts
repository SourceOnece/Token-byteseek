import { describe, expect, it } from 'vitest'

describe('chartTheme', () => {
  // 回归：主题模块可能先于任何图表组件的 Chart.register 执行，
  // 必须自行注册元素，否则 defaults.elements.line 为 undefined 直接抛错。
  it('applies bauhaus defaults without prior element registration', async () => {
    const { applyBauhausChartTheme, BH_CHART_PALETTE } = await import('@/utils/chartTheme')
    const { Chart } = await import('chart.js')

    // 模拟动态分包首次加载时元素默认分支尚不存在的状态。
    delete (Chart.defaults.elements as Record<string, unknown>).line
    delete (Chart.defaults.elements as Record<string, unknown>).point

    expect(() => applyBauhausChartTheme()).not.toThrow()
    expect(BH_CHART_PALETTE.length).toBeGreaterThan(0)
    expect(Chart.defaults.elements.line.tension).toBe(0)
    expect(Chart.defaults.elements.point.pointStyle).toBe('rect')
    expect(Chart.defaults.plugins.tooltip.cornerRadius).toBe(0)
    expect(Chart.defaults.font.family).toContain('Plus Jakarta Sans')
  })
})
