/**
 * Redeem code API endpoints
 * Handles redeem code redemption for users
 */

import { apiClient } from './client'
import type { RedeemCode, RedeemCodeRequest, PaginatedResponse } from '@/types'

export type RedeemHistoryItem = RedeemCode

/**
 * Redeem a code
 * @param code - Redeem code string
 * @returns Redemption result with updated balance or concurrency
 */
export async function redeem(code: string): Promise<RedeemCode> {
  const payload: RedeemCodeRequest = { code }

  const { data } = await apiClient.post<RedeemCode>('/redeem', payload)

  return data
}

/**
 * Get user's redemption history
 * @returns List of redeemed codes
 */
export async function getHistory(): Promise<RedeemHistoryItem[]> {
  const { data } = await apiClient.get<RedeemHistoryItem[]>('/redeem/history')
  return data
}

// 新分页方法不改变已有无参数调用的数组契约。
export async function getHistoryPage(page = 1, pageSize = 20): Promise<PaginatedResponse<RedeemHistoryItem>> {
  const { data } = await apiClient.get<PaginatedResponse<RedeemHistoryItem>>('/redeem/history', { params: { page, page_size: pageSize } })
  return data
}

export const redeemAPI = {
  redeem,
  getHistory,
  getHistoryPage
}

export default redeemAPI
