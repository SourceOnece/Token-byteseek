import { afterEach, describe, expect, it, vi } from 'vitest'
import { runTicketCollection } from '../codexTickets'
vi.mock('@/api/client', () => ({ apiClient: {}, buildApiUrl: (path: string) => path }))
afterEach(() => vi.unstubAllGlobals())

// 按字节切分中文/CRLF，断流不冒充完成，POST 不重发。
describe('ticket collection stream', () => {
  it('跨分块 UTF-8 事件逐条交付', async () => {
    const bytes = new TextEncoder().encode('data: {"type":"attempt","data":{"email":"测试"}}\r\n\r\ndata: {"type":"complete","data":{"status":"completed"}}\r\n\r\n')
    const stream = new ReadableStream<Uint8Array>({ start(controller) { for (const byte of bytes) controller.enqueue(Uint8Array.of(byte)); controller.close() } })
    const fetchMock = vi.fn().mockResolvedValue(new Response(stream, { headers: { 'Content-Type': 'text/event-stream' } })); vi.stubGlobal('fetch', fetchMock)
    const events: unknown[] = []
    await runTicketCollection([1], 'revision', new AbortController().signal, (kind, data) => events.push({ kind, data }))
    expect(events).toEqual([{ kind: 'attempt', data: { email: '测试' } }, { kind: 'complete', data: { status: 'completed' } }]); expect(fetchMock).toHaveBeenCalledOnce()
  })
  it('没有 complete 时明确报中断，不重试有副作用的请求', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('data: {"type":"start","data":{}}\n\n', { headers: { 'Content-Type': 'text/event-stream' } })); vi.stubGlobal('fetch', fetchMock)
    await expect(runTicketCollection([1], 'revision', new AbortController().signal, () => {})).rejects.toThrow('before completion'); expect(fetchMock).toHaveBeenCalledOnce()
  })
})
