import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelAllowlistField from '../ModelAllowlistField.vue'
import Toggle from '@/components/common/Toggle.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('独立模型调用白名单', () => {
  it('缺省关闭且不发出保存修改', () => {
    const wrapper = mount(ModelAllowlistField, { props: { modelValue: { enabled: false, models: [] } } })
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    wrapper.unmount()
  })
  it('输入时保留换行，关闭时保留已编辑条目', async () => {
    const wrapper = mount(ModelAllowlistField, { props: { modelValue: { enabled: true, models: ['gpt-*'] } } })
    await wrapper.get('textarea').setValue('gpt-*\n')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([{ enabled: true, models: ['gpt-*', ''] }])
    wrapper.getComponent(Toggle).vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual([{ enabled: false, models: ['gpt-*'] }])
    wrapper.unmount()
  })
})
