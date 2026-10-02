import { computed, ref } from 'vue'
import {
  createProviderTestRun,
  executeProviderTest,
  type ProviderTestRequestBody,
  type ProviderTestRun
} from './providerTestRun'

/** 一批最多测试的模型数，避免误触发大量真实请求。 */
export const MAX_BATCH_MODELS = 50
/** 可选的并发数，数值越大越容易触发上游限流。 */
export const BATCH_CONCURRENCY_OPTIONS = [1, 2, 3] as const

export type ProviderBatchRowState = 'queued' | 'running' | 'success' | 'failed' | 'stopped'

export interface ProviderBatchRow {
  model: string
  state: ProviderBatchRowState
  run: ProviderTestRun
}

type Translate = (key: string, params?: Record<string, unknown>) => string

/**
 * 批量模型测试的状态与调度：按并发数依次取模型执行连接测试。
 * 停止只会让后续模型不再开始，已发出的请求继续等待结果；关闭弹窗时才中止全部请求。
 */
export function useProviderBatchTest(t: Translate) {
  const baseModels = ref<string[]>([])
  const customModels = ref<string[]>([])
  const selected = ref(new Set<string>())
  const rows = ref<Record<string, ProviderBatchRow>>({})
  const running = ref(false)
  const stopping = ref(false)
  const concurrency = ref(2)
  const detailModel = ref<string | null>(null)
  let stopRequested = false
  let controller: AbortController | null = null
  let generation = 0

  const models = computed(() => [...new Set([...baseModels.value, ...customModels.value])])
  const entries = computed(() => Object.values(rows.value))
  const doneCount = computed(() => entries.value.filter((row) => row.state === 'success' || row.state === 'failed').length)
  const successCount = computed(() => entries.value.filter((row) => row.state === 'success').length)
  const failedModels = computed(() => entries.value.filter((row) => row.state === 'failed').map((row) => row.model))
  const stoppedCount = computed(() => entries.value.filter((row) => row.state === 'stopped').length)
  const progress = computed(() =>
    entries.value.length ? (100 * (doneCount.value + stoppedCount.value)) / entries.value.length : 0
  )
  const detailRow = computed(() => (detailModel.value ? rows.value[detailModel.value] ?? null : null))

  // 打开弹窗或切换提供商时，默认选中前 MAX_BATCH_MODELS 个模型。
  function reset(nextModels: string[]) {
    abort()
    // 旧请求可能晚于新弹窗返回，失效批次不得清理新批次的队列或运行标记。
    generation++
    controller = null
    running.value = false
    stopping.value = false
    baseModels.value = nextModels
    customModels.value = []
    selected.value = new Set(nextModels.slice(0, MAX_BATCH_MODELS))
    rows.value = {}
    detailModel.value = null
  }

  function toggle(model: string, checked: boolean) {
    const next = new Set(selected.value)
    if (checked && next.size < MAX_BATCH_MODELS) next.add(model)
    if (!checked) next.delete(model)
    selected.value = next
  }

  function setMany(targets: string[], checked: boolean) {
    const next = new Set(selected.value)
    for (const model of targets) {
      if (!checked) next.delete(model)
      else if (next.size < MAX_BATCH_MODELS) next.add(model)
    }
    selected.value = next
  }

  function clearSelection() {
    selected.value = new Set()
  }

  // 列表里没有的模型 ID 作为自定义项加入并选中。
  function addModel(model: string) {
    if (!model || models.value.includes(model)) return
    customModels.value = [...customModels.value, model]
    toggle(model, true)
  }

  async function start(options: {
    targets: string[]
    retry: boolean
    providerId: number
    providerName: string
    buildBody: (model: string) => ProviderTestRequestBody
  }) {
    if (running.value || options.targets.length === 0) return
    const currentGeneration = ++generation
    stopRequested = false
    stopping.value = false
    running.value = true
    detailModel.value = null
    controller = new AbortController()
    const signal = controller.signal

    const queued: Record<string, ProviderBatchRow> = {}
    for (const model of options.targets) {
      queued[model] = { model, state: 'queued', run: createProviderTestRun() }
    }
    rows.value = options.retry ? { ...rows.value, ...queued } : queued

    const targets = [...options.targets]
    let index = 0
    const worker = async () => {
      while (generation === currentGeneration && !stopRequested && index < targets.length) {
        const row = rows.value[targets[index++]]
        row.state = 'running'
        const finished = await executeProviderTest({
          providerId: options.providerId,
          providerName: options.providerName,
          body: options.buildBody(row.model),
          run: row.run,
          signal,
          t
        })
        if (generation !== currentGeneration) return
        if (!finished) {
          row.state = 'stopped'
        } else {
          row.state = row.run.status === 'success' ? 'success' : 'failed'
        }
      }
    }

    try {
      await Promise.all(Array.from({ length: Math.min(concurrency.value, targets.length) }, worker))
      if (generation !== currentGeneration) return
      // 停止后尚未开始的模型标记为未执行。
      for (; index < targets.length; index++) {
        rows.value[targets[index]].state = 'stopped'
      }
    } finally {
      if (generation === currentGeneration) {
        running.value = false
        stopping.value = false
        controller = null
      }
    }
  }

  // showDetail 传 null 时回到模型列表。
  function showDetail(model: string | null) {
    detailModel.value = model
  }

  function stop() {
    if (!running.value) return
    stopRequested = true
    stopping.value = true
  }

  // abort 立即中止所有进行中的请求，用于关闭弹窗。
  function abort() {
    stopRequested = true
    controller?.abort()
  }

  return {
    models,
    selected,
    rows,
    running,
    stopping,
    concurrency,
    detailModel,
    detailRow,
    entries,
    doneCount,
    successCount,
    failedModels,
    stoppedCount,
    progress,
    reset,
    toggle,
    setMany,
    clearSelection,
    addModel,
    showDetail,
    start,
    stop,
    abort
  }
}

export type ProviderBatchTest = ReturnType<typeof useProviderBatchTest>
