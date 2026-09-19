import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexTicketStatus from '../CodexTicketStatus.vue'
import type { TicketAccountStatus } from '@/api/admin/codexTickets'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('CodexTicketStatus', () => {
  it('手机点击详情展示本次长度调度结果，不把获票当满血检测', async () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'missing', diagnostic: { proxy_id: 'legacy', proxy_name: 'A', attempt: 1, header_length: 312, header_present: true, prefix_valid: true, scheduling: 'disabled' } }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status }, global: { stubs: { BaseDialog: { template: '<div data-testid="dialog"><slot/></div>' } } } })
    await w.get('[data-testid="ticket-model-detail"]').trigger('click')
    expect(w.get('[data-testid="dialog"]').text()).toContain('ticketWorkbench.scheduling.disabled')
    w.unmount()
  })
  it('点击模型查看守护次数及原因，读取失败不继续展示旧异常', async () => {
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'gpt-6-astra', state: 'missing', watchdog: { mode: 'observe', count: 3, reason: 'model_mismatch', action: 'observed', checked_at: new Date().toISOString() } }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status }, global: { stubs: { BaseDialog: { template: '<div><slot/></div>' } } } })
    expect(w.get('[data-testid="ticket-watchdog-signal"]').text()).toContain('3')
    await w.get('[data-testid="ticket-model-detail"]').trigger('click')
    expect(w.get('[data-testid="ticket-watchdog-detail"]').text()).toContain('guards.observe')
    await w.setProps({ failed: true }); expect(w.find('[data-testid="ticket-watchdog-signal"]').exists()).toBe(false); expect(w.find('[data-testid="ticket-watchdog-detail"]').exists()).toBe(false); w.unmount()
  })
  it('实际/目标长度独立高亮，点击自定义模型打开详情并移除旧长提示', async () => {
    const diagnostic = { proxy_id: 'legacy', proxy_name: 'A', attempt: 1, http_status: 200, header_length: 356, header_present: true, prefix_valid: true, degraded_signal: true }
    const status: TicketAccountStatus = { account_id: 1, eligible: true, collection_paused: false, models: [{ model: 'custom-codex', target_length: 292, state: 'missing', diagnostic }] }
    const w = mount(CodexTicketStatus, { props: { now: Date.now(), status }, global: { stubs: { BaseDialog: { template: '<div data-testid="dialog"><slot/><slot name="footer"/></div>' } } } })
    expect(w.get('[data-testid="ticket-length-ratio"]').text()).toBe('356/292')
    expect(w.get('[data-testid="ticket-length-actual"]').classes()).toContain('text-bh-red')
    expect(w.get('[data-testid="ticket-length-target"]').classes()).toContain('text-emerald-700')
    expect(w.get('[data-testid="ticket-length-ratio"]').classes().some(c => c.startsWith('border'))).toBe(false)
    expect(w.get('[data-testid="ticket-model-detail"]').element.tagName).toBe('BUTTON')
    expect(w.get('[data-testid="ticket-degraded-signal"]').text()).toContain('degradedSignal')
    expect(w.find('[data-testid="dialog"]').exists()).toBe(false)
    await w.get('[data-testid="ticket-model-detail"]').trigger('click')
    expect(w.get('[data-testid="dialog"]').text()).toContain('tickets.currentTicket')
    expect(w.get('[data-testid="dialog"]').attributes('title')).toBe('custom-codex')
    expect(w.html()).not.toContain('notQuality')
    await w.setProps({ status: { ...status, models: [{ ...status.models[0], diagnostic: { ...diagnostic, header_length: 292 } }] } })
    // 信号命中即使等于目标也保留红色，左右不能被同一状态染色。
    expect(w.get('[data-testid="ticket-length-actual"]').classes()).toContain('text-bh-red')
    await w.setProps({ status: { ...status, models: [{ ...status.models[0], diagnostic: { ...diagnostic, header_length: 292, degraded_signal: false } }] } })
    expect(w.get('[data-testid="ticket-length-actual"]').classes()).toContain('text-emerald-700')
    await w.setProps({ status: { ...status, account_id: 2 } }); expect(w.find('[data-testid="dialog"]').exists()).toBe(false); w.unmount()
  })
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
