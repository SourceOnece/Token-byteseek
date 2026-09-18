import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexTicketLength from '../CodexTicketLength.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('票据长度仅控制显示', () => {
  it.each([
    [332, 292, false, 'text-yellow-700'],
    [292, 292, false, 'text-emerald-700'],
    [356, 292, true, 'text-bh-red'],
    [292, 292, true, 'text-bh-red'],
    [356, 356, false, 'text-emerald-700']
  ] as const)('实际 %i / 目标 %i，信号 %s', (actual, target, signal, color) => {
    const w = mount(CodexTicketLength, { props: { actual, target, signal } })
    expect(w.text()).toBe(`${actual}/${target}`)
    expect(w.get('[data-testid="ticket-length-actual"]').classes()).toContain(color)
    expect(w.get('[data-testid="ticket-length-target"]').classes()).toContain('text-emerald-700')
    expect(w.get('[data-testid="ticket-length-ratio"]').classes().some(value => value.startsWith('border'))).toBe(false)
    expect(w.get('[data-testid="ticket-length-ratio"]').attributes('aria-label')).toBeTruthy()
    w.unmount()
  })
})
