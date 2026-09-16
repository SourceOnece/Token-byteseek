import { describe, expect, it } from 'vitest'
import { applyOpenCodeGoProtocolRules, cloneOpenCodeGoProtocolRules, defaultCNAdaptiveBaseUrls, defaultOpenCodeProtocolRules, validOpenCodeGoProtocolRules } from '../credentialsBuilder'

describe('OpenCode 凭据规则', () => {
  it.each(['zen', 'go'] as const)('正确生成 %s 模式三个端点', mode => {
    const base = mode === 'zen' ? 'https://opencode.ai/zen' : 'https://opencode.ai/zen/go'
    expect(defaultCNAdaptiveBaseUrls('opencode_go', mode)).toEqual({ chat_completions: `${base}/v1`, responses: `${base}/v1`, anthropic: base })
  })
  it('创建时显式空规则不能被替换为默认规则', () => {
    const credentials = { existing: 'preserved' }
    applyOpenCodeGoProtocolRules(credentials, [], 'create')
    expect(credentials).toEqual({ existing: 'preserved', protocol_rules: [] })
  })
  it('每次复制规则相互隔离，默认 Go 与 Zen 不混用', () => {
    const go = cloneOpenCodeGoProtocolRules(defaultOpenCodeProtocolRules('go'))
    const zen = cloneOpenCodeGoProtocolRules(defaultOpenCodeProtocolRules('zen'))
    expect(go.some(rule => rule.pattern === 'minimax-*')).toBe(true)
    expect(zen.some(rule => rule.pattern === 'claude-*')).toBe(true)
    go[0].pattern = 'changed'
    expect(defaultOpenCodeProtocolRules('go')[0].pattern).toBe('grok-*')
  })
  it('拒绝空名称、多通配及超长规则', () => {
    for (const pattern of ['', '*gpt', 'gpt**', 'gpt\n5', 'x'.repeat(129)]) {
      expect(validOpenCodeGoProtocolRules([{ pattern, protocol: 'responses' }])).toBe(false)
    }
    expect(validOpenCodeGoProtocolRules([])).toBe(true)
    expect(validOpenCodeGoProtocolRules([{ pattern: 'gpt-*', protocol: 'responses' }])).toBe(true)
  })
})
