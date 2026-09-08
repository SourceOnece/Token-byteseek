// 仅在管理员浏览器内解析展示字段，不上传令牌、不校验签名、不作为登录依据。
export function openaiTokenEmail(token: unknown): string {
  if (typeof token !== 'string' || token.length > 128 * 1024) return ''
  try {
    const parts = token.trim().split('.')
    if (parts.length !== 3) return ''
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const bytes = Uint8Array.from(atob(payload.padEnd(Math.ceil(payload.length / 4) * 4, '=')), value => value.charCodeAt(0))
    const data = JSON.parse(new TextDecoder().decode(bytes))
    const email = data.email || data['https://api.openai.com/profile']?.email
    return typeof email === 'string' && email.length <= 254 && /^[^\s<>@]+@[^\s<>@]+\.[^\s<>@]+$/.test(email) ? email : ''
  } catch { return '' }
}

export function openaiImportEmails(input: string): string[] {
  if (input.length > 1024 * 1024) return []
  const emails = new Set<string>()
  const walk = (value: unknown, depth = 0) => {
    if (depth > 6 || emails.size >= 100) return
    if (typeof value === 'string') { const email = openaiTokenEmail(value); if (email) emails.add(email); return }
    if (Array.isArray(value)) { value.slice(0, 500).forEach(item => walk(item, depth + 1)); return }
    if (value && typeof value === 'object') {
      const record = value as Record<string, unknown>
      for (const key of ['access_token', 'accessToken', 'id_token', 'tokens', 'accounts']) walk(record[key], depth + 1)
    }
  }
  try { walk(JSON.parse(input)) } catch { input.split(/\s+/).slice(0, 500).forEach(value => walk(value)) }
  return [...emails]
}
