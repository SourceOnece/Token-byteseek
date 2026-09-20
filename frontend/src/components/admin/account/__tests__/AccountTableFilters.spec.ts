import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import AccountTableFilters from '../AccountTableFilters.vue'
import type { AdminGroup } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: `
    <div data-test="select-stub" :data-model-value="String(modelValue ?? '')">
      <!-- 测试只暴露项目 Select 的 options，不使用浏览器原生选择框行为。 -->
      <span v-for="option in options" :key="String(option.value)" data-test="select-option">
        {{ option.label }}
      </span>
    </div>
  `
}

function group(overrides: Partial<AdminGroup>): AdminGroup {
  return {
    id: 1,
    name: 'group',
    description: null,
    platform: 'openai',
    rate_multiplier: 1,
    is_exclusive: false,
    session_isolation_enabled: false,
    status: 'active',
    allow_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    unavailable_fallback_group_id: null,
    require_oauth_only: false,
    require_privacy_set: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    model_routing: null,
    model_routing_enabled: false,
    mcp_xml_inject: false,
    sort_order: 0,
    ...overrides
  }
}

describe('AccountTableFilters', () => {
  afterEach(() => vi.useRealTimers())
  it('实际长度留空起步，防抖自动筛选，清空取消，不再展示合格长度预设', async () => {
    vi.useFakeTimers()
    const wrapper=mount(AccountTableFilters,{props:{searchQuery:'',filters:{ticket_filter:'on'}},global:{stubs:{Select:SelectStub,SearchInput:true}}})
    await wrapper.get('[data-testid="account-filters-toggle"]').trigger('click')
    const input=wrapper.get('[data-testid="ticket-actual-length-filter"]')
    expect((input.element as HTMLInputElement).value).toBe('')
    const options=wrapper.findAllComponents(SelectStub).find(s=>s.attributes('data-testid')==='ticket-type-filter')!.props('options') as {value:string}[]
    expect(options.some(o=>o.value.startsWith('length:'))).toBe(false)
    await input.setValue('356')
    expect(wrapper.emitted('update:filters')).toBeUndefined()
    await vi.advanceTimersByTimeAsync(449)
    expect(wrapper.emitted('update:filters')).toBeUndefined()
    await vi.advanceTimersByTimeAsync(1)
    expect(wrapper.emitted('update:filters')?.at(-1)).toEqual([{ticket_filter:'on,actual_length:356'}])
    await wrapper.setProps({filters:{ticket_filter:'on,actual_length:356'}})
    await input.setValue('3');await vi.advanceTimersByTimeAsync(500)
    expect(wrapper.emitted('update:filters')).toHaveLength(1)
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    await input.setValue('');await vi.advanceTimersByTimeAsync(500)
    expect(wrapper.emitted('update:filters')?.at(-1)).toEqual([{ticket_filter:'on'}])
    wrapper.unmount()
  })
  it('卸载取消待触发筛选，旧目标长度不能冒充实际长度', async () => {
    vi.useFakeTimers()
    const updates=vi.fn()
    const wrapper=mount(AccountTableFilters,{props:{searchQuery:'',filters:{ticket_filter:'length:356'},'onUpdate:filters':updates},global:{stubs:{Select:SelectStub,SearchInput:true}}})
    expect(wrapper.emitted('update:filters')?.[0]).toEqual([{ticket_filter:''}])
    await wrapper.setProps({filters:{ticket_filter:''}})
    await wrapper.get('[data-testid="account-filters-toggle"]').trigger('click')
    await wrapper.get('[data-testid="ticket-actual-length-filter"]').setValue('332')
    wrapper.unmount();await vi.advanceTimersByTimeAsync(500)
    expect(updates).toHaveBeenCalledOnce()
  })
  it('检测筛选保留为独立字段，重置可清空', async () => {
    const wrapper = mount(AccountTableFilters, { props: { searchQuery: '', filters: { quality_status: 'full' } }, global: { stubs: { Select: SelectStub, SearchInput: true } } })
    await wrapper.get('[data-testid="account-filters-toggle"]').trigger('click')
    expect(wrapper.text()).toContain('admin.accounts.quality.status.failed')
    const reset = wrapper.findAll('button').find(button => button.text() === 'common.reset')!
    await reset.trigger('click')
    expect(wrapper.emitted('update:filters')?.[0]).toEqual([{ platform: '', type: '', status: '', privacy_mode: '', group: '', quality_status: '', ticket_filter: '' }])
  })
  it('keeps inactive groups visible in the group filter', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          status: '',
          privacy_mode: '',
          group: ''
        },
        groups: [
          group({ id: 10, name: 'Active Pool', status: 'active' }),
          group({ id: 11, name: 'Disabled Pool', status: 'inactive' })
        ]
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: true
        }
      }
    })

    await wrapper.get('[data-testid="account-filters-toggle"]').trigger('click')

    const selectComponents = wrapper.findAllComponents(SelectStub)
    const groupOptions = selectComponents.find(select => select.props('options').some((item: { value: string }) => item.value === '10'))?.props('options') as Array<{ value: string; label: string }>

    expect(groupOptions).toEqual(expect.arrayContaining([
      { value: '10', label: 'Active Pool' },
      { value: '11', label: 'Disabled Pool (common.inactive)' }
    ]))
  })
})
