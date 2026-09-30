import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ModelAttributesView from '../ModelAttributesView.vue'
import { modelAttributesAPI } from '@/api/admin/modelAttributes'

vi.mock('@/api/admin/modelAttributes', () => ({ modelAttributesAPI: { list: vi.fn(), defaults: vi.fn(), update: vi.fn(), save: vi.fn(), remove: vi.fn() } }))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: { getAll: vi.fn().mockResolvedValue([{ id: 7, name: 'Group' }]) } } }))
vi.mock('vue-i18n', async () => ({ ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
  DataTable: { props: ['data'], template: '<div><div v-for="row in data" :key="row.id || row.model"><span>{{ row.name || row.model }}</span><slot name="cell-actions" :row="row" /></div></div>' },
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  RuleListEditor: { props: ['items'], template: '<div><div v-for="(item, index) in items" :key="index"><slot name="row" :item="item" :index="index" /></div></div>' },
  Icon: true, Select: true, Toggle: true, Pagination: true, ConfirmDialog: true,
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(modelAttributesAPI.list).mockResolvedValue({ items: [{ id: 1, name: 'Profile', description: '', status: 'active', group_ids: [7], rules: [{ models: ['upstream'], attributes: { tool_call: false } }] }], total: 1 })
  vi.mocked(modelAttributesAPI.defaults).mockResolvedValue({ items: [{ model: 'default-keep', provider: 'original', source: 'models.dev', attributes: {} }], total: 1, providers: ['original'], version: 'abc123', last_updated: '2026-09-30T00:00:00Z' })
})

describe('属性管理页面', () => {
  it('默认属性页只读查询，更新失败保留现有目录', async () => {
    const wrapper = mount(ModelAttributesView, { global: { stubs } })
    await flushPromises()
    await wrapper.findAll('[role="tab"]')[1]!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('default-keep')
    expect(modelAttributesAPI.update).not.toHaveBeenCalled()
    vi.mocked(modelAttributesAPI.update).mockRejectedValueOnce(new Error('offline'))
    await wrapper.findAll('button').find(button => button.text() === 'admin.pricing.defaults.update')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('default-keep')
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    wrapper.unmount()
  })
  it('编辑回显与保存保留显式 false 及分组关联', async () => {
    const wrapper = mount(ModelAttributesView, { global: { stubs } })
    await flushPromises()
    await wrapper.find('button[aria-label="common.edit"]').trigger('click')
    await flushPromises()
    await wrapper.find('#attribute-form').trigger('submit')
    await flushPromises()
    expect(modelAttributesAPI.save).toHaveBeenCalledWith(expect.objectContaining({ id: 1, group_ids: [7], rules: [{ models: ['upstream'], attributes: { tool_call: false } }] }))
    wrapper.unmount()
  })
})
