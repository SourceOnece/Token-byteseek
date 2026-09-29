import type { Account } from '@/types'

// 原有批量字段仍展示共同实际值；不同值留空，不依赖已撤回的扩展字段目录。
export function commonAccountValue(accounts: Account[], section: 'account' | 'credentials' | 'extra', key: string): unknown {
  if (!accounts.length) return ''
  const get = (account: Account) => section === 'account'
    ? (account as unknown as Record<string, unknown>)[key]
    : account[section]?.[key]
  const value = get(accounts[0])
  return accounts.every(account => JSON.stringify(get(account)) === JSON.stringify(value)) ? value ?? '' : ''
}
