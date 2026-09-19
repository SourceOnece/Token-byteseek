import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodexTicketAccountSettings from '../CodexTicketAccountSettings.vue'
import Select from '@/components/common/Select.vue'

const { get, update, testProxy } = vi.hoisted(() => ({ get: vi.fn(), update: vi.fn(), testProxy: vi.fn() }))
vi.mock('@/api/admin/codexTickets', () => ({ ticketAccountAPI: { get, update }, testTicketProxy: testProxy }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const rules = () => ({ models: ['gpt-6-astra', 'gpt-5.6-sol'], target_length: 332, degraded_signal_length: 312, max_attempts: 3, concurrency: 4, cache_minutes: 60, refresh_before_minutes: 10, retry_interval_seconds: 1, probe_interval_seconds: 6, failure_threshold: 0, cooldown_seconds: 300 })
const account = () => ({ rules: rules(), proxy_policy: { mode: 'fixed', dynamic_source: 'template', proxy_protocol: 'http', extraction_configured: false, fixed_proxy_id: 'account', proxies: [{ id: 'account', name: 'A', configured: true }] }, account_id: 1, mode: 'inherit', watchdog_mode: 'inherit', effective_watchdog_mode: 'observe', effective_enabled: true, proxy_source: 'account', proxy_configured: true, revision: 'r1' })

describe('CodexTicketAccountSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); get.mockResolvedValue(account()); update.mockResolvedValue([{ ...account(), revision: 'r2' }]) })
  it('单号读取脱敏配置，不自动保存或清空已有代理', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1] } }); await flushPromises()
    expect(get).toHaveBeenCalledWith(1, expect.any(AbortSignal)); expect(update).not.toHaveBeenCalled()
    expect(w.find('select').exists()).toBe(false)
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update).toHaveBeenCalledWith([1], { mode: 'on', watchdog_mode: 'observe', rules: rules() }, 'r1')
    expect(w.emitted('saved')).toHaveLength(1); w.unmount()
  })
  it('批量全部默认未勾选，只提交选中项，保存后取消勾选', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1, 2], bulk: true } }); await flushPromises()
    expect(get).not.toHaveBeenCalled(); expect(w.get('[data-testid="ticket-account-save"]').attributes('disabled')).toBeDefined()
    await w.get('[data-testid="ticket-edit-mode"]').setValue(true)
    w.findAllComponents(Select)[0].vm.$emit('update:modelValue', 'off'); await flushPromises()
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update).toHaveBeenCalledWith([1, 2], { mode: 'off' }, undefined)
    expect(w.get('[data-testid="ticket-account-save"]').attributes('disabled')).toBeDefined(); w.unmount()
  })
  it('恢复继承必须明确选中，不把密码空白当作清除', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1, 2], bulk: true } }); await flushPromises()
    await w.get('[data-testid="ticket-edit-proxy"]').setValue(true)
    w.findAllComponents(Select)[2].vm.$emit('update:modelValue', 'fixed'); await flushPromises()
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises(); expect(update).not.toHaveBeenCalled()
    w.findAllComponents(Select)[2].vm.$emit('update:modelValue', 'inherit'); await flushPromises()
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update).toHaveBeenCalledWith([1, 2], { proxy_policy: { mode: 'inherit' } }, undefined); w.unmount()
  })
  it('代理保存后清空输入，切换账号不保留密码；加载失败禁止覆盖', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1] } }); await flushPromises()
    await w.get('[data-testid="ticket-edit-proxy"]').setValue(true)
    await w.get('[data-testid="ticket-proxy-url"]').setValue('http://user:synthetic@proxy.invalid:8080')
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect((w.get('[data-testid="ticket-proxy-url"]').element as HTMLInputElement).value).toBe('')
    get.mockRejectedValueOnce(new Error('offline')); await w.setProps({ ids: [2] }); await flushPromises()
    expect(w.get('[data-testid="ticket-account-save"]').attributes('disabled')).toBeDefined(); w.unmount()
  })
  it('账号支持0不限和独立缓存规则，不提交未勾选的批量项目', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1] } }); await flushPromises()
    await w.get('[data-testid="ticket-rule-max_attempts"]').setValue('0')
    await w.get('[data-testid="ticket-rule-cache_minutes"]').setValue('90')
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update.mock.calls[0][1].rules).toMatchObject({ max_attempts: 0, cache_minutes: 90, target_length: 332 })
    w.unmount()
  })
})
