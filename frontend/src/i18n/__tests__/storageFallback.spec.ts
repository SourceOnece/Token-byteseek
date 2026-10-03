import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// 直接导入初始化入口，避免只测工具函数却漏掉页面启动时的调用。
describe('语言初始化存储兼容', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
  })
  afterEach(() => vi.restoreAllMocks())

  it('读取被禁止时仍按浏览器语言启动', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('zh-TW')
    const { getLocale } = await import('../index')
    expect(getLocale()).toBe('zh')
  })

  it('本站旧键优先，不被 TokenFlux 同域偏好覆盖', async () => {
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('tokenrouter_locale', 'en')
    const { getLocale } = await import('../index')
    expect(getLocale()).toBe('zh')
  })

  it('没有本站记录时兼容上游键，写入失败仍可读取', async () => {
    localStorage.setItem('tokenrouter_locale', 'zh')
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota') })
    const { getLocale } = await import('../index')
    expect(getLocale()).toBe('zh')
  })
})
