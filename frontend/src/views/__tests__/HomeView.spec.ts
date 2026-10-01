import { nextTick, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from '../HomeView.vue'
import { initVisualTheme, setVisualTheme } from '@/composables/useVisualTheme'

const api = vi.hoisted(() => ({ models: vi.fn(), stats: vi.fn(), checkAuth: vi.fn() }))
const locale = ref('zh')
const app = reactive({ siteName: 'ByteSeek', siteLogo: '', docUrl: '', publicSettingsLoaded: true, cachedPublicSettings: { home_content: '', site_title_zh: '统一网关', site_title_en: 'One gateway' } })
vi.mock('@/api/marketplace', () => ({ getMarketplaceModels: api.models, getMarketplaceStats: api.stats }))
vi.mock('@/stores', () => ({ useAuthStore: () => ({ isAuthenticated: false, isAdmin: false, checkAuth: api.checkAuth }), useAppStore: () => app }))
vi.mock('vue-i18n', async () => ({ ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')), useI18n: () => ({ t: (key: string) => key, locale }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ path: '/home' }) }))
vi.mock('@/directives/contentReveal', () => ({ vContentReveal: {} }))

function renderHome() {
  return mount(HomeView, {
    global: {
      stubs: {
        GoogleOneTap: true, AppHeader: true, Icon: true, ProviderIcon: true, ModelIcon: true,
        GitHubMark: true, MotionTransition: { template: '<div><slot /></div>' },
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
}

describe('站点主题首页', () => {
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
    expect(wrapper.find('[data-testid="bauhaus-home"]').exists()).toBe(false)
    setVisualTheme('bauhaus')
    await nextTick()
    expect(wrapper.find('[data-testid="bauhaus-home-orbit"]').exists()).toBe(true)
    expect(wrapper.get('.bauhaus-home-stage-panel').attributes('href')).toBe('/models')
    const words = wrapper.findAll('.bh-home-marquee-copy').at(0)!.findAll('.bauhaus-home-marquee-word').map(n => n.text())
    expect(words).toEqual(['一站接入', '统一管理', '灵活切换', '智能中枢', '透明计费', '智能分发', '接口统一', '全链加速', '企业稳定'])
    expect(wrapper.findAll('.bh-home-marquee-copy')[1].attributes('aria-hidden')).toBe('true')
    locale.value = 'en'
    await nextTick()
    expect(wrapper.get('.bh-home-marquee-copy').text()).toContain('UNIFIED MANAGEMENT')
    expect(wrapper.get('h1').text()).toContain('One gateway')
    setVisualTheme('tokenflux')
    await nextTick()
    expect(wrapper.find('.bauhaus-home-stage').exists()).toBe(false)
    expect(wrapper.find('.home-marquee-track').exists()).toBe(true)
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
