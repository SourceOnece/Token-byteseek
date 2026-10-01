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

  it('忽略旧个人偏好，使用服务端注入值', () => {
    localStorage.setItem('visual-theme', 'bauhaus')
    initVisualTheme('tokenflux')
    expect(useVisualTheme().visualTheme.value).toBe('tokenflux')
    initVisualTheme('bauhaus')
    expect(useVisualTheme().visualTheme.value).toBe('bauhaus')
    initVisualTheme()
    expect(useVisualTheme().visualTheme.value).toBe('tokenflux')
  })

  it('站点配置更新不写个人偏好、不重建页面且不更改明暗模式', () => {
    initVisualTheme('tokenflux')
    document.documentElement.classList.add('dark')
    const input = document.createElement('input')
    input.value = '未保存草稿'
    document.body.append(input)
    setVisualTheme('bauhaus')
    expect(document.documentElement.classList.contains('theme-bauhaus')).toBe(true)
    expect(localStorage.getItem('visual-theme')).toBeNull()
    expect(document.documentElement.classList.contains('dark')).toBe(true) // check-ui-allow: 验证主题投影。
    expect(input.isConnected).toBe(true)
    expect(input.value).toBe('未保存草稿')
    setVisualTheme('tokenflux')
    expect(document.documentElement.classList.contains('theme-bauhaus')).toBe(false)
    expect(document.documentElement.dataset.visualTheme).toBe('tokenflux')
    input.remove()
  })
})
