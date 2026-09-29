import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpenAIOAuthImportDefaultsSettings from '../OpenAIOAuthImportDefaultsSettings.vue'
import { ticketSettingsFixture } from './ticketSettingsFixture'

const mocks = vi.hoisted(() => ({ get: vi.fn(), update: vi.fn(), defaults: vi.fn(), updateTickets: vi.fn(), success: vi.fn(), error: vi.fn() }))
vi.mock('@/api', () => ({ adminAPI: {
  settings: { getOpenAIOAuthImportDefaults: mocks.get, updateOpenAIOAuthImportDefaults: mocks.update },
  tlsFingerprintProfiles: { list: vi.fn().mockResolvedValue([]) },
  tlsFingerprintRouters: { list: vi.fn().mockResolvedValue([]) }
} }))
vi.mock('@/api/admin/codexTickets', () => ({ ticketAccountAPI: { defaults: mocks.defaults, updateDefaults: mocks.updateTickets }, testTicketProxy: vi.fn() }))
vi.mock('@/api/admin/proxies', () => ({ getAll: vi.fn().mockResolvedValue([]) }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess: mocks.success, showError: mocks.error }) }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))

describe('OpenAIOAuthImportDefaultsSettings unified ticket save', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.get.mockResolvedValue({ account: {}, credentials: {}, extra: {} })
    mocks.update.mockResolvedValue({ account: {}, credentials: {}, extra: {} })
    mocks.defaults.mockResolvedValue(ticketSettingsFixture(0))
    mocks.updateTickets.mockResolvedValue({ ...ticketSettingsFixture(0), revision: 'r2' })
  })

  // 保留真实票据组件，只替换与本次保存无关的大型子表单。
  const create = () => mount(OpenAIOAuthImportDefaultsSettings, { global: { stubs: { ModelWhitelistSelector: true, CodexImageToolModeSelector: true } } })
  const save = async (w: ReturnType<typeof create>) => {
    await w.findAll('button').find(button => button.text() === 'common.save')!.trigger('click')
    await flushPromises()
  }

  it('模板与普通默认值共用保存按钮，未修改票据不重复写模板', async () => {
    const w = create()
    await flushPromises()
    expect(w.find('[data-testid="ticket-account-save"]').exists()).toBe(false)
    expect(w.get('[data-testid="import-defaults-model-grid"]').classes()).toContain('lg:grid-cols-2')
    expect(w.get('[data-testid="import-defaults-model-grid"]').findAll('section')).toHaveLength(2)
    expect(w.get('[data-testid="import-defaults-models"]').text()).toContain('admin.accounts.modelWhitelist')
    expect(w.get('[data-testid="import-defaults-mappings"]').text()).toContain('admin.accounts.modelMapping')
    await save(w)
    expect(mocks.update).toHaveBeenCalledTimes(1)
    expect(mocks.updateTickets).not.toHaveBeenCalled()
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('356')
    await save(w)
    expect(mocks.updateTickets).toHaveBeenCalledWith({ rules: { ...ticketSettingsFixture().rules, target_length: 356 } }, 'r1')
    await save(w)
    expect(mocks.updateTickets).toHaveBeenCalledTimes(1)
    w.unmount()
  })

  it('模板票据失败显示部分成功，不冒充保存完成；重试仍带旧代次', async () => {
    mocks.updateTickets.mockRejectedValueOnce(new Error('revision conflict'))
    const w = create()
    await flushPromises()
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('356')
    await save(w)
    expect(mocks.update).toHaveBeenCalledTimes(1)
    expect(mocks.success).not.toHaveBeenCalled()
    expect(mocks.error).toHaveBeenCalledWith('admin.accounts.ticketPolicy.partialSave')
    expect((w.get('[data-testid="ticket-rule-target_length"]').element as HTMLInputElement).value).toBe('356')
    await save(w)
    expect(mocks.updateTickets.mock.calls[1][1]).toBe('r1')
    expect(mocks.success).toHaveBeenCalledTimes(1)
    w.unmount()
  })
})
