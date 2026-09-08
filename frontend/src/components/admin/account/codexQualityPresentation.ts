import type { QualityStatus } from '@/api/admin/codexQuality'

// 状态同时有文字和配色，失败不伪装为关键词未命中。
export function qualityStatusClass(status: QualityStatus) {
  if (status === 'full') return 'bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300'
  if (status === 'degraded' || status === 'failed') return 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}
