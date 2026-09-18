import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import CodexTicketCollectModal from '../CodexTicketCollectModal.vue'

const { settings, runs, detail, start, clearHistory, deleteRun, deleteEvent } = vi.hoisted(() => ({ settings: vi.fn(), runs: vi.fn(), detail: vi.fn(), start: vi.fn(), clearHistory: vi.fn(), deleteRun: vi.fn(), deleteEvent: vi.fn() }))
const admin = reactive({ user: { id: 1 } })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => admin }))
vi.mock('@/api/admin/codexTickets', () => ({ ticketCollectionAPI: { settings, runs, detail, clearHistory, deleteRun, deleteEvent }, runTicketCollection: start }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (s: string) => s }) }))
const config = { enabled: true, target_length: 332, revision: 'r1', max_attempts: 123, selection_mode: 'rotate', retry_interval_seconds: 1, probe_interval_seconds: 6, proxies: [] }
const render = (historyOnly = false) => mount(CodexTicketCollectModal, { props: { show: true, accountIds: [1, 2, 1], historyOnly }, global: { stubs: {
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot/><slot name="footer"/></div>' }, Pagination: true
} } })
describe('CodexTicketCollectModal', () => {
  beforeEach(() => { vi.clearAllMocks(); admin.user.id = 1; settings.mockResolvedValue(config); runs.mockResolvedValue([]); detail.mockResolvedValue({ run: { status: 'completed' }, items: [], total: 0 }); clearHistory.mockResolvedValue({ deleted: 2 }); deleteRun.mockResolvedValue({ deleted: 1 }); deleteEvent.mockResolvedValue({ deleted: 1 }) })
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
    expect(w.emitted('result')).toHaveLength(1)
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
  it('清空与单批删除点击即执行，不出现重复确认，活动批次禁用', async () => {
    const past = { id: 'past', config, total: 2, counts: {}, started_at: '2026-09-18T00:00:00Z', status: 'completed' }
    runs.mockResolvedValue([past, { ...past, id: 'active', status: 'running' }])
    const w = render(true); await flushPromises()
    expect(deleteRun).not.toHaveBeenCalled(); expect(clearHistory).not.toHaveBeenCalled()
    expect(w.findAll('[data-testid="ticket-history-delete"]')[1].attributes('disabled')).toBeDefined()
    await w.findAll('[data-testid="ticket-history-delete"]')[0].trigger('click'); await flushPromises()
    expect(deleteRun).toHaveBeenCalledOnce(); expect(deleteRun).toHaveBeenCalledWith('past')
    await w.get('[data-testid="ticket-history-clear"]').trigger('click'); await flushPromises()
    expect(clearHistory).toHaveBeenCalledOnce(); expect(start).not.toHaveBeenCalled(); w.unmount()
  })
  it('单事件删除只发指定批次和ID，失败保留显示并报错', async () => {
    runs.mockResolvedValue([{ id: 'past', config, total: 2, counts: {}, started_at: '2026-09-18T00:00:00Z', status: 'completed' }])
    detail.mockResolvedValue({ run: { status: 'completed' }, total: 1, items: [{ id: 8, account_id: 1, model: 'gpt-6-astra', status: 'failed', target_length: 332, started_at: '2026-09-18T00:00:00Z' }] })
    deleteEvent.mockRejectedValueOnce(new Error('Delete failed'))
    const w = render(true); await flushPromises()
    await w.findAll('button').find(b => b.text().includes('332 bytes'))!.trigger('click'); await flushPromises()
    await w.get('[data-testid="ticket-event-delete"]').trigger('click'); await flushPromises()
    expect(deleteEvent).toHaveBeenCalledWith('past', 8); expect(w.text()).toContain('Delete failed'); expect(w.findAll('article')).toHaveLength(1); w.unmount()
  })
  it('历史比值复用左右重点色，无头不伪造零长度', async () => {
    runs.mockResolvedValue([{ id: 'past', config, total: 2, counts: {}, started_at: '2026-09-18T00:00:00Z', status: 'completed' }])
    const event = { id: 8, account_id: 1, model: 'gpt-6-astra', status: 'missing', target_length: 292, started_at: '2026-09-18T00:00:00Z', diagnostic: { http_status: 200, header_present: true, header_length: 356, degraded_signal: true } }
    detail.mockResolvedValue({ run: { status: 'completed' }, total: 2, items: [event, { ...event, id: 9, diagnostic: { ...event.diagnostic, header_present: false, header_length: 0, degraded_signal: false } }] })
    const w = render(true); await flushPromises()
    await w.findAll('button').find(b => b.text().includes('332 bytes'))!.trigger('click'); await flushPromises()
    expect(w.findAll('[data-testid="ticket-length-ratio"]')).toHaveLength(1)
    expect(w.get('[data-testid="ticket-length-actual"]').classes()).toContain('text-bh-red')
    expect(w.get('[data-testid="ticket-length-target"]').classes()).toContain('text-emerald-700')
    expect(w.text()).toContain('tickets.noHeader')
    w.unmount()
  })
})
