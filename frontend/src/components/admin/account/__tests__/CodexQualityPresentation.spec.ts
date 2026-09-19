import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CodexQualityProgress from '../CodexQualityProgress.vue'
import CodexQualityResult from '../CodexQualityResult.vue'
import CodexQualitySummary from '../CodexQualitySummary.vue'
import CodexQualityRules from '../CodexQualityRules.vue'
import BauhausHelp from '@/components/common/BauhausHelp.vue'
import { qualityStatusClass } from '../codexQualityPresentation'
import type { CodexQualityResult as Result } from '@/api/admin/codexQuality'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('Quality presentation', () => {
  it('规则用三元素和文字表达，辅助说明默认折叠', () => {
    const w = mount(CodexQualityRules)
    expect(w.findAll('[aria-hidden="true"].bg-current')).toHaveLength(3)
    for (const status of ['full', 'degraded', 'failed']) {
      expect(w.text()).toContain(`status.${status}`)
      expect(w.text()).toContain(`rule.${status}`)
    }
    expect(w.findAll('button')).toHaveLength(0)
    expect(w.get('[data-testid="quality-rules"]').classes()).toContain('quality-rules')
    expect(w.get('[data-testid="quality-rule-status"]').classes()).toContain('text-lg')
    expect(w.get('[data-testid="quality-rule-action"]').classes()).toContain('text-sm')
    const help = mount(BauhausHelp, { props: { title: '规则' }, slots: { default: '辅助说明' } })
    expect(help.get('details').attributes('open')).toBeUndefined()
    expect(help.get('summary').text()).toBe('规则')
    expect(help.get('details').classes()).toContain('text-sm')
  })
  it('满血统一绿，降智红失败黄，不填充状态背景', () => {
    expect(qualityStatusClass('full')).toContain('text-emerald-700')
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
