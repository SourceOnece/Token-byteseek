import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { useProtocolCatalogFixture } from '@/__tests__/helpers/protocolCatalog'
import { flushPromises, mount } from '@vue/test-utils'
import { ticketAccountAPI } from '@/api/admin/codexTickets'
import { ticketSettingsFixture } from '@/components/admin/provider/__tests__/ticketSettingsFixture'

vi.mock('@/api/admin/codexTickets', () => ({ ticketAccountAPI: { get: vi.fn(), update: vi.fn() }, testTicketProxy: vi.fn() }))
vi.mock('@/api/admin/proxies', () => ({ getAll: vi.fn().mockResolvedValue([]) }))

const { updateAccountMock, checkMixedChannelRiskMock, getWebSearchEmulationConfigMock, getSettingsMock, listTLSProfilesMock, authIsSimpleMode } = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  getWebSearchEmulationConfigMock: vi.fn(),
  getSettingsMock: vi.fn(),
  listTLSProfilesMock: vi.fn(),
  authIsSimpleMode: { value: true }
}))

function coerceSelectStubValue(value: string, options: unknown[]): string | number | boolean | null {
  const option = (options as Array<Record<string, unknown>>).find((item) => String(item.value ?? '') === value)
  return option ? (option.value as string | number | boolean | null) : value
}

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return authIsSimpleMode.value
    }
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    settings: {
      getSettings: getSettingsMock,
      getWebSearchEmulationConfig: getWebSearchEmulationConfigMock
    },
    providers: {
      update: updateAccountMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    },
    tlsFingerprintProfiles: {
      list: listTLSProfilesMock
    }
  }
}))

vi.mock('@/api/admin/providers', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import EditAccountModal from '../EditProviderModal.vue'

useProtocolCatalogFixture()

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div>
      <button
        type="button"
        data-testid="rewrite-to-snapshot"
        @click="$emit('update:modelValue', ['gpt-5.2-2025-12-11'])"
      >
        rewrite
      </button>
      <button
        type="button"
        data-testid="rewrite-to-qoder-defaults"
        @click="$emit('update:modelValue', ['claude-opus-4-6', 'auto'])"
      >
        rewrite qoder
      </button>
      <span data-testid="model-whitelist-value">
        {{ Array.isArray(modelValue) ? modelValue.join(',') : '' }}
      </span>
    </div>
  `
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue', 'change'],
  methods: {
    coerceSelectStubValue
  },
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="
        (event) => {
          const value = coerceSelectStubValue(event.target.value, options)
          const option = options.find((item) => String(item.value ?? '') === event.target.value) ?? null
          $emit('update:modelValue', value)
          $emit('change', value, option)
        }
      "
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
})

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: {
    groups: { type: Array, default: () => [] },
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div data-testid="group-selector">
      <button
        type="button"
        data-testid="set-shadow-group"
        @click="$emit('update:modelValue', [7])"
      >
        group
      </button>
    </div>
  `
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI Key',
    notes: '',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      api_key: 'sk-test',
      base_url: 'https://api.openai.com',
      model_whitelist: ['gpt-5.2']
    },
    extra: {},
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildOpenAIOAuthAccount() {
  const account = buildAccount()
  return {
    ...account,
    id: 7,
    name: 'OpenAI OAuth',
    type: 'oauth',
    credentials: {
      email: 'oauth@example.com',
      plan_type: 'chatgptpro',
      model_mapping: {
        'gpt-5.4': 'gpt-5.4'
      }
    }
  } as any
}
function mountModal(account = buildAccount()) {
  return mount(EditAccountModal, {
    props: {
      show: true,
      provider: account,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: SelectStub,
        Icon: true,
        ProxySelector: true,
        GroupSelector: GroupSelectorStub,
        ModelWhitelistSelector: ModelWhitelistSelectorStub
      }
    }
  })
}

