import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpenAIOAuthClientPolicy } from '@/types'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_HTTP_BRIDGE,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  type OpenAIWSMode
} from '@/utils/openaiWsMode'

export type CodexFingerprintMode = 'off' | 'device' | 'session' | 'full'
export type RpmStrategy = 'tiered' | 'sticky_exempt'
export type AnthropicAPIKeyAuthScheme = 'x_api_key' | 'authorization_bearer'

// 以下选项在创建、编辑和批量编辑中含义一致，统一在这里维护文案与取值。

export function useCodexFingerprintModeOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: 'off' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintOff') },
    { value: 'device' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintDevice') },
    { value: 'session' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintSession') },
    { value: 'full' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintFull') }
  ])
}

export function useOpenAIWSModeOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: OPENAI_WS_MODE_OFF as OpenAIWSMode, label: t('admin.providers.openai.wsModeOff') },
    { value: OPENAI_WS_MODE_CTX_POOL as OpenAIWSMode, label: t('admin.providers.openai.wsModeCtxPool') },
    { value: OPENAI_WS_MODE_PASSTHROUGH as OpenAIWSMode, label: t('admin.providers.openai.wsModePassthrough') },
    { value: OPENAI_WS_MODE_HTTP_BRIDGE as OpenAIWSMode, label: t('admin.providers.openai.wsModeHttpBridge') }
  ])
}

export function useOpenAIOAuthClientPolicyOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: 'any' as OpenAIOAuthClientPolicy, label: t('admin.providers.openai.clientPolicyAny') },
    { value: 'codex_only' as OpenAIOAuthClientPolicy, label: t('admin.providers.openai.clientPolicyCodexOnly') },
    {
      value: 'tls_router_matched_only' as OpenAIOAuthClientPolicy,
      label: t('admin.providers.openai.clientPolicyTLSRouterMatchedOnly')
    }
  ])
}

export function useUserMsgQueueModeOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: '', label: t('admin.providers.quotaControl.rpmLimit.umqModeOff') },
    { value: 'throttle', label: t('admin.providers.quotaControl.rpmLimit.umqModeThrottle') },
    { value: 'serialize', label: t('admin.providers.quotaControl.rpmLimit.umqModeSerialize') }
  ])
}

export function useWebSearchEmulationOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: 'default', label: t('admin.providers.anthropic.webSearchDefault') },
    { value: 'enabled', label: t('admin.providers.anthropic.webSearchEnabled') },
    { value: 'disabled', label: t('admin.providers.anthropic.webSearchDisabled') }
  ])
}

export function useAnthropicAPIKeyAuthSchemeOptions() {
  const { t } = useI18n()
  return computed(() => [
    { value: 'x_api_key' as AnthropicAPIKeyAuthScheme, label: t('admin.providers.anthropic.apiKeyAuthSchemeXApiKey') },
    {
      value: 'authorization_bearer' as AnthropicAPIKeyAuthScheme,
      label: t('admin.providers.anthropic.apiKeyAuthSchemeBearer')
    }
  ])
}

export const CACHE_TTL_OVERRIDE_TARGET_OPTIONS = [
  { value: '5m', label: '5m' },
  { value: '1h', label: '1h' }
]
