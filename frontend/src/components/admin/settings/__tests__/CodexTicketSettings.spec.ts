import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodexTicketSettings from '../CodexTicketSettings.vue'
import CodexTicketProxyEditor from '@/components/admin/account/CodexTicketProxyEditor.vue'
const { get, put, showSuccess } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), showSuccess: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get, put, post: vi.fn() } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const data = () => ({ enabled: true, revision: 'r1', proxy_policy: { mode: 'fixed', dynamic_source: 'template', extraction_configured: false, proxy_protocol: 'http', fixed_proxy_id: 'legacy', proxies: [{ id: 'legacy', name: 'A', configured: true }] } })
describe('CodexTicketSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); get.mockResolvedValue({ data: data() }); put.mockResolvedValue({ data: data() }) })
  it('网关只有总开关和代理，不再显示账号采集规则', async () => {
    const w = mount(CodexTicketSettings); await flushPromises()
    for (const id of ['ticket-models', 'ticket-attempts', 'ticket-watchdog', 'ticket-length']) expect(w.find('#' + id).exists()).toBe(false)
    expect(put).not.toHaveBeenCalled()
    await w.get('button[role=switch]').trigger('click')
    await w.get('[data-testid=codex-ticket-save]').trigger('click'); await flushPromises()
    expect(put).toHaveBeenCalledWith('/admin/settings/codex-ticket', { enabled: false, revision: 'r1', proxy_policy: undefined })
    w.unmount()
  })
  it('代理必须勾选修改，已有密码不回显', async () => {
    const w = mount(CodexTicketSettings); await flushPromises()
    const editor = w.findComponent(CodexTicketProxyEditor)
    expect(editor.props('locked')).toBe(true)
    await w.get('input[type=checkbox]').setValue(true)
    expect(editor.props('locked')).toBe(false)
    await w.get('[data-testid=ticket-proxy-url]').setValue('http://user:synthetic@p.invalid:8080')
    await w.get('[data-testid=codex-ticket-save]').trigger('click'); await flushPromises()
    expect(put.mock.calls[0][1].proxy_policy.proxies[0].harvest_proxy_url).toContain('synthetic')
    expect((w.get('[data-testid=ticket-proxy-url]').element as HTMLInputElement).value).toBe('')
    expect(showSuccess).toHaveBeenCalledOnce(); w.unmount()
  })
  it('旧后端或加载失败禁止覆盖', async () => {
    get.mockResolvedValue({ data: { enabled: true } })
    const w = mount(CodexTicketSettings); await flushPromises()
    expect(w.get('[data-testid=codex-ticket-save]').attributes('disabled')).toBeDefined()
    expect(put).not.toHaveBeenCalled(); w.unmount()
  })
})
