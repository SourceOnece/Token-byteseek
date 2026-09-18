import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexTicketStatus from '../CodexTicketStatus.vue'
import type { TicketAccountStatus } from '@/api/admin/codexTickets'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('CodexTicketStatus', () => {
  it('有效票显示绿色倒计时，到期立即变更，不维持绿色零秒', async () => {
    const now = Date.parse('2026-09-18T00:00:00Z')
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [
      { model: 'gpt-6-astra', state: 'ready', expires_at: new Date(now + 3483000).toISOString() },
      { model: 'gpt-5.6-sol', state: 'missing' }
    ] }
    const w = mount(CodexTicketStatus, { props: { now, status } })
    expect(w.text()).toContain('Astra')
    expect(w.text()).toContain('58m03s')
    expect(w.get('.text-emerald-700').text()).toBe('58m03s')
    expect(w.get('.text-yellow-700').text()).toContain('state.missing')
    await w.setProps({ now: now + 3483000 })
    expect(w.text()).toContain('state.expired')
    expect(w.find('.text-emerald-700').exists()).toBe(false)
    expect(w.text()).not.toContain('暂停')
    w.unmount()
  })

  it('读取失败不继续展示旧票据倒计时，也不显示成降智', () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'ready', expires_at: '2099-01-01T00:00:00Z' }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status, failed: true } })
    expect(w.text()).toContain('state.unavailable')
    expect(w.find('.text-emerald-700').exists()).toBe(false)
    expect(w.text()).not.toContain('degraded')
    w.unmount()
  })

  it('采集失败显示红色，非法原因内容不进入 DOM', () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'failed', reason: 'secret-from-upstream' }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status } })
    expect(w.get('.text-bh-red').text()).toContain('state.failed')
    expect(w.html()).not.toContain('secret-from-upstream')
    w.unmount()
  })
})
