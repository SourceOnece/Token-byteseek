import { describe, expect, it } from 'vitest'
import { getKeyGroupProvider } from '../keyGroupProviders'
import type { GroupPlatform } from '@/types'

describe('密钥供应商分类', () => {
  it.each([
    ['anthropic', 'anthropic'], ['openai', 'openai'],
    ['deepseek', 'domestic'], ['kimi', 'domestic'], ['zhipu', 'domestic'], ['minimax', 'domestic'],
    ['qoder', 'other'], ['gemini', 'other'], ['grok', 'other'], ['antigravity', 'other'], ['opencode_go', 'other']
  ])('%s 保留本地十一平台分类', (platform, expected) => {
    expect(getKeyGroupProvider(platform as GroupPlatform)).toBe(expected)
  })
})
