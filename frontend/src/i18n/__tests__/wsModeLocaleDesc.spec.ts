import { describe, expect, it } from 'vitest'

import en from '../locales/en/admin/providers'
import zh from '../locales/zh/admin/providers'

describe('OpenAI WS mode locale descriptions', () => {
  it('documents the global v2 router requirement for provider WS modes', () => {
    expect(zh.providers.openai.wsModeDesc).toContain('mode_router_v2_enabled')
    expect(zh.providers.openai.wsModeDesc).toContain('http_bridge')
    expect(en.providers.openai.wsModeDesc).toContain('mode_router_v2_enabled')
    expect(en.providers.openai.wsModeDesc).toContain('http_bridge')
  })
})
