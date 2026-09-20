import { describe, expect, it } from 'vitest'
import { commonAccountValue } from '../bulkAccountValues'
import type { Account } from '@/types'

describe('原批量配置共同值', () => {
  it('保留零值和关闭值，差异留空且不依赖更多账号配置', () => {
    const rows = [{ id: 1, concurrency: 0, extra: { enabled: false }, credentials: { base_url: 'https://a.invalid' } }, { id: 2, concurrency: 0, extra: { enabled: false }, credentials: { base_url: 'https://b.invalid' } }] as Account[]
    expect(commonAccountValue(rows, 'account', 'concurrency')).toBe(0)
    expect(commonAccountValue(rows, 'extra', 'enabled')).toBe(false)
    expect(commonAccountValue(rows, 'credentials', 'base_url')).toBe('')
    expect(commonAccountValue([], 'account', 'concurrency')).toBe('')
  })
})
