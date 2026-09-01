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

    // 全局控件使用 44px 硬边基线，显式 h-8/h-9/h-10 的紧凑工具栏仍由页面保留。
    expect(globalStyle).toContain('@apply rounded-none px-4 py-2.5 text-sm font-bold;')
    expect(globalStyle).toContain('@apply min-h-11;')
    expect(globalStyle).toContain('@apply rounded-none p-2.5;')
    expect(globalStyle).toContain('@apply w-full rounded-none px-4 py-2.5 text-sm font-medium;')
    expect(globalStyle).toContain('@apply flex items-center gap-3 rounded-none py-2.5;')
    expect(selectSource).toContain('@apply rounded-none px-4 py-2.5 text-sm font-semibold;')
    expect(proxySelectorSource).toContain('@apply h-9 min-h-9 rounded-control px-4 py-1.5 text-sm;')
    expect(dateRangePickerSource).toContain('@apply h-9 min-h-9 rounded-control px-4 py-1.5 text-sm;')
    expect(dateRangePickerSource).toContain('@apply inline-flex h-9 min-h-9 items-center justify-center rounded-control px-4 py-1.5 text-sm font-medium;')
    expect(paginationSource).toContain('height: 2.25rem;')
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
    const accountBulkActionsSource = readSource('../../admin/account/AccountBulkActionsBar.vue')
    const userUsageSource = readSource('../../../views/user/UsageView.vue')
    const adminUsageSource = readSource('../../../views/admin/UsageView.vue')
    const riskControlSource = readSource('../../../views/admin/RiskControlView.vue')
    const settingsSource = readSource('../../../views/admin/SettingsView.vue')
    const emailTemplateSource = readSource('../../../views/admin/settings/EmailTemplateEditor.vue')
    const backupSource = readSource('../../../views/admin/BackupView.vue')
    const providerListSource = readSource('../../payment/PaymentProviderList.vue')
    const adminOrdersSource = readSource('../../../views/admin/orders/AdminOrdersView.vue')
    const adminPaymentPlansSource = readSource('../../../views/admin/orders/AdminPaymentPlansView.vue')

    expect(accountBulkActionsSource).toContain('class="btn btn-primary btn-sm h-[30px]"')
    expect(userUsageSource).toContain('<div class="card p-4">')
    expect(adminUsageSource).toContain('<div class="card p-4">')
    expect(riskControlSource).toContain('class="grid grid-cols-1 items-start gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,440px)]"')
    expect(riskControlSource).toContain(":class=\"apiKeyRowsExpanded ? 'overflow-visible' : ''\"")
    expect(settingsSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(settingsSource).toContain('class="btn btn-secondary btn-sm h-9 w-fit"')
    expect(emailTemplateSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(backupSource).toContain('class="btn btn-primary btn-sm h-9"')
    expect(providerListSource).toContain('class="btn btn-secondary btn-sm h-9 w-9 p-0"')
    expect(adminOrdersSource).toContain('<TablePageLayout>')
    expect(adminOrdersSource).toContain('<template #table>')
    expect(adminPaymentPlansSource).toContain('<TablePageLayout>')
    expect(adminPaymentPlansSource).toContain('<template #table>')
  })
})
