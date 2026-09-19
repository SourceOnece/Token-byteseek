import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodexTicketAccountSettings from '../CodexTicketAccountSettings.vue'
import Select from '@/components/common/Select.vue'

const { get, update } = vi.hoisted(() => ({ get: vi.fn(), update: vi.fn() }))
vi.mock('@/api/admin/codexTickets', () => ({ ticketAccountAPI: { get, update } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const account = () => ({ account_id: 1, mode: 'inherit', watchdog_mode: 'inherit', effective_watchdog_mode: 'observe', effective_enabled: true, proxy_source: 'account', proxy_configured: true, revision: 'r1' })

describe('CodexTicketAccountSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); get.mockResolvedValue(account()); update.mockResolvedValue([{ ...account(), revision: 'r2' }]) })
  it('单号读取脱敏配置，不自动保存或清空已有代理', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1] } }); await flushPromises()
    expect(get).toHaveBeenCalledWith(1, expect.any(AbortSignal)); expect(update).not.toHaveBeenCalled()
    expect(w.find('select').exists()).toBe(false)
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update).toHaveBeenCalledWith([1], { mode: 'inherit', watchdog_mode: 'inherit' }, 'r1')
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
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises(); expect(update).not.toHaveBeenCalled()
    w.findAllComponents(Select)[2].vm.$emit('update:modelValue', 'inherit'); await flushPromises()
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect(update).toHaveBeenCalledWith([1, 2], { harvest_proxy_url: '' }, undefined); w.unmount()
  })
  it('代理保存后清空输入，切换账号不保留密码；加载失败禁止覆盖', async () => {
    const w = mount(CodexTicketAccountSettings, { props: { ids: [1] } }); await flushPromises()
    await w.get('[data-testid="ticket-edit-proxy"]').setValue(true)
    await w.get('[data-testid="ticket-account-proxy"]').setValue('http://user:synthetic@proxy.invalid:8080')
    await w.get('[data-testid="ticket-account-save"]').trigger('click'); await flushPromises()
    expect((w.get('[data-testid="ticket-account-proxy"]').element as HTMLInputElement).value).toBe('')
    get.mockRejectedValueOnce(new Error('offline')); await w.setProps({ ids: [2] }); await flushPromises()
    expect(w.get('[data-testid="ticket-account-save"]').attributes('disabled')).toBeDefined(); w.unmount()
  })
})
