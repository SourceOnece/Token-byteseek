import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import CodexQualityTestModal from '../CodexQualityTestModal.vue'

const { runBatch, getModels } = vi.hoisted(() => ({ runBatch: vi.fn(), getModels: vi.fn() }))
const admin = reactive({ user: { id: 7 } })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => admin }))
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
  beforeEach(() => { localStorage.clear(); admin.user.id = 7; getModels.mockReset(); getModels.mockResolvedValue([{ id: 'gpt-6-astra' }]); runBatch.mockReset() })
  it('重新挂载和换账号选择仍恢复七项配置，但不自动确认或执行', async () => {
    const wrapper = create()
    await wrapper.setProps({ show: true }); await flushPromises()
    wrapper.getComponent('#quality-model').vm.$emit('update:modelValue', 'custom-model')
    wrapper.getComponent('#quality-effort').vm.$emit('update:modelValue', 'high')
    wrapper.getComponent('#quality-protocol').vm.$emit('update:modelValue', 'chat_completions')
    wrapper.getComponent('#quality-concurrency').vm.$emit('update:modelValue', 4)
    await wrapper.get('#quality-timeout').setValue('300')
    await wrapper.get('#quality-prompt').setValue('保留多行\n测试')
    await wrapper.get('#quality-keyword').setValue('关键词')
    await wrapper.get('[data-testid="quality-confirm"]').setValue(true)
    wrapper.unmount()

    getModels.mockRejectedValue(new Error('offline'))
    const restored = create()
    await restored.setProps({ show: true, accountIds: [99] }); await flushPromises()
    expect(restored.getComponent('#quality-model').props('modelValue')).toBe('custom-model')
    expect(restored.getComponent('#quality-effort').props('modelValue')).toBe('high')
    expect(restored.getComponent('#quality-protocol').props('modelValue')).toBe('chat_completions')
    expect(restored.getComponent('#quality-concurrency').props('modelValue')).toBe(4)
    expect(restored.get<HTMLInputElement>('#quality-timeout').element.value).toBe('300')
    expect(restored.get<HTMLTextAreaElement>('#quality-prompt').element.value).toBe('保留多行\n测试')
    expect(restored.get<HTMLInputElement>('#quality-keyword').element.value).toBe('关键词')
    expect(restored.get<HTMLInputElement>('[data-testid="quality-confirm"]').element.checked).toBe(false)
    expect(runBatch).not.toHaveBeenCalled()
    runBatch.mockResolvedValue(undefined)
    await restored.get('[data-testid="quality-confirm"]').setValue(true)
    await restored.get('[data-testid="quality-start"]').trigger('click'); await flushPromises()
    expect(runBatch.mock.calls[0][0]).toMatchObject({ account_ids: [99], model: 'custom-model', timeout_seconds: 300 })
    restored.unmount()
  })
  it('打开期间换管理员清空他人题目，切回可恢复且确认不记住', async () => {
    const wrapper = create()
    await wrapper.setProps({ show: true }); await flushPromises()
    await wrapper.get('#quality-prompt').setValue('管理员7题目')
    await wrapper.get('#quality-keyword').setValue('答案7')
    await wrapper.get('[data-testid="quality-confirm"]').setValue(true)
    admin.user.id = 8; await flushPromises()
    expect(wrapper.get<HTMLTextAreaElement>('#quality-prompt').element.value).toBe('')
    expect(wrapper.get<HTMLInputElement>('[data-testid="quality-confirm"]').element.checked).toBe(false)
    await wrapper.get('#quality-prompt').setValue('管理员8题目')
    admin.user.id = 7; await flushPromises()
    expect(wrapper.get<HTMLTextAreaElement>('#quality-prompt').element.value).toBe('管理员7题目')
    wrapper.unmount()
  })
  it('显式确认才允许改调度，发送模型题目关键词与冻结账号', async () => {
    runBatch.mockResolvedValue(undefined)
    const wrapper = create()
    await wrapper.setProps({ show: true }); await flushPromises()
    await wrapper.get('#quality-prompt').setValue('题目')
    await wrapper.get('#quality-keyword').setValue('答案')
    expect(wrapper.get('#quality-timeout').element).toHaveProperty('value', '120')
    await wrapper.get('#quality-timeout').setValue('240')
    wrapper.getComponent('#quality-protocol').vm.$emit('update:modelValue', 'chat_completions')
    expect(wrapper.get('[data-testid="quality-start"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="quality-confirm"]').setValue(true)
    await wrapper.setProps({ accountIds: [99] })
    await wrapper.get('[data-testid="quality-start"]').trigger('click'); await flushPromises()
    expect(runBatch).toHaveBeenCalledTimes(1)
    expect(runBatch.mock.calls[0][0]).toMatchObject({ account_ids: [12, 19], model: 'gpt-6-astra', prompt: '题目', keyword: '答案', api_protocol: 'chat_completions', confirm_scheduling: true, concurrency: 3, timeout_seconds: 240 })
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
