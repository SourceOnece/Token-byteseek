import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CodexQualityTestModal from '../CodexQualityTestModal.vue'

const { runBatch, getModels } = vi.hoisted(() => ({ runBatch: vi.fn(), getModels: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { getAvailableModels: getModels } } }))
vi.mock('@/api/admin/codexQuality', async () => ({
  ...await vi.importActual<typeof import('@/api/admin/codexQuality')>('@/api/admin/codexQuality'), runCodexQualityBatch: runBatch
}))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))

const create = () => mount(CodexQualityTestModal, {
  props: { show: false, accountIds: [12, 19] },
  global: { stubs: {
    BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' },
    Select: { props: ['modelValue'], template: '<div />' },
    CodexQualityResultCard: true
  } }
})

describe('CodexQualityTestModal', () => {
  beforeEach(() => { getModels.mockResolvedValue([{ id: 'gpt-6-astra' }]); runBatch.mockReset() })
  it('显式确认才允许改调度，发送模型题目关键词与冻结账号', async () => {
    runBatch.mockResolvedValue(undefined)
    const wrapper = create()
    await wrapper.setProps({ show: true }); await flushPromises()
    await wrapper.get('#quality-prompt').setValue('题目')
    await wrapper.get('#quality-keyword').setValue('答案')
    expect(wrapper.get('#quality-timeout').element).toHaveProperty('value', '120')
    await wrapper.get('#quality-timeout').setValue('240')
    expect(wrapper.get('[data-testid="quality-start"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="quality-confirm"]').setValue(true)
    await wrapper.setProps({ accountIds: [99] })
    await wrapper.get('[data-testid="quality-start"]').trigger('click'); await flushPromises()
    expect(runBatch).toHaveBeenCalledTimes(1)
    expect(runBatch.mock.calls[0][0]).toMatchObject({ account_ids: [12, 19], model: 'gpt-6-astra', prompt: '题目', keyword: '答案', confirm_scheduling: true, concurrency: 3, timeout_seconds: 240 })
    expect(wrapper.emitted('finished')).toHaveLength(1)
  })
  it('关闭取消请求，保留已完成结果事件', async () => {
    let resolve: () => void = () => {}
    runBatch.mockImplementation((_request, _signal, onResult) => {
      onResult({ account_id: 12, status: 'full' })
      return new Promise<void>(done => { resolve = done })
    })
    const wrapper = create(); await wrapper.setProps({ show: true }); await flushPromises()
    await wrapper.get('#quality-prompt').setValue('题目'); await wrapper.get('#quality-keyword').setValue('答案')
    await wrapper.get('[data-testid="quality-confirm"]').setValue(true)
    await wrapper.get('[data-testid="quality-start"]').trigger('click')
    expect(wrapper.emitted('result')).toHaveLength(1)
    await wrapper.findAll('button').find(button => button.text() === 'common.close')!.trigger('click')
    expect(runBatch.mock.calls[0][1].aborted).toBe(true)
    resolve(); await flushPromises()
  })
})
