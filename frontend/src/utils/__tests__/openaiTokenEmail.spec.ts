import { describe, expect, it } from 'vitest'
import { openaiTokenEmail, openaiImportEmails } from '../openaiTokenEmail'
const token = (email: string) => `head.${btoa(JSON.stringify({ 'https://api.openai.com/profile': { email } }))}.sig`
describe('OpenAI email display', () => {
  it('识别命名空间邮箱，支持 auth.json 和逐行 AT', () => {
    expect(openaiTokenEmail(token('one@example.invalid'))).toBe('one@example.invalid')
    expect(openaiImportEmails(JSON.stringify({ tokens: { access_token: token('one@example.invalid') } }))).toEqual(['one@example.invalid'])
    expect(openaiImportEmails(`${token('one@example.invalid')}\n${token('two@example.invalid')}`)).toHaveLength(2)
  })
  it('不把 RT 或恶意文本当邮箱', () => {
    expect(openaiTokenEmail('opaque-refresh-token')).toBe('')
    expect(openaiTokenEmail(token('<script>@x.com'))).toBe('')
  })
})
