import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexTicketStatus from '../CodexTicketStatus.vue'
import type { TicketAccountStatus } from '@/api/admin/codexTickets'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('CodexTicketStatus', () => {
  it('最新手动失败与旧有效票分开显示，不用倒计时盖住最新结果', () => {
    const now = Date.now()
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'ready', expires_at: new Date(now + 600000).toISOString(), latest: { source: 'manual', state: 'failed', checked_at: new Date(now).toISOString(), ip_status: 'timeout' } }] }
    const w = mount(CodexTicketStatus, { props: { status, now } })
    expect(w.get('[data-testid="ticket-latest"]').text()).toContain('source.manual')
    expect(w.get('[data-testid="ticket-latest"]').text()).toContain('status.failed')
    expect(w.get('[data-testid="ticket-current"]').text()).toContain('10m00s')
    expect(w.html()).toContain('ipStatus.timeout'); w.unmount()
  })
  it('模型门控显示暂停，不把它当作质量检测降智', () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'missing', blocked: true }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status } })
    expect(w.text()).toContain('tickets.modelBlocked')
    expect(w.text()).not.toContain('degraded')
    w.unmount()
  })
  it('无票展示 HTTP/头长度和固定错误分类，原始响应不参与显示', () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{
      model: 'gpt-6-astra', state: 'failed', diagnostic: { proxy_id: 'legacy', proxy_name: 'A', attempt: 2, http_status: 200, header_present: false, header_length: 0, prefix_valid: false, response_kind: 'sse', error_kind: 'overloaded' }
    }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status } })
    expect(w.get('[data-testid="ticket-diagnostic"]').text()).toContain('HTTP 200')
    expect(w.get('[data-testid="ticket-diagnostic"]').text()).toContain('tickets.noHeader')
    expect(w.get('[data-testid="ticket-diagnostic"]').text()).toContain('errorKind.overloaded')
    w.unmount()
  })
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
