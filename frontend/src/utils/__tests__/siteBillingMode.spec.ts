import { describe, expect, it } from 'vitest'
import { billingModeToSettings, purchaseLabelKey, resolveSiteBillingMode, type SiteBillingMode } from '../siteBillingMode'

describe('销售模式保留历史默认', () => {
  it('缺省仍提供充值和订阅', () => {
    expect(resolveSiteBillingMode(undefined)).toBe('recharge_and_subscription')
    expect(purchaseLabelKey({})).toBe('nav.buySubscription')
  })
  it.each<SiteBillingMode>(['recharge_and_subscription', 'recharge_only', 'subscription_only'])('显式选择 %s 才成对改写', mode => {
    expect(resolveSiteBillingMode(billingModeToSettings(mode))).toBe(mode)
  })
  it('读取两个入口都关闭的旧设置不会原地修改', () => {
    const existing = { subscription_enabled: false, payment_balance_disabled: true }
    resolveSiteBillingMode(existing)
    expect(existing).toEqual({ subscription_enabled: false, payment_balance_disabled: true })
  })
})
