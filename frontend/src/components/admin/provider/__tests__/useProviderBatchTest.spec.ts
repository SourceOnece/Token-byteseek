import { describe, expect, it, vi } from 'vitest'
import { useProviderBatchTest } from '../useProviderBatchTest'
import { executeProviderTest } from '../providerTestRun'

vi.mock('../providerTestRun', async () => ({
  ...await vi.importActual('../providerTestRun'),
  executeProviderTest: vi.fn(),
}))

// 真实关闭/重开会中止 fetch，但 promise 的结束时间不能假定早于新一轮开始。
describe('连接测试批次隔离', () => {
  it('旧批次迟到结束不覆盖新批次，也不访问已删除的排队行', async () => {
    let finishOld!: (value: boolean) => void
    let finishNew!: (value: boolean) => void
    vi.mocked(executeProviderTest)
      .mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { finishNew = resolve }))
    const batch = useProviderBatchTest(key => key)
    batch.concurrency.value = 1
    batch.reset(['old-a', 'old-b'])
    const options = { retry: false, providerId: 1, providerName: 'test', buildBody: (model: string) => ({ model_id: model, prompt: 'test', test_type: 'text' as const }) }
    const old = batch.start({ ...options, targets: ['old-a', 'old-b'] })
    batch.reset(['new'])
    const next = batch.start({ ...options, providerId: 2, targets: ['new'] })
    finishOld(false)
    await old
    expect(batch.running.value).toBe(true)
    expect(Object.keys(batch.rows.value)).toEqual(['new'])
    expect(batch.rows.value.new.state).toBe('running')
    finishNew(true)
    await next
    expect(batch.running.value).toBe(false)
    expect(executeProviderTest).toHaveBeenCalledTimes(2)
  })
})
