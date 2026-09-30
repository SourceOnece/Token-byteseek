/** OpenAI 套餐标识与展示名称；其它平台保持独立标签。 */
export function normalizePlanType(value?: string | null): string {
  return (value || '').trim().toLowerCase().replace(/[\s_-]+/g, '')
}

/** 按产品标识去重，不能仅按显示标签合并不同套餐。 */
export function openAIPlanTypeKey(value?: string | null): string {
  const key = normalizePlanType(value)
  return key === 'chatgptpro' ? 'pro' : key
}

// 来自 sub2api 0.2.11 固定列表，上游记录来源为 Codex b1e72963。
export const openAIPlanTypes = [
  'free', 'go', 'plus', 'prolite', 'pro', 'promax', 'team',
  'self_serve_business_usage_based', 'self_serve_business_prolite', 'business',
  'enterprise', 'ent26', 'enterprise_cbp_usage_based', 'enterprise_cbp_automation',
  'edu', 'edu_plus', 'edu_pro', 'unknown'
] as const

/** 状态标签保留具体套餐，与统计使用的套餐家族名称区分。 */
export function openAIPlanTypeLabel(value?: string | null, display: 'status' | 'analytics' = 'status'): string {
  switch (openAIPlanTypeKey(value)) {
    case 'free': return 'Free'
    case 'go': return 'Go'
    case 'plus': return 'Plus'
    case 'prolite': return 'Pro 100'
    case 'pro': return 'Pro 200'
    case 'promax': return 'Pro 500'
    case 'team':
    case 'selfservebusinessusagebased': return 'Business'
    case 'business': return display === 'status' ? 'Enterprise' : 'Business'
    case 'selfservebusinessprolite': return display === 'status' ? 'Business Premium' : 'Business'
    case 'enterprisecbpautomation': return display === 'status' ? 'Enterprise (Automation)' : 'Enterprise'
    case 'enterprise':
    case 'ent26':
    case 'enterprisecbpusagebased': return 'Enterprise'
    case 'edu': return display === 'status' ? 'Edu' : 'Education'
    case 'eduplus': return display === 'status' ? 'Edu Plus' : 'Education'
    case 'edupro': return display === 'status' ? 'Edu Pro' : 'Education'
    case 'unknown': return display === 'status' ? 'Unknown' : 'Account'
    default: return ''
  }
}
