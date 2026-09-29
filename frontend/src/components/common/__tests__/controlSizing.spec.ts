import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import Pagination from '../Pagination.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const testDirectory = dirname(fileURLToPath(import.meta.url))
const readSource = (relativePath: string) => readFileSync(resolve(testDirectory, relativePath), 'utf8')

const SelectStub = defineComponent({
  name: 'PaginationSelectStub',
  props: ['modelValue', 'options'],
  setup(props) {
    // 真实 Select 的 36px 由 .input 基线(min-h-9)提供,stub 以 h-9 顶替这一高度契约。
    return () => h('button', { class: 'select-trigger h-9' }, String(props.modelValue))
  }
})

const IconStub = defineComponent({
  name: 'Icon',
  setup() {
    return () => h('span')
  }
})

describe('Bauhaus control sizing', () => {
  it('uses the Bauhaus baseline in shared controls while preserving compact page controls', () => {
    const globalStyle = readSource('../../../style.css')
    const selectSource = readSource('../Select.vue')
    const proxySelectorSource = readSource('../ProxySelector.vue')
    const dateRangePickerSource = readSource('../DateRangePicker.vue')
    const paginationSource = readSource('../Pagination.vue')

    // 当前包豪斯表单采用 44px 触控基线，分页继续保留 36px 紧凑高度。
    expect(globalStyle).toContain('@apply rounded-none px-4 py-2.5 text-sm font-bold;')
    expect(globalStyle).toContain('@apply min-h-11;')
    expect(globalStyle).toContain('@apply inline-flex h-11 w-11 items-center justify-center rounded-control p-0;')
    expect(globalStyle).toContain('@apply w-full rounded-none px-4 py-2.5 text-sm font-medium;')
    expect(globalStyle).toContain('@apply flex min-h-11 items-center gap-3 rounded-control py-2.5;')
    // 三个下拉触发器以模板组合 input input-trigger 共享 36px 基线,不再各自复制配方。
    expect(selectSource).toContain("'input input-trigger'")
    expect(proxySelectorSource).toContain("'input input-trigger'")
    expect(dateRangePickerSource).toContain("'input input-trigger'")
    for (const source of [selectSource, proxySelectorSource, dateRangePickerSource]) {
      expect(source).not.toContain('@apply h-9 min-h-9 rounded-control px-4 py-1.5 text-sm;')
    }
    // 弹层内的确认按钮直接使用共享按钮配方。
    expect(dateRangePickerSource).toContain('class="btn btn-primary"')
    expect(dateRangePickerSource).not.toContain('.date-picker-apply')
    // 分页控件直接由模板里的 h-9 提供 36px 基线,不再有局部高度覆盖。
    expect(paginationSource).toContain('pagination-jump-button btn btn-ghost btn-sm h-9')
    expect(paginationSource).not.toContain('--pagination-control-height')
  })

  it('renders every pagination button with the shared hard-edge control classes', () => {
    const wrapper = mount(Pagination, {
      props: {
        total: 30,
        page: 2,
        pageSize: 10
      },
      global: {
        stubs: {
          Select: SelectStub,
          Icon: IconStub
        }
      }
    })

    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBe(8)
    const paginationButtons = buttons.filter((button) => !button.classes().includes('select-trigger'))
    expect(paginationButtons.every((button) => button.classes().includes('pagination-control'))).toBe(true)
    expect(paginationButtons.every((button) => button.classes().includes('bh-page-btn'))).toBe(true)
  })

  it('keeps the latest page-specific sizing fixes explicit', () => {
    const providerBulkActionsSource = readSource('../../admin/provider/ProviderBulkActionsBar.vue')
    const userUsageSource = readSource('../../../views/user/UsageView.vue')
    const adminUsageSource = readSource('../../../views/admin/UsageView.vue')
    const riskControlSource = readSource('../../../views/admin/RiskControlView.vue')
    const settingsSource = readSource('../../../views/admin/SettingsView.vue')
    const emailTemplateSource = readSource('../../../views/admin/settings/EmailTemplateEditor.vue')
    const backupSource = readSource('../../../views/admin/BackupView.vue')
    const providerListSource = readSource('../../payment/PaymentProviderList.vue')
    const adminOrdersSource = readSource('../../../views/admin/orders/AdminOrdersView.vue')
    const adminPaymentPlansSource = readSource('../../../views/admin/orders/AdminPaymentPlansView.vue')

    expect(providerBulkActionsSource).toContain('class="btn btn-primary btn-sm h-[30px]"')
    expect(userUsageSource).toContain('<div class="card p-4">')
    expect(adminUsageSource).toContain('<div class="card p-4">')
    expect(riskControlSource).toContain('class="grid grid-cols-1 items-start gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,440px)]"')
    expect(riskControlSource).toContain(":class=\"apiKeyRowsExpanded ? 'overflow-visible' : ''\"")
    expect(settingsSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(settingsSource).toContain('class="btn btn-secondary btn-sm h-9 w-fit"')
    expect(emailTemplateSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(backupSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(providerListSource).toContain('class="btn btn-secondary btn-icon"')
    expect(adminOrdersSource).toContain('<TablePageLayout>')
    expect(adminOrdersSource).toContain('<template #table>')
    expect(adminPaymentPlansSource).toContain('<TablePageLayout>')
    expect(adminPaymentPlansSource).toContain('<template #table>')
  })
})
