import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodexTicketSettings from '../CodexTicketSettings.vue'

const { get, put, showSuccess } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), showSuccess: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get, put } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('CodexTicketSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockResolvedValue({ data: { enabled: false, proxy_configured: false } })
    put.mockResolvedValue({ data: { enabled: true, proxy_configured: true } })
  })

  it('默认关闭且不自动保存，手动保存后清空代理密码', async () => {
    const wrapper = mount(CodexTicketSettings)
    expect(wrapper.get('button[role="switch"]').attributes('disabled')).toBeDefined()
    await flushPromises()
    expect(get).toHaveBeenCalledWith('/admin/settings/codex-ticket')
    expect(wrapper.get('button[role="switch"]').attributes('aria-checked')).toBe('false')
    expect(put).not.toHaveBeenCalled()
    await wrapper.get('button[role="switch"]').trigger('click')
    await wrapper.get('input[type="password"]').setValue(' socks5h://user:secret@proxy:1080 ')
    await wrapper.get('[data-testid="codex-ticket-save"]').trigger('click')
    await flushPromises()
    expect(put).toHaveBeenCalledWith('/admin/settings/codex-ticket', { enabled: true, harvest_proxy_url: 'socks5h://user:secret@proxy:1080', clear_proxy: false })
    expect((wrapper.get('input[type="password"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).not.toContain('secret')
    expect(showSuccess).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('加载失败时禁止保存，可重试；不把未知状态当成关闭覆盖后端', async () => {
    get.mockRejectedValueOnce(new Error('offline'))
    const wrapper = mount(CodexTicketSettings)
    await flushPromises()
    expect(wrapper.get('[data-testid="codex-ticket-save"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="codex-ticket-save"]').trigger('click')
    expect(put).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b => b.text() === 'common.retry')!.trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="codex-ticket-save"]').attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('清除代理必须显式勾选，保存中防重复，失败保留草稿', async () => {
    get.mockResolvedValue({ data: { enabled: true, proxy_configured: true } })
    let reject!: (e: Error) => void
    put.mockImplementation(() => new Promise((_resolve, r) => { reject = r }))
    const wrapper = mount(CodexTicketSettings)
    await flushPromises()
    await wrapper.get('button[role="switch"]').trigger('click')
    await wrapper.get('input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="codex-ticket-save"]').trigger('click')
    await wrapper.get('[data-testid="codex-ticket-save"]').trigger('click')
    expect(put).toHaveBeenCalledOnce()
    expect(put.mock.calls[0][1]).toEqual({ enabled: false, harvest_proxy_url: '', clear_proxy: true })
    reject(new Error('failed'))
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect((wrapper.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(showSuccess).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
