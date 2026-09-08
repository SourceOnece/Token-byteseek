import type { QualityStatus } from '@/api/admin/codexQuality'
import type { Account } from '@/types'

// 与服务端检测资格保持一致；不把影子或 Agent Identity 显示为待检测账号。
export function isQualityTestable(account: Account) {
  return account.platform === 'openai' && ['oauth', 'apikey'].includes(account.type) && !account.parent_account_id &&
    (account.type !== 'oauth' || String(account.credentials?.auth_mode || '').trim().toLowerCase() !== 'agentidentity')
}

export function qualityProtocolOptions(legacyLabel: string) {
  return [{ value: 'responses', label: 'Responses' }, { value: 'chat_completions', label: 'Chat Completions' }, { value: '', label: legacyLabel }]
}
export function qualityProtocolLabel(value?: string) {
  return value === 'chat_completions' ? 'Chat Completions' : value === 'responses' ? 'Responses' : ''
}

// 状态同时有文字和配色，失败不伪装为关键词未命中。
export function qualityStatusClass(status: QualityStatus) {
  if (status === 'full') return 'text-bh-blue dark:text-blue-300'
  if (status === 'degraded') return 'text-bh-red dark:text-red-400'
  if (status === 'failed') return 'text-yellow-700 dark:text-bh-yellow'
  return 'text-gray-500 dark:text-gray-300'
}
