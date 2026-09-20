import { describe, expect, it } from 'vitest'
import { ticketDiagnosticText, ticketReasonText } from '../codexTicketDiagnostic'

const t=(key:string, values?:Record<string,unknown>)=>key+(values?JSON.stringify(values):'')

describe('票据失败诊断',()=>{
  it('取消、超时、保护期明确区分，未知原因不冒充账号变化',()=>{
    for(const reason of ['timeout','task_timeout','round_timeout','client_disconnected','lease_lost','config_unavailable','account_rate_limited','proxy_timeout','transport_timeout']) expect(ticketReasonText(t,reason)).toContain('.'+reason)
    expect(ticketReasonText(t,'http://user:secret@private')).toContain('.unknown')
    expect(ticketReasonText(t,'http://user:secret@private')).not.toContain('secret')
  })
  it('明确上一轮、失败阶段、耗时/预算、取号HTTP与安全网络类别',()=>{
    const text=ticketDiagnosticText(t,{proxy_id:'provider',proxy_name:'API',attempt:41,header_length:0,header_present:false,prefix_valid:false,phase:'verify',timeout_seconds:120,elapsed_ms:120050,network_kind:'timeout',previous_attempt:true,provider_status:429})
    expect(text).toContain('previousAttempt')
    expect(text).toContain('phases.verify')
    expect(text).toContain('120.0')
    expect(text).toContain('429')
    expect(text).not.toContain('cancelled')
  })
})
