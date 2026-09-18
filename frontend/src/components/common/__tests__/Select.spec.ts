import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import Select from '../Select.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const originalInnerWidth = window.innerWidth

afterEach(() => {
  document.body.innerHTML = ''
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: originalInnerWidth
  })
  vi.restoreAllMocks()
})

describe('Select dropdown viewport constraints', () => {
  it('滚动后重新决定上下方向，不把长表单的菜单留在视口外', async () => {
    let top = window.innerHeight - 50
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => ({
      x: 20, y: top, top, right: 220, bottom: top + 40, left: 20, width: 200, height: 40, toJSON: () => ({})
    }))
    const wrapper = mount(Select, { props: { modelValue: 'a', options: [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }] } })
    await wrapper.get('button').trigger('click'); await nextTick(); await nextTick()
    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown-portal')!
    expect(dropdown.style.bottom).not.toBe('')
    top = 20
    window.dispatchEvent(new Event('scroll')); await nextTick(); await nextTick()
    expect(dropdown.style.bottom).toBe('')
    expect(dropdown.style.top).toBe('64px')
    wrapper.unmount()
  })

  it('repositions the teleported dropdown within the viewport', async () => {
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 320
    })

    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      x: 220,
      y: 20,
      top: 20,
      right: 300,
      bottom: 60,
      left: 220,
      width: 80,
      height: 40,
      toJSON: () => ({})
    })

    const wrapper = mount(Select, {
      props: {
        modelValue: null,
        options: [
          {
            value: 'example',
            label: 'very-long-unbroken-option-value-that-must-not-overflow'
          }
        ]
      }
    })

    await wrapper.get('button').trigger('click')
    await nextTick()
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown-portal')

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.left).toBe('104px')
    expect(dropdown?.style.minWidth).toBe('80px')
    expect(dropdown?.style.maxWidth).toBe('288px')

    wrapper.unmount()
  })
})
