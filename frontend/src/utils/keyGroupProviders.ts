import type { ProviderPlatform as GroupPlatform, ProtocolID } from '@/types'

// 跨平台分组按真实准入协议分类；同一分组可同时支持多种客户端。
export type KeyGroupProtocol = 'anthropic' | 'openai' | 'gemini' | 'other'
export const KEY_GROUP_PROTOCOLS = ['anthropic', 'openai', 'gemini', 'other'] as const
export const KEY_GROUP_PROTOCOL_LABELS = { anthropic: 'Messages', openai: 'Responses / Chat', gemini: 'Gemini', other: 'Images / Audio' }
export const KEY_GROUP_PROTOCOL_ICONS: Record<KeyGroupProtocol, GroupPlatform[]> = {
  anthropic: ['anthropic'], openai: ['openai'], gemini: ['gemini'], other: ['grok']
}
export function getKeyGroupProtocols(protocols: ProtocolID[]): KeyGroupProtocol[] {
  const result = new Set<KeyGroupProtocol>()
  for (const protocol of protocols) {
    if (protocol === 'anthropic_messages') result.add('anthropic')
    else if (['openai_responses', 'openai_responses_websocket', 'openai_chat_completions'].includes(protocol)) result.add('openai')
    else if (protocol === 'gemini_generate_content') result.add('gemini')
    else result.add('other')
  }
  return [...result]
}

export type KeyGroupProvider = 'anthropic' | 'openai' | 'domestic' | 'other'
export const KEY_GROUP_PROVIDERS = ['anthropic', 'openai', 'domestic', 'other'] as const

// 分类只依赖真实平台，不使用可由管理员重命名的分组名称或展示品牌。
const PROVIDER_BY_PLATFORM: Record<GroupPlatform, KeyGroupProvider> = {
  anthropic: 'anthropic',
  openai: 'openai',
  kimi: 'domestic',
  zhipu: 'domestic',
  deepseek: 'domestic',
  minimax: 'domestic',
  gemini: 'other',
  grok: 'other',
  antigravity: 'other',
  qoder: 'other',
  opencode_go: 'other'
}

export function getKeyGroupProvider(platform: GroupPlatform): KeyGroupProvider {
  return PROVIDER_BY_PLATFORM[platform] ?? 'other'
}

// 集合沿用现有供应商图标，不添加虚构品牌。
export const KEY_GROUP_PROVIDER_ICONS: Record<KeyGroupProvider, GroupPlatform[]> = {
  anthropic: ['anthropic'],
  openai: ['openai'],
  domestic: ['deepseek', 'kimi'],
  other: ['gemini', 'grok']
}
