import { describe, expect, it, vi } from 'vitest'
import { consumeQualityStream, qualityStats, qualitySchedulesAPI, type CodexQualityResult } from '../codexQuality'

const { deleteRequest } = vi.hoisted(() => ({ deleteRequest: vi.fn().mockResolvedValue({}) }))
vi.mock('../../client', () => ({ apiClient: { delete: deleteRequest }, buildApiUrl: (path: string) => path }))

describe('Codex quality SSE', () => {
  it('删除 API 指定单一计划并显式传递确认', async () => {
    await qualitySchedulesAPI.remove(7)
    expect(deleteRequest).toHaveBeenCalledWith('/admin/accounts/codex-quality-schedules/7', { data: { confirm_delete: true } })
  })
  it('保留逐字节 UTF-8 分块，不丢中文回答', async () => {
    const result = { account_id: 1, response_text: '满血答案', status: 'full' }
    const bytes = new TextEncoder().encode(`: heartbeat\n\ndata: ${JSON.stringify({ type: 'result', data: result })}\n\ndata: {"type":"complete"}\n\n`)
    const stream = new ReadableStream<Uint8Array>({ start(controller) {
      for (const byte of bytes) controller.enqueue(new Uint8Array([byte]))
      controller.close()
    } })
    const onResult = vi.fn()
    await consumeQualityStream(stream, onResult)
    expect(onResult).toHaveBeenCalledTimes(1)
    expect(onResult).toHaveBeenCalledWith(result)
  })
  it('断流不冒充整批完成', async () => {
    const stream = new ReadableStream<Uint8Array>({ start(controller) { controller.close() } })
    await expect(consumeQualityStream(stream, vi.fn())).rejects.toThrow('before completion')
  })
  it('失败计入分母，取消跳过与过时排除', () => {
    const results = ['full', 'degraded', 'failed', 'skipped', 'cancelled', 'stale'].map(status => ({ status }) as CodexQualityResult)
    expect(qualityStats(results)).toMatchObject({ full: 1, degraded: 1, failed: 1, tested: 3, rate: '33.3' })
    expect(qualityStats([]).rate).toBeNull()
  })
})
