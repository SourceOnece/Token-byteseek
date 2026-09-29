import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import GroupFormTabs from '../GroupFormTabs.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

afterEach(() => { document.body.innerHTML = '' })

describe('GroupFormTabs', () => {
  it('所有分组提供相同的功能页签', async () => {
    const wrapper = mount(GroupFormTabs, { props: { idPrefix: 'create' } })
    expect(wrapper.findAll('[data-group-tab-button]').map(tab => tab.attributes('data-group-tab-button'))).toEqual(['general', 'models', 'scheduling', 'protocol', 'request'])
    await wrapper.get('[data-group-tab-button="request"]').trigger('click')
    expect(wrapper.get('[data-group-tab="request"]').isVisible()).toBe(true)
    wrapper.unmount()
  })

  it('切页保持内部草稿并支持键盘导航及滚动复位', async () => {
    // 子组件的本地状态用于检测是否错误地在切页时卸载。
    const Draft = defineComponent({ setup: () => ({ value: ref('') }), template: '<input v-model="value" />' })
    const wrapper = mount(GroupFormTabs, {
      attachTo: document.body,
      props: { idPrefix: 'edit' },
      slots: { models: Draft },
    })
    await wrapper.get('[data-group-tab-button="models"]').trigger('click')
    await wrapper.get('input').setValue('draft')
    const content = wrapper.get('.group-tab-content').element
    content.scrollTop = 300
    await wrapper.get('[data-group-tab-button="models"]').trigger('keydown', { key: 'ArrowRight' })
    await flushPromises()
    expect(document.activeElement).toBe(wrapper.get('[data-group-tab-button="scheduling"]').element)
    expect(content.scrollTop).toBe(0)
    await wrapper.get('[data-group-tab-button="scheduling"]').trigger('keydown', { key: 'Home' })
    expect(wrapper.get('[data-group-tab-button="general"]').attributes('aria-selected')).toBe('true')
    await wrapper.get('[data-group-tab-button="models"]').trigger('click')
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('draft')
    wrapper.unmount()
  })

  it('显示无效输入所在页签后才报告原生校验错误', async () => {
    const wrapper = mount(GroupFormTabs, {
      attachTo: document.body,
      props: { idPrefix: 'edit' },
      slots: { models: '<input type="number" min="0.001" value="-1" />' },
    })
    const field = wrapper.get('input').element as HTMLInputElement
    const report = vi.spyOn(field, 'reportValidity')
    expect(await wrapper.vm.validate()).toBe(false)
    expect(wrapper.get('[data-group-tab="models"]').isVisible()).toBe(true)
    expect(document.activeElement).toBe(field)
    expect(report).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('创建引导可以往返定位倍率与通用字段', async () => {
    const wrapper = mount(GroupFormTabs, {
      props: { idPrefix: 'create' },
      slots: {
        general: '<input data-tour="group-form-name" /><input data-tour="group-form-multiplier" />',
      },
    })
    for (const [field, tab] of [['multiplier', 'general'], ['name', 'general']]) {
      wrapper.get(`[data-tour="group-form-${field}"]`).element
        .dispatchEvent(new Event('onboarding-reveal', { bubbles: true }))
      await flushPromises()
      expect(wrapper.get(`[data-group-tab="${tab}"]`).isVisible()).toBe(true)
    }
    wrapper.unmount()
  })

  it('跨页定位错误字段后保留滚动位置', async () => {
    const wrapper = mount(GroupFormTabs, {
      attachTo: document.body,
      props: { idPrefix: 'edit' },
      slots: { general: '<input data-testid="probe-model" />' },
    })
    try {
      await wrapper.get('[data-group-tab-button="models"]').trigger('click')
      const content = wrapper.get('.group-tab-content').element
      const field = wrapper.get('[data-testid="probe-model"]').element as HTMLInputElement
      // jsdom 没有布局引擎，模拟浏览器将下方错误字段滚入视口后的实际位置。
      field.scrollIntoView = vi.fn(() => { content.scrollTop = 480 })

      await wrapper.vm.revealField('[data-testid="probe-model"]')
      await flushPromises()

      expect(wrapper.get('[data-group-tab="general"]').isVisible()).toBe(true)
      expect(document.activeElement).toBe(field)
      expect(content.scrollTop).toBe(480)
    } finally {
      wrapper.unmount()
    }
  })
})
