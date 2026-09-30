import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import CodexQualityDashboard from '../CodexQualityDashboard.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/admin/codexQuality', () => ({
  qualitySchedulesAPI: {
    stats: vi.fn().mockResolvedValue({ full: 2, degraded: 1, failed: 1, untested: 1, stale: 1, cancelled: 1 })
  }
}))

const unmounts: Array<() => void> = []
afterEach(() => { unmounts.splice(0).forEach(unmount => unmount()) })

// 使用真实路由跳转验证目的页和参数，不能只断言 Summary 发出了 select 事件。
describe('仪表盘检测统计导航', () => {
  it.each([
    ['', 'admin.accounts.quality.poolRate'],
    ['full', 'admin.accounts.quality.status.full'],
    ['degraded', 'admin.accounts.quality.status.degraded'],
    ['failed', 'admin.accounts.quality.status.failed'],
    ['untested', 'admin.accounts.quality.untested'],
    ['stale', 'admin.accounts.quality.status.stale'],
    ['cancelled', 'admin.accounts.quality.status.cancelled']
  ])('点击 %s 进入当前账号管理并带检测条件', async (status, label) => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/admin/dashboard', name: 'AdminDashboard', component: { template: '<div />' } },
        { path: '/admin/providers', name: 'AdminProviders', component: { template: '<div />' } },
        { path: '/:pathMatch(.*)*', redirect: '/admin/dashboard' }
      ]
    })
    await router.push('/admin/dashboard')
    const wrapper = mount(CodexQualityDashboard, { global: { plugins: [router] } })
    unmounts.push(() => wrapper.unmount())
    await flushPromises()
    expect(wrapper.text()).toContain('50.0%')
    const button = wrapper.findAll('button').find(item => item.text().includes(label))
    expect(button).toBeDefined()
    await button!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/providers')
    expect(router.currentRoute.value.query).toEqual({ platform: 'openai', ...(status ? { quality_status: status } : {}) })
  })
})
