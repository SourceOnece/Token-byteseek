import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Stages from '../CodexTicketValidationStages.vue'
import type { TicketValidationStage } from '@/api/admin/codexTickets'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('票据两阶段长度语义', () => {
  it.each([
    ['harvest', 332, 'text-emerald-700'], ['harvest', 312, 'text-bh-red'], ['harvest', 356, 'text-yellow-700'], ['harvest', 0, 'text-yellow-700'],
    ['verify', 332, 'text-emerald-700'], ['verify', 312, 'text-bh-red'], ['verify', 356, 'text-yellow-700'], ['verify', 0, 'text-emerald-700']
  ] as const)('%s 返回 %d B 使用 %s', (name, length, color) => {
    const stage: TicketValidationStage = { name, state_length: length, http_status: 200, request_model: 'gpt-6-astra', response_model: 'gpt-6-astra', complete: true }
    const w = mount(Stages, { props: { stages: [stage], target: 332, degradedSignalLength: 312 } })
    expect(w.get('[data-testid="ticket-stage-length-' + name + '"]').classes()).toContain(color)
    w.unmount()
  })
  it('按历史事件自己的目标和信号着色，信号优先于合格，不拿当前设置改历史结论', () => {
    const stage: TicketValidationStage = { name: 'verify', state_length: 332, target_length: 332, degraded_signal_length: 332, http_status: 200, request_model: 'gpt-6-astra', complete: true }
    const w = mount(Stages, { props: { stages: [stage], target: 356, degradedSignalLength: 312 } })
    expect(w.get('[data-testid="ticket-stage-length-verify"]').classes()).toContain('text-bh-red')
    w.unmount()
  })
})
