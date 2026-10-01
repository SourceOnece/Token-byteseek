import { nextTick, reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import { baseCompile } from '@intlify/message-compiler'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from '../HomeView.vue'
import { initVisualTheme, setVisualTheme } from '@/composables/useVisualTheme'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const api = vi.hoisted(() => ({ models: vi.fn(), stats: vi.fn(), checkAuth: vi.fn() }))
// 使用真实词典，不能让返回 key 的 mock 掩盖首页丢翻译。
const missingKeys = vi.fn()
// 测试配置使用 runtime-only i18n，先用同源编译器预编译仓库词典。
function compileMessages(value: any): any {
  if (typeof value === 'string') return new Function(`return ${baseCompile(value, { mode: 'arrow' }).code}`)()
  return Object.fromEntries(Object.entries(value).map(([key, entry]) => [key, compileMessages(entry)]))
}
const translations = createI18n({ legacy: false, locale: 'zh', fallbackLocale: false, messages: { zh: compileMessages(zh), en: compileMessages(en) }, missing: missingKeys })
const locale = translations.global.locale
const app = reactive({ siteName: 'ByteSeek', siteLogo: '', docUrl: '', publicSettingsLoaded: true, cachedPublicSettings: { home_content: '', site_title_zh: '统一网关', site_title_en: 'One gateway' } })
vi.mock('@/api/marketplace', () => ({ getMarketplaceModels: api.models, getMarketplaceStats: api.stats }))
vi.mock('@/stores', () => ({ useAuthStore: () => ({ isAuthenticated: false, isAdmin: false, checkAuth: api.checkAuth }), useAppStore: () => app }))
vi.mock('vue-router', () => ({ useRoute: () => ({ path: '/home' }) }))
vi.mock('@/directives/contentReveal', () => ({ vContentReveal: {} }))

function renderHome() {
  return mount(HomeView, {
    global: {
      plugins: [translations],
      stubs: {
        GoogleOneTap: true, AppHeader: true, Icon: true, ProviderIcon: true, ModelIcon: true,
        GitHubMark: true, MotionTransition: { template: '<div><slot /></div>' },
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
}

describe('站点主题首页', () => {
  it.each(['zh', 'en'])('首页 %s 只保留用户和模型统计，两种主题都不展示 Token 总量', async language => {
    locale.value = language
    api.stats.mockResolvedValue({ today_tokens: 123456789, total_tokens: 987654321, total_users: 42 })
    const wrapper = renderHome()
    await flushPromises()
    for (const skin of ['tokenflux', 'bauhaus']) {
      setVisualTheme(skin)
      await nextTick()
      expect(wrapper.findAll('[data-home-stat]').map(card => card.attributes('data-home-stat'))).toEqual(['total-users', 'supported-models'])
      expect(wrapper.text()).not.toMatch(/今日总 Token 量|历史总 Token 量|Today Total Tokens|Historical Total Tokens/)
    }
    expect(api.stats).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
  it('首页所有静态翻译键在中英文词典中均存在', () => {
    const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../HomeView.vue'), 'utf8')
    const keys = [...source.matchAll(/\bt\(['"]([^'"]+)['"]/g)].map(match => match[1])
    for (const dictionary of [zh, en]) for (const key of keys) {
      const message = key.split('.').reduce<any>((value, segment) => value?.[segment], dictionary)
      expect(typeof message, key).toBe('string')
    }
  })
  beforeEach(() => {
    vi.clearAllMocks()
    locale.value = 'zh'
    app.cachedPublicSettings.home_content = ''
    initVisualTheme('tokenflux')
    api.models.mockResolvedValue([])
    api.stats.mockResolvedValue({ today_tokens: 10, total_tokens: 20, total_users: 1 })
  })

  it('切换呈现时复用已加载的数据，中文走马灯不逐词重复，英文随语言切换', async () => {
    const wrapper = renderHome()
    await flushPromises()
    const original = wrapper.get('[data-testid="home-content"]').element
    expect(wrapper.text()).toContain('稳定可靠')
    expect(wrapper.html()).not.toMatch(/home\.(features|steps|providers|stats)\./)
    setVisualTheme('bauhaus')
    await nextTick()
    expect(wrapper.find('[data-testid="bauhaus-home-orbit"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="home-content"]').element).toBe(original)
    expect(wrapper.get('.bauhaus-home-stage-panel').attributes('href')).toBe('/models')
    const words = wrapper.findAll('.bh-home-marquee-copy').at(0)!.findAll('.bauhaus-home-marquee-word').map(n => n.text())
    expect(words).toEqual(['一站接入', '统一管理', '灵活切换', '智能中枢', '透明计费', '智能分发', '接口统一', '全链加速', '企业稳定'])
    expect(wrapper.findAll('.bh-home-marquee-copy')[1].attributes('aria-hidden')).toBe('true')
    locale.value = 'en'
    await nextTick()
    expect(wrapper.get('.bh-home-marquee-copy').text()).toContain('UNIFIED MANAGEMENT')
    expect(wrapper.get('h1').text()).toContain('One gateway')
    expect(wrapper.text()).toContain('Always Reliable')
    setVisualTheme('tokenflux')
    await nextTick()
    expect(wrapper.find('.bauhaus-home-stage').exists()).toBe(true)
    expect(wrapper.get('[data-testid="home-content"]').element).toBe(original)
    expect(wrapper.get('.home-landing').attributes('data-home-theme')).toBe('tokenflux')
    expect(missingKeys).not.toHaveBeenCalled()
    expect(api.models).toHaveBeenCalledOnce()
    expect(api.stats).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it.each(['tokenflux', 'bauhaus'])('自定义首页优先，不因 %s 主题发起默认首页查询', async theme => {
    app.cachedPublicSettings.home_content = '<p data-testid="configured-home">Configured</p>'
    initVisualTheme(theme)
    const wrapper = renderHome()
    await flushPromises()
    expect(wrapper.get('[data-testid="configured-home"]').text()).toBe('Configured')
    expect(wrapper.find('main').exists()).toBe(false)
    expect(api.models).not.toHaveBeenCalled()
    expect(api.stats).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
