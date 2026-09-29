import { describe, expect, it } from 'vitest'
import type { Group } from '@/types'
import { availableClients, buildClientConfig, modelsForProtocol, groupForKeyConfig, type ClientKind } from '../clientConfig'

describe('客户端配置', () => {
  const base = { model: 'custom-model', baseUrl: 'https://api.example.com/v1/', apiKey: 'secret', shell: 'unix' as const }
  it('只保留入口协议可请求的模型，通配规则不伪装为模型', () => {
    expect(modelsForProtocol({ models: ['a', 'b', '*'], model_protocols: { a: ['anthropic_messages'], b: ['openai_responses'] } }, 'openai_responses')).toEqual(['b'])
    expect(modelsForProtocol({ models: ['real', 'custom*'] }, 'anthropic_messages')).toEqual(['real'])
  })
  it('只有媒体入口时不显示 OpenCode', () => expect(availableClients(['image_batches'])).toEqual([]))
  it.each<ClientKind>(['claude', 'codex', 'gemini', 'grok', 'opencode'])('%s 使用明确模型，不附加专用平台路由', client => {
    const files = buildClientConfig({ ...base, client, protocol: 'openai_responses' })
    expect(files.length).toBeGreaterThan(0)
    expect(files.map(file => file.content).join('\n')).toContain('custom-model')
    expect(files.map(file => file.content).join('\n')).not.toContain('/antigravity')
  })
  it('Codex 默认使用 auth.json，显式认证与 WS 可独立打开', () => {
    const original = buildClientConfig({ ...base, client: 'codex', protocol: 'openai_responses' })
    expect(JSON.parse(original[1].content)).toEqual({ OPENAI_API_KEY: 'secret' })
    const direct = buildClientConfig({ ...base, client: 'codex', protocol: 'openai_responses', directAuth: true, websocket: true })
    expect(direct).toHaveLength(1)
    expect(direct[0].content).toContain('experimental_bearer_token = "secret"')
    expect(direct[0].content).toContain('responses_websockets_v2 = true')
    expect(direct[0].content).toContain('https://api.example.com/v1')
    expect(direct[0].content).not.toContain('/v1/v1')
  })
  it('OpenCode 根据入口协议选择 adapter', () => {
    const [file] = buildClientConfig({ ...base, client: 'opencode', protocol: 'gemini_generate_content' })
    const config = JSON.parse(file.content)
    expect(config.provider.tokenrouter.npm).toBe('@ai-sdk/google')
    expect(config.provider.tokenrouter.options.baseURL).toBe('https://api.example.com/v1beta')
    expect(Object.keys(config.provider.tokenrouter.models)).toEqual(['custom-model'])
  })
  it('Grok 配置保持 Responses 后端及用量端点', () => {
    const files = buildClientConfig({ ...base, client: 'grok', protocol: 'openai_responses' })
    expect(files[0].content).toContain('XAI_API_KEY')
    expect(files[1].content).toContain('api_backend = "responses"')
    expect(files[1].content).toContain('[model."custom-model"]')
  })
  it('shell 和 TOML 对特殊字符进行转义', () => {
    const [unix] = buildClientConfig({ ...base, client: 'gemini', protocol: 'gemini_generate_content', apiKey: "key'$(id)" })
    expect(unix.content).toContain("'key'\"'\"'$(id)'")
    const [ps] = buildClientConfig({ ...base, client: 'gemini', protocol: 'gemini_generate_content', apiKey: "key'$(id)", shell: 'powershell' })
    expect(ps.content).toContain("'key''$(id)'")
    const [toml] = buildClientConfig({ ...base, model: 'a"b', client: 'codex', protocol: 'openai_responses' })
    expect(toml.content).toContain('model = "a\\"b"')
  })
  it('Key 别名使用一跳目标的协议资格，不枚举通配符或无效目标', () => {
    const group = { models: ['gpt-real', 'claude-real'], model_protocols: { 'gpt-real': ['openai_responses'], 'claude-real': ['anthropic_messages'] } } as Group
    const result = groupForKeyConfig(group, { 'alias': 'gpt-real', 'gpt-*': 'claude-real', 'claude-real': 'missing', 'unavailable': 'missing', '*': 'missing' })!
    expect(result.models).toEqual(['gpt-real', 'alias'])
    expect(result.model_protocols).toEqual({ 'gpt-real': ['anthropic_messages'], alias: ['openai_responses'] })
    expect(group.models).toEqual(['gpt-real', 'claude-real'])
  })
  it('空模型和通配符不能生成客户端配置', () => {
    expect(buildClientConfig({ ...base, model: '', client: 'codex', protocol: 'openai_responses' })).toEqual([])
    expect(buildClientConfig({ ...base, model: '*', client: 'codex', protocol: 'openai_responses' })).toEqual([])
  })
})
