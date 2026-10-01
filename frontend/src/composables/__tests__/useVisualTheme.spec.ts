import { beforeEach, describe, expect, it } from 'vitest'
import { initVisualTheme, setVisualTheme, useVisualTheme } from '../useVisualTheme'

describe('useVisualTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
  })

  it('默认使用 TokenFlux，与明暗模式独立', () => {
    initVisualTheme()
    expect(useVisualTheme().visualTheme.value).toBe('tokenflux')
    expect(document.documentElement.dataset.visualTheme).toBe('tokenflux')
    expect(document.documentElement.classList.contains('dark')).toBe(false) // check-ui-allow: 测试只验证 DOM 主题投影。
  })

  it('可以切换并记住包豪斯，切回原生皮肤不重建页面', () => {
    initVisualTheme()
    setVisualTheme('bauhaus')
    expect(document.documentElement.classList.contains('theme-bauhaus')).toBe(true)
    expect(localStorage.getItem('visual-theme')).toBe('bauhaus')
    initVisualTheme()
    expect(useVisualTheme().visualTheme.value).toBe('bauhaus')

    setVisualTheme('tokenflux')
    expect(document.documentElement.classList.contains('theme-bauhaus')).toBe(false)
    expect(document.documentElement.dataset.visualTheme).toBe('tokenflux')
  })
})
