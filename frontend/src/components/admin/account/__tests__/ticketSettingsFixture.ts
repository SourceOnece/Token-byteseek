import type { TicketAccountSettings } from '@/api/admin/codexTickets'

// 合成账号票据配置；父表单测试不连接真实账号、代理或模型。
export const ticketSettingsFixture = (id = 1): TicketAccountSettings => ({
  account_id: id, mode: 'on', verified_flow: false, global_enabled: true,
  watchdog_mode: 'inherit', effective_watchdog_mode: 'observe', effective_enabled: true,
  revision: 'r1', proxy_source: 'gateway', proxy_configured: false,
  rules: {
    models: ['gpt-6-astra'], target_length: 332, degraded_signal_length: 312,
    max_attempts: 3, concurrency: 4, cache_minutes: 60, refresh_before_minutes: 10,
    retry_interval_seconds: 1, probe_interval_seconds: 6, failure_threshold: 0, cooldown_seconds: 300
  },
  proxy_policy: {
    mode: 'fixed', dynamic_source: 'template', proxy_protocol: 'http',
    extraction_configured: false, fixed_proxy_id: '', proxies: []
  }
})