describe('EditAccountModal', () => {
  beforeEach(() => {
    vi.mocked(ticketAccountAPI.get).mockReset().mockImplementation(async id => ticketSettingsFixture(id))
    vi.mocked(ticketAccountAPI.update).mockReset().mockResolvedValue([{ ...ticketSettingsFixture(), revision: 'r2' }])
    authIsSimpleMode.value = true
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    getWebSearchEmulationConfigMock.mockReset()
    getSettingsMock.mockReset()
    listTLSProfilesMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    getSettingsMock.mockResolvedValue({ account_quota_notify_enabled: false })
    getWebSearchEmulationConfigMock.mockResolvedValue({ enabled: false, providers: [] })
    listTLSProfilesMock.mockResolvedValue([])
  })

  it('原账号保存按钮统一提交票据，无改动时不写票据', async () => {
    const account = buildOpenAIOAuthAccount()
    updateAccountMock.mockResolvedValue(account)
    const w = mountModal(account)
    await flushPromises()
    expect(w.find('[data-testid="ticket-account-save"]').exists()).toBe(false)
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('356')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(ticketAccountAPI.update).toHaveBeenCalledWith([account.id], { rules: { ...ticketSettingsFixture().rules, target_length: 356 } }, 'r1')
    expect(updateAccountMock.mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(ticketAccountAPI.update).mock.invocationCallOrder[0])
    expect(w.emitted('close')).toHaveLength(1)
    w.unmount()
    const unchanged = mountModal(account)
    await flushPromises()
    await unchanged.get('form').trigger('submit')
    await flushPromises()
    expect(ticketAccountAPI.update).toHaveBeenCalledTimes(1)
    unchanged.unmount()
  })

  it('关闭只保存票据开关，隐藏的非法草稿不会卡住保存', async () => {
    const account = buildOpenAIOAuthAccount()
    updateAccountMock.mockResolvedValue(account)
    const w = mountModal(account)
    await flushPromises()
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('2')
    await w.get('[data-testid="ticket-master-toggle"]').trigger('click')
    expect(w.get('[data-testid="ticket-workbench-body"]').isVisible()).toBe(false)
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(ticketAccountAPI.update).toHaveBeenCalledWith([account.id], { mode: 'off' }, 'r1')
    expect(w.emitted('close')).toHaveLength(1)
    w.unmount()
  })

  it('票据保存失败保留弹窗和草稿，普通账号失败不写票据', async () => {
    const account = buildOpenAIOAuthAccount()
    updateAccountMock.mockResolvedValue(account)
    vi.mocked(ticketAccountAPI.update).mockRejectedValueOnce(new Error('revision conflict'))
    const w = mountModal(account)
    await flushPromises()
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('356')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(w.emitted('close')).toBeUndefined()
    expect(w.get('[role="alert"]').text()).toContain('revision conflict')
    expect((w.get('[data-testid="ticket-rule-target_length"]').element as HTMLInputElement).value).toBe('356')
    vi.mocked(ticketAccountAPI.update).mockClear()
    updateAccountMock.mockRejectedValueOnce(new Error('ordinary failed'))
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(ticketAccountAPI.update).not.toHaveBeenCalled()
    expect(w.emitted('close')).toBeUndefined()
    w.unmount()
  })

  it('票据无效或加载失败时先拦截，不先改普通账号', async () => {
    const w = mountModal(buildOpenAIOAuthAccount())
    await flushPromises()
    await w.get('[data-testid="ticket-rule-target_length"]').setValue('312')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(updateAccountMock).not.toHaveBeenCalled()
    expect(ticketAccountAPI.update).not.toHaveBeenCalled()
    w.unmount()
    vi.mocked(ticketAccountAPI.get).mockRejectedValueOnce(new Error('offline'))
    const failed = mountModal(buildOpenAIOAuthAccount())
    await flushPromises()
    await failed.get('form').trigger('submit')
    await flushPromises()
    expect(updateAccountMock).not.toHaveBeenCalled()
    expect(failed.emitted('close')).toBeUndefined()
    failed.unmount()
  })

  it('补回已绑定停用分组且保留活跃列表最新投影，移除后只提交剩余绑定', async () => {
    authIsSimpleMode.value = false
    const account = { ...buildAccount(), group_ids: [1, 2], groups: [
      { id: 1, name: '旧名称', platform: 'openai', status: 'active' },
      { id: 2, name: '停用组', platform: 'openai', status: 'inactive' },
      { id: 3, name: '未绑定组', platform: 'openai', status: 'inactive' }
    ] }
    updateAccountMock.mockResolvedValue(account)
    const wrapper = mountModal(account)
    await wrapper.setProps({ groups: [{ id: 1, name: '最新名称', platform: 'openai', status: 'active', account_count: 5 }] as any })
    const selector = wrapper.getComponent(GroupSelectorStub)
    expect(selector.props('groups')).toEqual([
      expect.objectContaining({ id: 1, name: '最新名称', account_count: 5 }),
      expect.objectContaining({ id: 2, status: 'inactive' })
    ])
    selector.vm.$emit('update:modelValue', [1])
    await wrapper.get('form#edit-provider-form').trigger('submit.prevent')
    await flushPromises()
    await flushPromises()
    expect(updateAccountMock.mock.calls[0]?.[1].group_ids).toEqual([1])
    wrapper.unmount()
  })

  it('MiniMax 编辑保留中继端点及 Coding 套餐', async () => {
    const account = buildAccount()
    account.platform = 'minimax'
    account.credentials = { api_key: 'sk-minimax-test', provider_mode: 'coding', upstream_protocols: ['anthropic_messages', 'openai_responses', 'openai_chat_completions'], base_url: 'https://relay.example/v1', api_base_urls: { chat_completions: 'https://relay.example/v1', anthropic: 'https://relay.example/anthropic', responses: 'https://relay.example/v1' } }
    updateAccountMock.mockReset().mockResolvedValue(account)
    checkMixedChannelRiskMock.mockReset().mockResolvedValue({ has_risk: false })
    const wrapper = mountModal(account)
    await wrapper.get('form#edit-provider-form').trigger('submit.prevent')
    await flushPromises()
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials).toMatchObject(account.credentials)
    wrapper.unmount()
  })

  it.each(['go', 'zen'])('OpenCode %s 编辑保留规则顺序和模式', async mode => {
    const account = buildAccount()
    account.platform = 'opencode_go'
    account.credentials = { api_key: 'sk-opencode-test', provider_mode: mode, upstream_protocols: ['anthropic_messages', 'openai_responses', 'openai_chat_completions'], base_url: 'https://relay.example/v1', api_base_urls: { chat_completions: 'https://relay.example/v1', anthropic: 'https://relay.example/native', responses: 'https://relay.example/v1' }, protocol_rules: [{ pattern: 'gpt-*', protocol: 'chat_completions' }, { pattern: '*', protocol: 'anthropic' }] }
    updateAccountMock.mockReset().mockResolvedValue(account)
    checkMixedChannelRiskMock.mockReset().mockResolvedValue({ has_risk: false })
    const wrapper = mountModal(account)
    await wrapper.get('form#edit-provider-form').trigger('submit.prevent')
    await flushPromises()
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials).toMatchObject(account.credentials)
    wrapper.unmount()
  })


})
