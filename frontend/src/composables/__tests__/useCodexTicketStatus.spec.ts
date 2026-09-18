import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { useCodexTicketStatus } from '../useCodexTicketStatus'
import type { Account } from '@/types'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/admin/codexTickets', () => ({ getCodexTicketStatus: get }))
const account = (id: number, type = 'oauth') => ({ id, type, platform: 'openai', updated_at: String(id), credentials: {} }) as Account
const response = (id: number) => ({ enabled: true, server_time: '2026-09-18T00:00:00Z', items: [{ account_id: id, eligible: true, collection_paused: false, models: [] }] })
let handles: ReturnType<typeof useCodexTicketStatus>
const accounts = ref<Account[]>([])
const mountHost = () => mount(defineComponent({ setup() { handles = useCodexTicketStatus(accounts); return () => null } }))

describe('useCodexTicketStatus', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
    accounts.value = [account(1), account(2, 'apikey')]
    get.mockResolvedValue(response(1))
  })
  afterEach(() => { vi.useRealTimers() })

  it('只查询当前页 OAuth，每 15 秒一批；隐藏或销毁时不继续轮询', async () => {
    const w = mountHost(); await flushPromises()
    expect(get.mock.calls[0][0]).toEqual([1])
    expect(handles.ticketNow.value).toBe(Date.parse(response(1).server_time))
    await vi.advanceTimersByTimeAsync(15000)
    expect(get).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    await vi.advanceTimersByTimeAsync(30000)
    expect(get).toHaveBeenCalledTimes(2)
    w.unmount()
    await vi.advanceTimersByTimeAsync(30000)
    expect(get).toHaveBeenCalledTimes(2)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('翻页取消旧请求，延迟返回不能污染新页；失败不冒充无票', async () => {
    let resolveOld!: (data: unknown) => void
    get.mockImplementationOnce(() => new Promise(r => { resolveOld = r }))
    const w = mountHost(); await flushPromises()
    const signal = get.mock.calls[0][1] as AbortSignal
    get.mockResolvedValueOnce(response(3))
    accounts.value = [account(3)]; await flushPromises()
    expect(signal.aborted).toBe(true)
    resolveOld(response(1)); await flushPromises()
    expect(Object.keys(handles.ticketStatus.value)).toEqual(['3'])
    get.mockRejectedValueOnce(new Error('cache offline'))
    await vi.advanceTimersByTimeAsync(15000)
    expect(handles.ticketLoadFailed.value).toBe(true)
    expect(Object.keys(handles.ticketStatus.value)).toEqual(['3'])
    w.unmount()
  })

  it('超过 100 个账号按顺序分批，不为不支持账号请求', async () => {
    accounts.value = Array.from({ length: 101 }, (_, i) => account(i + 1))
    const w = mountHost(); await flushPromises()
    expect(get).toHaveBeenCalledTimes(2)
    expect(get.mock.calls.map(c => c[0].length)).toEqual([100, 1])
    accounts.value = [account(2, 'apikey')]; await flushPromises()
    expect(handles.ticketStatus.value).toEqual({})
    expect(get).toHaveBeenCalledTimes(2)
    w.unmount()
  })
})
