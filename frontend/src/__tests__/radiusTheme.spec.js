import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

import tailwindConfig from '../../tailwind.config.js'

// 固定全站圆角契约：工具类一律经 var() 引用 :root 变量,数值只在 style.css 维护一份。
const styleCss = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../styles/visual-palette.css'),
  'utf8'
)

describe('ByteSeek 包豪斯直角主题', () => {
  const radius = tailwindConfig.theme.borderRadius

  it('提供紧凑、控件、表面和弹窗四级语义令牌(经 CSS 变量引用)', () => {
    expect(radius).toMatchObject({
      compact: 'var(--radius-compact)',
      control: 'var(--radius-control)',
      surface: 'var(--radius-surface)',
      dialog: 'var(--radius-dialog)'
    })
  })

  it('所选包豪斯皮肤统一直角，默认 TokenFlux 保留原生 token', () => {
    expect(styleCss).toContain('--radius-compact: 0px')
    expect(styleCss).toContain('--radius-control: 0px')
    expect(styleCss).toContain('--radius-surface: 0px')
    expect(styleCss).toContain('--radius-dialog: 0px')
  })

  it('旧尺度兼容 key 已删除(由 check:ui 门禁阻止复活)', () => {
    for (const legacy of ['DEFAULT', 'sm', 'md', 'lg', 'xl', '2xl', '3xl', '4xl']) {
      expect(radius).not.toHaveProperty(legacy)
    }
  })

  it('保留无圆角和全圆工具类', () => {
    expect(radius.none).toBe('0px')
    expect(radius.full).toBe('9999px')
  })
})
