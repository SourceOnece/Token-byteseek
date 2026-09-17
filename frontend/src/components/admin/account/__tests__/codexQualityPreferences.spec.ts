import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultQualityPreferences, loadQualityPreferences, saveQualityPreferences } from '../codexQualityPreferences'

describe('批量题目测试配置记忆', () => {
  beforeEach(() => { localStorage.clear(); vi.restoreAllMocks() })
  it('只持久化七个配置项，管理员互相隔离，未知身份不存储', () => {
    const input = { ...defaultQualityPreferences(), model: 'custom-model', protocol: 'chat_completions', effort: 'max', prompt: '题目\n第二行', keyword: '答案', concurrency: 5, timeoutSeconds: 600, account_ids: [99], confirmed: true }
    saveQualityPreferences(7, input)
    const { account_ids: _ids, confirmed: _confirmed, ...expected } = input
    expect(loadQualityPreferences(7)).toEqual(expected)
    expect(localStorage.getItem('byteseek:codex-quality-preferences:v1:7')).not.toContain('confirmed')
    expect(loadQualityPreferences(8)).toEqual(defaultQualityPreferences())
    saveQualityPreferences(undefined, input)
    expect(localStorage.length).toBe(1)
  })
  it('损坏缓存和非法字段回退，显式沿用账号协议及空题目不会丢失', () => {
    const key = 'byteseek:codex-quality-preferences:v1:7'
    localStorage.setItem(key, '{bad')
    expect(loadQualityPreferences(7)).toEqual(defaultQualityPreferences())
    localStorage.setItem(key, JSON.stringify({ model: 'bad\nmodel', effort: 'fake', protocol: '', prompt: '', keyword: 'valid', concurrency: 9, timeoutSeconds: 0 }))
    expect(loadQualityPreferences(7)).toEqual({ ...defaultQualityPreferences(), protocol: '', keyword: 'valid' })
  })
  it('禁止读取/配额用尽时不抛出异常', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota') })
    expect(loadQualityPreferences(7)).toEqual(defaultQualityPreferences())
    expect(() => saveQualityPreferences(7, defaultQualityPreferences())).not.toThrow()
  })
})
