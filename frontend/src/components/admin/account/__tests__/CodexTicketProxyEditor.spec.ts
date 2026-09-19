import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Editor from '../CodexTicketProxyEditor.vue'
import Select from '@/components/common/Select.vue'
const { getAll, testProxy } = vi.hoisted(() => ({ getAll: vi.fn(), testProxy: vi.fn() }))
vi.mock('@/api/admin/proxies', () => ({ getAll }))
vi.mock('@/api/admin/codexTickets', () => ({ testTicketProxy: testProxy }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }) }))
describe('采集代理来源', () => {
  beforeEach(() => { vi.clearAllMocks(); getAll.mockResolvedValue([{ id: 9, name: 'TW', host: 'proxy.invalid', port: 8080 }]) })
  it('打开已有管理代理就加载选项，轮换可同时使用管理代理与手填地址', async () => {
    const w = mount(Editor, { props: { value: { mode: 'rotate', dynamic_source: 'template', proxy_protocol: 'http', extraction_configured: false, fixed_proxy_id: 'managed', proxies: [{ id: 'managed', name: 'TW', configured: true, managed_proxy_id: 9 }, { id: 'manual', name: 'Manual', configured: true }] } } })
    await flushPromises()
    expect(getAll).toHaveBeenCalledOnce()
    expect(w.findAllComponents(Select).some(s => s.props('options').some(o => o.value === 9))).toBe(true)
    await w.get('[data-testid="ticket-proxy-url"]').setValue('socks5h://user:synthetic@manual.invalid:1080')
    const patch = (w.vm as unknown as { patch: () => unknown }).patch()
    expect(patch).toMatchObject({ mode: 'rotate', proxies: [{ managed_proxy_id: 9, harvest_proxy_url: '' }, { harvest_proxy_url: 'socks5h://user:synthetic@manual.invalid:1080' }] })
    expect(testProxy).not.toHaveBeenCalled()
    w.unmount()
  })
})
