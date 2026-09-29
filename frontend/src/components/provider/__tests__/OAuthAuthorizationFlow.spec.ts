import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OAuthAuthorizationFlow from '../OAuthAuthorizationFlow.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: false,
    copyToClipboard: vi.fn()
  })
}))

describe('OAuthAuthorizationFlow', () => {
  it('AT 输入显示邮箱，但不显示完整令牌', async () => {
    const wrapper = mount(OAuthAuthorizationFlow, { props: { addMethod: 'oauth', platform: 'openai', showCodexSessionImportOption: true, initialInputMethod: 'codex_session' }, global: { stubs: { Icon: true } } })
    const token = `head.${btoa(JSON.stringify({ 'https://api.openai.com/profile': { email: 'preview@example.invalid' } }))}.signature`
    await wrapper.find('textarea').setValue(token)
    expect(wrapper.text()).toContain('preview@example.invalid')
    expect(wrapper.text()).not.toContain(token)
  })
  it('emits Codex PAT token for OpenAI PAT auth mode', async () => {
    const wrapper = mount(OAuthAuthorizationFlow, {
      props: {
        addMethod: 'oauth',
        platform: 'openai',
        showCookieOption: false,
        showRefreshTokenOption: false,
        showCodexPatOption: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const patRadio = wrapper.find('input[value="codex_pat"]')
    expect(patRadio.exists()).toBe(true)

    await patRadio.setValue(true)
    const tokenInput = wrapper.find('input[type="password"]')
    await tokenInput.setValue(' at-test-token ')
    await wrapper.find('button.btn-primary').trigger('click')

    expect(wrapper.emitted('update:inputMethod')?.at(-1)).toEqual(['codex_pat'])
    expect(wrapper.emitted('import-codex-pat')).toEqual([['at-test-token']])
  })

  it('emits trimmed batch input for Grok SSO auth mode', async () => {
    const wrapper = mount(OAuthAuthorizationFlow, {
      props: {
        addMethod: 'oauth',
        platform: 'grok',
        showSsoOption: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const ssoRadio = wrapper.find('input[value="sso_cookie"]')
    expect(ssoRadio.exists()).toBe(true)

    await ssoRadio.setValue(true)
    await wrapper.find('textarea').setValue('  sso-one\nsso-two  ')
    await wrapper.find('button.btn-primary').trigger('click')

    expect(wrapper.emitted('update:inputMethod')?.at(-1)).toEqual(['sso_cookie'])
    expect(wrapper.emitted('import-sso')).toEqual([['sso-one\nsso-two']])
  })
})
