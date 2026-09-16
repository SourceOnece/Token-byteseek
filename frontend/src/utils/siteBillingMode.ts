// 销售模式不改变已有订阅、余额与历史用量，仅组合购买入口开关。
export type SiteBillingMode = 'recharge_and_subscription' | 'recharge_only' | 'subscription_only'
export interface BillingModeSettings {
  subscription_enabled?: boolean
  payment_balance_disabled?: boolean
}

export function resolveSiteBillingMode(settings: BillingModeSettings | null | undefined): SiteBillingMode {
  if (settings?.subscription_enabled === false) return 'recharge_only'
  if (settings?.payment_balance_disabled === true) return 'subscription_only'
  return 'recharge_and_subscription'
}

export function billingModeToSettings(mode: SiteBillingMode): Required<BillingModeSettings> {
  switch (mode) {
    case 'recharge_only': return { subscription_enabled: false, payment_balance_disabled: false }
    case 'subscription_only': return { subscription_enabled: true, payment_balance_disabled: true }
    default: return { subscription_enabled: true, payment_balance_disabled: false }
  }
}

export function purchaseLabelKey(settings: BillingModeSettings | null | undefined): string {
  switch (resolveSiteBillingMode(settings)) {
    case 'recharge_only': return 'nav.recharge'
    case 'subscription_only': return 'nav.subscribe'
    default: return 'nav.buySubscription'
  }
}
