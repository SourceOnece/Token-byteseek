import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import CodexQualitySchedulesModal from '../CodexQualitySchedulesModal.vue'

const { list, save, accounts } = vi.hoisted(() => ({ list: vi.fn(), save: vi.fn(), accounts: vi.fn() }))
vi.mock('@/api/admin/codexQuality', () => ({ qualitySchedulesAPI: { list, save, runs: vi.fn().mockResolvedValue([]) } }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list: accounts } } }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))

describe('Scheduled quality tests', () => {
  beforeEach(() => { list.mockResolvedValue([]); save.mockResolvedValue({ id: 1 }); accounts.mockResolvedValue({ items: [{ id: 1, platform: 'openai', type: 'oauth', name: 'one' }, { id: 2, platform: 'openai', type: 'oauth', parent_account_id: 1 }], total: 2 }) })
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
    expect(save.mock.calls[0][0]).toMatchObject({ interval_minutes: 60, keep_runs: 30, config: { account_ids: [1], timeout_seconds: 300, confirm_scheduling: true } })
    wrapper.unmount()
  })
})
