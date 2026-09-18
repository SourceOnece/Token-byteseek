import { onMounted, onBeforeUnmount, ref, watch, type Ref, type WatchStopHandle } from 'vue'
import { getCodexTicketStatus, type TicketAccountStatus } from '@/api/admin/codexTickets'
import { isQualityTestable } from '@/components/admin/account/codexQualityPresentation'
import type { Account } from '@/types'

export const supportsCodexTickets = (account: Account) => account.type === 'oauth' && isQualityTestable(account)

// 单个页面共用轮询与时钟，虚拟表格每行不单独计时/请求；分页销毁时取消旧响应。
export function useCodexTicketStatus(accounts: Ref<Account[]>) {
  const ticketStatus = ref<Record<number, TicketAccountStatus>>({})
  const ticketLoadFailed = ref(false)
  const ticketNow = ref(Date.now())
  let controller: AbortController | null = null
  let version = 0
  let stopWatch: WatchStopHandle | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  let anchor = Date.now(), anchorMono = performance.now(), lastRead = 0
  let destroyed = false

  async function refresh() {
    if (controller || destroyed || document.hidden) return
    const ids = accounts.value.filter(supportsCodexTickets).map(a => a.id)
    if (!ids.length) return
    const current = new AbortController(), request = ++version
    controller = current
    try {
      const next: Record<number, TicketAccountStatus> = {}
      let serverTime = 0
      // 大页分批读取，最多一个批次在途，避免每行请求或无界并发。
      for (let i = 0; i < ids.length; i += 100) {
        const data = await getCodexTicketStatus(ids.slice(i, i + 100), current.signal)
        if (current.signal.aborted || request !== version || destroyed) return
        serverTime = Date.parse(data.server_time)
        if (!Number.isFinite(serverTime) || !Array.isArray(data.items)) throw new Error('Invalid ticket status')
        for (const row of data.items) next[row.account_id] = row
      }
      // 已被删除或未返回的账号不能无限显示“读取中”，也不能沿用上页的有效票。
      for (const id of ids) if (!next[id]) next[id] = {
        account_id: id, eligible: true, collection_paused: false,
        models: ['gpt-6-astra', 'gpt-5.6-sol'].map(model => ({ model, state: 'unavailable' }))
      }
      ticketStatus.value = next
      anchor = serverTime; anchorMono = performance.now(); lastRead = anchorMono
      ticketNow.value = serverTime
      ticketLoadFailed.value = false
    } catch {
      if (!current.signal.aborted && request === version && !destroyed) ticketLoadFailed.value = true
    } finally { if (request === version) controller = null }
  }
  const visible = () => { if (!document.hidden) { if (lastRead && performance.now() - lastRead > 45000) ticketLoadFailed.value = true; void refresh() } }
  onMounted(() => {
    stopWatch = watch(() => accounts.value.map(a => `${a.id}:${a.updated_at}:${a.schedulable}`).join(','), () => {
      version++; controller?.abort(); controller = null
      ticketStatus.value = {}; ticketLoadFailed.value = false; lastRead = 0
      void refresh()
    }, { immediate: true })
    let ticks = 0
    timer = setInterval(() => {
      ticketNow.value = anchor + performance.now() - anchorMono
      if (++ticks % 15 === 0) void refresh()
      if (lastRead && performance.now() - lastRead > 45000) ticketLoadFailed.value = true
    }, 1000)
    document.addEventListener('visibilitychange', visible)
  })
  onBeforeUnmount(() => {
    destroyed = true; version++; controller?.abort(); stopWatch?.()
    clearInterval(timer); document.removeEventListener('visibilitychange', visible)
  })
  return { ticketStatus, ticketLoadFailed, ticketNow, refreshTicketStatus: refresh }
}
