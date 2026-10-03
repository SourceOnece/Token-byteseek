import { readStorageWithLegacyKey } from './storage'

export const LOGIN_AGREEMENT_STORAGE_KEY = 'sub2api_login_agreement_consent'

// 登录协议按 revision 记录，条款更新后旧确认不会继续放行快捷登录。
export function hasAcceptedLoginAgreement(revision: string): boolean {
  if (!revision || typeof window === 'undefined') return false
  try {
    const raw = readStorageWithLegacyKey(window.localStorage, LOGIN_AGREEMENT_STORAGE_KEY, 'tokenrouter_login_agreement_consent')
    if (!raw) return false
    const parsed = JSON.parse(raw) as { revision?: string }
    return parsed.revision === revision
  } catch {
    return false
  }
}

// 撤回时写入否定记录，防止下次读取另一品牌键中的旧确认。
export function revokeLoginAgreement(): void {
  try { localStorage.setItem(LOGIN_AGREEMENT_STORAGE_KEY, '{}') } catch { /* 存储受限时不阻断界面。 */ }
}
