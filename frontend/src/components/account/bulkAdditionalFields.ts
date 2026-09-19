import type { Account, AccountPlatform, AccountType } from '@/types'

export interface BulkAdditionalField {
  key: string; label: string; en: string; section: 'account' | 'credentials' | 'extra'
  kind: 'text' | 'number' | 'boolean' | 'json' | 'secret' | 'datetime'
  platforms?: AccountPlatform[]; types?: AccountType[]; options?: string[]; min?: number; max?: number
}
const field = (section: BulkAdditionalField['section'], key: string, label: string, en: string, kind: BulkAdditionalField['kind'], rest: Partial<BulkAdditionalField> = {}): BulkAdditionalField => ({ section, key, label, en, kind, ...rest })
const apiTypes: AccountType[] = ['apikey','bedrock']
const cn: AccountPlatform[] = ['kimi','zhipu','deepseek','minimax','opencode_go']
// 与新增账号持久化字段逐项对应；身份密钥只写不回填，运行时快照不作为可编辑配置。
export const bulkAdditionalFields: BulkAdditionalField[] = [
  field('account','name','账号名称','Account name','text'),
  field('account','notes','备注','Notes','text'),
  field('account','expires_at','到期时间','Expiry','datetime'),
  field('account','auto_pause_on_expired','到期自动暂停','Pause on expiry','boolean'),
  field('account','schedulable','调度','Scheduling','boolean'),
  ...['quota_limit','quota_daily_limit','quota_weekly_limit'].map((key,i)=>field('extra',key,['总额度','每日额度','每周额度'][i],['Total quota','Daily quota','Weekly quota'][i],'number',{types:apiTypes,min:0})),
  field('extra','quota_daily_reset_mode','每日重置模式','Daily reset mode','text',{types:apiTypes,options:['rolling','fixed']}),
  field('extra','quota_weekly_reset_mode','每周重置模式','Weekly reset mode','text',{types:apiTypes,options:['rolling','fixed']}),
  field('extra','quota_daily_reset_hour','每日重置小时','Daily reset hour','number',{types:apiTypes,min:0,max:23}),
  field('extra','quota_weekly_reset_day','每周重置星期','Weekly reset day','number',{types:apiTypes,min:0,max:6}),
  field('extra','quota_weekly_reset_hour','每周重置小时','Weekly reset hour','number',{types:apiTypes,min:0,max:23}),
  field('extra','quota_reset_timezone','额度重置时区','Quota timezone','text',{types:apiTypes}),
  ...(['daily','weekly','total'] as const).flatMap((dimension,i)=>[
    field('extra',`quota_notify_${dimension}_enabled`,['每日额度通知','每周额度通知','总额度通知'][i],['Daily quota notification','Weekly quota notification','Total quota notification'][i],'boolean',{types:apiTypes}),
    field('extra',`quota_notify_${dimension}_threshold`,['每日通知阈值','每周通知阈值','总额度通知阈值'][i],['Daily notification threshold','Weekly notification threshold','Total notification threshold'][i],'number',{types:apiTypes,min:0}),
    field('extra',`quota_notify_${dimension}_threshold_type`,['每日阈值类型','每周阈值类型','总额度阈值类型'][i],['Daily threshold type','Weekly threshold type','Total threshold type'][i],'text',{types:apiTypes,options:['fixed','percentage']})
  ]),
  field('credentials','pool_mode','上游号池模式','Upstream pool mode','boolean',{types:apiTypes}),
  field('credentials','pool_mode_retry_count','同账号重试次数','Same-account retries','number',{types:apiTypes,min:0,max:10}),
  field('credentials','pool_mode_retry_status_codes','重试状态码','Retry status codes','json',{types:apiTypes}),
  field('credentials','temp_unschedulable_enabled','异常临时停调','Temporary scheduling pause','boolean'),
  field('credentials','temp_unschedulable_rules','临时停调规则','Temporary pause rules','json'),
  field('extra','upstream_usage_query','上游用量查询配置','Upstream usage query','json',{types:['apikey']}),
  field('extra','upstream_request_id_header','上游请求 ID 响应头','Upstream request ID header','text'),
  field('extra','images_url_to_b64_json','图片链接转 Base64','Image URL to Base64','boolean',{platforms:['openai'],types:['apikey']}),
  field('extra','anthropic_passthrough','Anthropic 透传','Anthropic passthrough','boolean',{platforms:['anthropic'],types:['apikey']}),
  field('extra','anthropic_apikey_auth_scheme','Anthropic 鉴权方式','Anthropic authentication','text',{platforms:['anthropic'],types:['apikey'],options:['x_api_key','authorization_bearer']}),
  field('extra','web_search_emulation','联网搜索兼容模式','Web search emulation','text',{platforms:['anthropic'],types:['apikey'],options:['default','disabled','enabled']}),
  field('extra','window_cost_limit','窗口费用限制','Window cost limit','number',{platforms:['anthropic'],min:0}),
  field('extra','window_cost_sticky_reserve','粘性窗口预留','Sticky window reserve','number',{platforms:['anthropic'],min:0,max:100}),
  field('extra','max_sessions','最大会话数','Maximum sessions','number',{platforms:['anthropic'],min:0}),
  field('extra','session_idle_timeout_minutes','会话空闲时间（分钟）','Session idle minutes','number',{platforms:['anthropic'],min:1}),
  field('extra','session_id_masking_enabled','会话 ID 掩码','Session ID masking','boolean',{platforms:['anthropic']}),
  field('extra','cache_ttl_override_enabled','覆盖缓存时长','Override cache TTL','boolean',{platforms:['anthropic']}),
  field('extra','cache_ttl_override_target','缓存时长','Cache TTL','text',{platforms:['anthropic'],options:['5m','1h']}),
  field('extra','custom_base_url_enabled','自定义上游地址开关','Custom upstream URL enabled','boolean',{platforms:['anthropic']}),
  field('extra','custom_base_url','自定义上游地址','Custom upstream URL','text',{platforms:['anthropic']}),
  field('extra','mixed_scheduling','混合调度','Mixed scheduling','boolean',{platforms:['antigravity']}),
  field('extra','allow_overages','允许超额','Allow overages','boolean',{platforms:['antigravity']}),
  field('credentials','project_id','项目 ID','Project ID','text',{platforms:['gemini']}),
  field('credentials','antigravity_project_id','Antigravity 项目 ID','Antigravity project ID','text',{platforms:['antigravity']}),
  field('credentials','tier_id','订阅等级','Subscription tier','text',{platforms:['gemini']}),
  field('credentials','provider_type','Gemini 来源','Gemini provider','text',{platforms:['gemini'],types:['apikey'],options:['official','third_party']}),
  field('credentials','location','Vertex 区域','Vertex location','text',{platforms:['gemini'],types:['service_account']}),
  field('credentials','service_account_json','服务账号 JSON','Service account JSON','secret',{platforms:['gemini'],types:['service_account']}),
  field('credentials','client_email','服务账号邮箱','Service account email','text',{platforms:['gemini'],types:['service_account']}),
  field('credentials','account_mode','账号套餐模式','Account plan mode','text',{platforms:cn,types:['apikey']}),
  field('credentials','api_protocol','上游协议','Upstream protocol','text',{platforms:cn,types:['apikey'],options:['adaptive','responses','chat_completions','anthropic']}),
  field('credentials','api_base_urls','多协议地址','Protocol base URLs','json',{platforms:cn,types:['apikey']}),
  field('credentials','protocol_rules','OpenCode 模型协议规则','OpenCode model protocol rules','json',{platforms:['opencode_go']}),
  field('credentials','zhipu_organization','智谱组织','Zhipu organization','text',{platforms:['zhipu']}),
  field('credentials','zhipu_project','智谱项目','Zhipu project','text',{platforms:['zhipu']}),
  field('credentials','auth_mode','Bedrock 鉴权方式','Bedrock authentication','text',{types:['bedrock'],options:['sigv4','apikey']}),
  field('credentials','aws_region','AWS 区域','AWS region','text',{types:['bedrock']}),
  field('credentials','aws_force_global','强制全局推理','Force global inference','text',{types:['bedrock'],options:['true','false']}),
  field('credentials','aws_access_key_id','AWS Access Key ID','AWS Access Key ID','secret',{types:['bedrock']}),
  field('credentials','aws_secret_access_key','AWS Secret Access Key','AWS Secret Access Key','secret',{types:['bedrock']}),
  field('credentials','aws_session_token','AWS Session Token','AWS Session Token','secret',{types:['bedrock']}),
  field('credentials','site','Qoder 站点','Qoder site','text',{platforms:['qoder'],options:['global','cn']}),
  field('credentials','machine_id','Qoder 机器 ID','Qoder machine ID','text',{platforms:['qoder']}),
  field('credentials','user_type','Qoder 用户类型','Qoder user type','text',{platforms:['qoder']}),
  field('credentials','uid','Qoder UID','Qoder UID','text',{platforms:['qoder']}),
  field('credentials','aid','Qoder AID','Qoder AID','text',{platforms:['qoder']}),
  field('credentials','pat','Qoder PAT','Qoder PAT','secret',{platforms:['qoder']}),
  field('credentials','security_oauth_token','Qoder 安全令牌','Qoder security token','secret',{platforms:['qoder']}),
  field('credentials','api_key','API Key','API Key','secret',{types:['apikey','bedrock','upstream']}),
  field('credentials','new_api_user_access_token','New API 钱包访问令牌','New API wallet token','secret',{types:['apikey']}),
  field('credentials','new_api_user_id','New API 钱包用户 ID','New API wallet user ID','text',{types:['apikey']}),
]

export function commonAccountValue(accounts: Account[], section: BulkAdditionalField['section'], key: string): unknown {
  if (!accounts.length) return ''
  const get=(a: Account)=>section==='account'?(a as unknown as Record<string,unknown>)[key]:a[section]?.[key]
  const value=get(accounts[0])
  return accounts.every(a=>JSON.stringify(get(a))===JSON.stringify(value)) ? value ?? '' : ''
}
