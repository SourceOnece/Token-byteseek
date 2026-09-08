import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import CodexQualitySchedulesModal from '../CodexQualitySchedulesModal.vue'

const { list, save, accounts, trigger, setEnabled } = vi.hoisted(() => ({ list: vi.fn(), save: vi.fn(), accounts: vi.fn(), trigger: vi.fn(), setEnabled: vi.fn() }))
vi.mock('@/api/admin/codexQuality', () => ({ qualitySchedulesAPI: { list, save, trigger, setEnabled, runs: vi.fn().mockResolvedValue([]) } }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list: accounts } } }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))

describe('Scheduled quality tests', () => {
  beforeEach(() => { vi.clearAllMocks(); list.mockResolvedValue([]); save.mockResolvedValue({ id: 1 }); accounts.mockResolvedValue({ items: [{ id: 1, platform: 'openai', type: 'oauth', name: 'one' }, { id: 2, platform: 'openai', type: 'oauth', parent_account_id: 1 }, { id: 3, platform: 'openai', type: 'apikey', name: 'API upstream' }], total: 3 }) })
  it('全选排除影子，默认120秒并要求明确确认', async () => {
    const wrapper = mount(CodexQualitySchedulesModal, {
      props: { show: false, accountIds: [] }, global: { stubs: { BaseDialog: { template: '<div><slot/></div>' }, Select: true, Pagination: true } }
    })
    await wrapper.setProps({ show: true }); await flushPromises()
    await wrapper.get('[data-testid="quality-plan-new"]').trigger('click'); await flushPromises()
    expect(wrapper.get('#quality-plan-timeout').element).toHaveProperty('value', '120')
    await wrapper.get('[data-testid="quality-plan-select-all"]').trigger('click'); await flushPromises()
    await wrapper.get('#quality-plan-name').setValue('计划')
    await wrapper.get('#quality-plan-prompt').setValue('题目')
    await wrapper.get('#quality-plan-keyword').setValue('PASS')
    await wrapper.get('#quality-plan-timeout').setValue('300')
    expect(wrapper.get('[data-testid="quality-plan-save"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="quality-plan-confirm"]').setValue(true)
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(save.mock.calls[0][0]).toMatchObject({ interval_minutes: 60, keep_runs: 30, config: { account_ids: [1, 3], timeout_seconds: 300, api_protocol: 'responses', confirm_scheduling: true } })
    wrapper.unmount()
  })
  it('关闭周期仍能单次检测，外置开关可启停，运行中保留停止入口', async () => {
    const plan = { id: 7, name: 'Plan', enabled: false, interval_minutes: 60, config: { account_ids: [3], model: 'gpt-6-astra', reasoning_effort: 'high' }, active_run_id: null as number | null }
    list.mockImplementation(async () => [{ ...plan }])
    setEnabled.mockImplementation(async (_id, value) => { plan.enabled = value })
    trigger.mockResolvedValue(undefined)
    const wrapper = mount(CodexQualitySchedulesModal, { props: { show: false, accountIds: [] }, global: { stubs: { BaseDialog: { template: '<div><slot/></div>' }, Select: true, Pagination: true } } })
    await wrapper.setProps({ show: true }); await flushPromises()
    const card = wrapper.get('[data-testid="quality-plan-7"]')
    expect(card.get('[data-testid="quality-plan-run"]').attributes('disabled')).toBeUndefined()
    await card.get('[data-testid="quality-plan-run"]').trigger('click'); await flushPromises()
    expect(trigger).toHaveBeenCalledTimes(1)
    expect(trigger).toHaveBeenCalledWith(7)
    expect(setEnabled).not.toHaveBeenCalled()
    await card.get('[role="switch"]').trigger('click'); await flushPromises()
    expect(setEnabled).toHaveBeenLastCalledWith(7, true)
    expect(card.get('[role="switch"]').attributes('aria-checked')).toBe('true')
    plan.active_run_id = 15
    await card.get('[data-testid="quality-plan-run"]').trigger('click'); await flushPromises()
    expect(card.get('[data-testid="quality-plan-run"]').attributes('disabled')).toBeDefined()
    expect(card.get('[role="switch"]').attributes('disabled')).toBeUndefined()
    await card.get('[role="switch"]').trigger('click'); await flushPromises()
    expect(setEnabled).toHaveBeenLastCalledWith(7, false)
    wrapper.unmount()
  })
})
