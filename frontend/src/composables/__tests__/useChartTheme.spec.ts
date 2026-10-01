import { beforeEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick } from 'vue'
import { setVisualTheme } from '@/composables/useVisualTheme'
import { setTheme } from '@/composables/useTheme'
import { useChartTheme, CHART_PALETTE, CHART_SERIES_COLORS } from '@/composables/useChartTheme'

describe('useChartTheme', () => {
  beforeEach(async () => { setVisualTheme('bauhaus'); await nextTick() })
  it('colors 跟随主题切换响应式更新(回归:非响应式快照曾导致切主题不重绘)', () => {
    const scope = effectScope()
    const theme = scope.run(() => useChartTheme())!

    setTheme(false)
    expect(theme.colors.value).toEqual({ text: '#403D36', muted: '#6B655A', grid: '#D8D1C2' })

    setTheme(true)
    expect(theme.colors.value).toEqual({ text: '#EAE5D8', muted: '#B7B1A4', grid: '#514D43' })

    scope.stop()
    setTheme(false)
  })

  it('onThemeChange 回调在切换时触发,返回的停止函数可断开', async () => {
    const scope = effectScope()
    const theme = scope.run(() => useChartTheme())!
    const seen: boolean[] = []
    const stop = theme.onThemeChange((dark) => seen.push(dark))

    setTheme(true)
    await nextTick() // watch 默认 pre 队列,微任务后才回调
    setTheme(false)
    await nextTick()
    expect(seen).toEqual([true, false])

    stop()
    setTheme(true)
    await nextTick()
    expect(seen).toEqual([true, false])

    scope.stop()
    setTheme(false)
  })

  it('调色板 12 色保留已发布包豪斯图表顺序', () => {
    // 分类顺序属于视觉契约，不能因合并主题入口退回上游默认色。
    expect(CHART_PALETTE.slice(0, 10)).toEqual([
      '#E1251B', '#1450A3', '#E0A800', '#141414', '#5581C2',
      '#0F7B4D', '#E55A51', '#8A6D3B', '#403D36', '#97B7E8'
    ])
    expect(CHART_PALETTE).toHaveLength(12)
  })

  it('序列色锁定 token 趋势图五色语义', () => {
    expect(CHART_SERIES_COLORS).toEqual({
      input: '#1450A3',
      output: '#E1251B',
      cacheCreation: '#E0A800',
      cacheRead: '#0F7B4D',
      cacheHitRate: '#1450A3'
    })
  })
})
