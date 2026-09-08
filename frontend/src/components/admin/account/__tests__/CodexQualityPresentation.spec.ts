import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexQualityProgress from '../CodexQualityProgress.vue'
import CodexQualityResult from '../CodexQualityResult.vue'
import CodexQualitySummary from '../CodexQualitySummary.vue'
import { qualityStatusClass } from '../codexQualityPresentation'
import type { CodexQualityResult as Result } from '@/api/admin/codexQuality'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('Quality presentation', () => {
  it('只用蓝红黄文字，不填充状态背景', () => {
    expect(qualityStatusClass('full')).toContain('text-bh-blue')
    expect(qualityStatusClass('degraded')).toContain('text-bh-red')
    expect(qualityStatusClass('failed')).toContain('text-yellow')
    expect(qualityStatusClass('full')).not.toContain('bg-')
  })
  it('统计点击只发出相应分类', async () => {
    const w = mount(CodexQualitySummary, { props: { counts: { full: 2, degraded: 1, failed: 1 } } })
    expect(w.text()).toContain('50.0%')
    await w.get('[data-testid="quality-summary-full"]').trigger('click')
    expect(w.emitted('select')).toEqual([['full']])
  })
  it('回答默认折叠，模型和等级清晰可见', () => {
    const w = mount(CodexQualityResult, { props: { result: { account_id: 1, model: 'gpt-6-astra', reasoning_effort: 'high', status: 'full', response_text: '<script>private</script>' } as Result } })
    expect(w.get('[data-testid="quality-answer"]').attributes('open')).toBeUndefined()
    expect(w.text()).toContain('gpt-6-astra'); expect(w.text()).toContain('high')
    expect(w.find('script').exists()).toBe(false)
  })
  it('中断时保持实际进度，不伪造完成', () => {
    const w = mount(CodexQualityProgress, { props: { done: 27, total: 63, label: 'Stopped' } })
    expect(w.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('27')
    expect(w.text()).toContain('42%')
  })
})
