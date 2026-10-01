import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../RedeemView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('RedeemView responsive layout', () => {
  it('locks the desktop viewport and keeps history inside its own card', () => {
    // 宽屏锁定视口高度，历史在卡内滚动并分页，页面本身不再滚动。
    expect(viewSource).toContain('<AppLayout fit-viewport>')
    expect(viewSource).toContain(
      'grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[22.5rem_minmax(0,1fr)] lg:grid-rows-[auto_minmax(0,1fr)]'
    )
    expect(viewSource).toContain('lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:min-h-0')
    expect(viewSource).toContain('<div class="lg:min-h-0 lg:flex-1 lg:overflow-y-auto">')
    expect(viewSource).toContain('<Pagination')
    // 根容器位于 flex 列中，mx-auto 会让内容收缩并与页头错位。
    expect(viewSource).not.toMatch(/class="[^"]*\bmx-auto\b/)
    // 历史行使用分隔线列表，不再嵌套灰底卡片。
    expect(viewSource).toContain('divide-y divide-gray-100')
  })

  it('shows active subscriptions below the redeem panel on wide screens', () => {
    // 订阅概览复用顶栏弹层的列表组件，宽屏占据左栏剩余高度。
    expect(viewSource).toContain('data-testid="redeem-subscriptions"')
    expect(viewSource).toContain('lg:col-start-1 lg:row-start-2 lg:min-h-0')
    expect(viewSource).toContain('<SubscriptionUsageList')
  })

  it('omits supplementary redeem guidance', () => {
    // 页面只保留兑换操作与历史，避免再次加入重复说明或大小写提示。
    expect(viewSource).not.toContain("t('redeem.redeemCodeHint')")
    expect(viewSource).not.toContain("t('redeem.aboutCodes')")
    expect(viewSource).not.toContain('contactInfo')
    expect(viewSource).not.toContain('authAPI')
  })
})
