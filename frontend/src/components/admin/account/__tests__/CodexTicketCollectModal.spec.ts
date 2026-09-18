import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import CodexTicketCollectModal from '../CodexTicketCollectModal.vue'

const { settings, runs, detail, start } = vi.hoisted(() => ({ settings: vi.fn(), runs: vi.fn(), detail: vi.fn(), start: vi.fn() }))
const admin = reactive({ user: { id: 1 } })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => admin }))
vi.mock('@/api/admin/codexTickets', () => ({ ticketCollectionAPI: { settings, runs, detail }, runTicketCollection: start }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (s: string) => s }) }))
const config = { enabled: true, target_length: 332, revision: 'r1', max_attempts: 123, selection_mode: 'rotate', retry_interval_seconds: 1, probe_interval_seconds: 6, proxies: [] }
const render = (historyOnly = false) => mount(CodexTicketCollectModal, { props: { show: true, accountIds: [1, 2, 1], historyOnly }, global: { stubs: {
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot/><slot name="footer"/></div>' }, Pagination: true
} } })
describe('CodexTicketCollectModal', () => {
  beforeEach(() => { vi.clearAllMocks(); admin.user.id = 1; settings.mockResolvedValue(config); runs.mockResolvedValue([]); detail.mockResolvedValue({ items: [], total: 0 }) })
  it('只使用网关配置并冻结账号，勾选前不发起采集，结果按点击展开', async () => {
    start.mockImplementation(async (_ids, _revision, _signal, callback) => {
      callback('start', { id: 'run1', total: 4, status: 'running', counts: {}, config })
      callback('attempt', { id: 1, account_id: 1, model: 'gpt-6-astra', status: 'ready', attempt: 1 })
      callback('result', { account_id: 1, model: 'gpt-6-astra', status: 'ready' })
      callback('complete', { id: 'run1', total: 4, status: 'completed', counts: { ready: 1, skipped: 3 }, config })
    })
    const w = render(); await flushPromises()
    expect(start).not.toHaveBeenCalled(); expect(w.get('[data-testid="ticket-collect-start"]').attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('332'); expect(w.text()).toContain('123'); expect(w.find('select').exists()).toBe(false)
    await w.setProps({ accountIds: [99] }); await w.get('[data-testid="ticket-collect-confirm"]').setValue(true)
    await w.get('[data-testid="ticket-collect-start"]').trigger('click'); await flushPromises()
    expect(start.mock.calls[0][0]).toEqual([1, 2]); expect(start.mock.calls[0][1]).toBe('r1')
    expect(w.get('details').attributes('open')).toBeUndefined(); expect(detail).not.toHaveBeenCalled()
    expect(w.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('4')
    const category = w.findAll('button').find(b => b.text().includes('ticketCollect.status.ready'))!
    await category.trigger('click'); await flushPromises(); expect(detail).toHaveBeenCalledWith('run1', expect.objectContaining({ status: 'ready', kind: 'result' }))
    w.unmount()
  })
  it('换管理员或关闭取消旧批次，不显示迟到日志', async () => {
    let callback!: (kind: string, data: unknown) => void
    let signal!: AbortSignal
    start.mockImplementation((_ids, _revision, input, handler) => { signal = input; callback = handler; return new Promise((_resolve, reject) => signal.addEventListener('abort', () => reject(new Error('cancelled')))) })
    const w = render(); await flushPromises(); await w.get('[data-testid="ticket-collect-confirm"]').setValue(true); await w.get('[data-testid="ticket-collect-start"]').trigger('click')
    admin.user.id = 2; await flushPromises(); expect(signal.aborted).toBe(true)
    callback('start', { id: 'old', counts: {}, total: 999, status: 'running' }); await flushPromises(); expect(w.text()).not.toContain('999')
    expect((w.get('[data-testid="ticket-collect-confirm"]').element as HTMLInputElement).checked).toBe(false); w.unmount()
  })
  it('历史入口只发 GET，不自动重新采集', async () => {
    runs.mockResolvedValue([{ id: 'past', config, total: 2, counts: { ready: 2 }, started_at: '2026-09-18T00:00:00Z', status: 'completed' }])
    const w = render(true); await flushPromises(); expect(runs).toHaveBeenCalledWith(1); expect(start).not.toHaveBeenCalled(); expect(w.text()).toContain('332 bytes'); w.unmount()
  })
})
