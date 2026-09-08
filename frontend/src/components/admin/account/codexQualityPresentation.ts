import type { QualityStatus } from '@/api/admin/codexQuality'

// 状态同时有文字和配色，失败不伪装为关键词未命中。
export function qualityStatusClass(status: QualityStatus) {
  if (status === 'full') return 'text-bh-blue dark:text-blue-300'
  if (status === 'degraded') return 'text-bh-red dark:text-red-400'
  if (status === 'failed') return 'text-yellow-700 dark:text-bh-yellow'
  return 'text-gray-500 dark:text-gray-300'
}
