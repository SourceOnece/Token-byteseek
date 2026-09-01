import { describe, expect, it } from 'vitest'

import tailwindConfig from '../../tailwind.config.js'

// 固定包豪斯全站直角契约，避免新增组件重新引入圆角。
describe('ByteSeek 包豪斯几何主题', () => {
  const radius = tailwindConfig.theme.borderRadius

  it('将紧凑、控件、表面和弹窗令牌统一为直角', () => {
    expect(radius).toMatchObject({
      compact: '0px',
      control: '0px',
      surface: '0px',
      dialog: '0px'
    })
  })

  it('将兼容工具类统一为直角并保留 full 圆形', () => {
    expect(radius).toMatchObject({
      DEFAULT: '0px',
      sm: '0px',
      md: '0px',
      lg: '0px',
      xl: '0px',
      '2xl': '0px',
      '3xl': '0px',
      '4xl': '0px'
    })
  })

  it('保留无圆角和全圆工具类', () => {
    expect(radius.none).toBe('0px')
    expect(radius.full).toBe('9999px')
  })
})
