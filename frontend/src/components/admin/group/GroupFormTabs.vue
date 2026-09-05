<template>
  <div ref="rootRef" class="group-form-tabs" @onboarding-reveal="onTourReveal">
    <div class="group-tab-list" role="tablist" :aria-label="t('admin.groups.tabs.label')">
      <button
        v-for="tab in visibleTabs"
        :id="`${idPrefix}-tab-${tab}`"
        :key="tab"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab"
        :aria-controls="`${idPrefix}-panel-${tab}`"
        :tabindex="activeTab === tab ? 0 : -1"
        :data-group-tab-button="tab"
        class="group-tab"
        :class="{ 'group-tab-active': activeTab === tab }"
        @click="selectTab(tab)"
        @keydown="onTabKeydown($event, tab)"
      >
        {{ t(`admin.groups.tabs.${tab}`) }}
      </button>
    </div>
    <div ref="contentRef" class="group-tab-content">
      <!-- 所有页签持续挂载，避免定价条目和推理规则的内部草稿在切页时丢失。 -->
      <section
        v-for="tab in allTabs"
        v-show="activeTab === tab && visibleTabs.includes(tab)"
        :id="`${idPrefix}-panel-${tab}`"
        :key="tab"
        role="tabpanel"
        :aria-labelledby="`${idPrefix}-tab-${tab}`"
        :data-group-tab="tab"
        tabindex="0"
        class="group-tab-panel space-y-5"
      >
        <slot :name="tab" />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ platform: string; idPrefix: string }>()
const { t } = useI18n()
const allTabs = ['general', 'platform', 'pricing', 'protocol'] as const
type GroupFormTab = typeof allTabs[number]
const activeTab = ref<GroupFormTab>('general')
const rootRef = ref<HTMLElement | null>(null)
const contentRef = ref<HTMLElement | null>(null)
const visibleTabs = computed(() => allTabs.filter(tab =>
  tab !== 'platform' || ['anthropic', 'openai', 'gemini', 'antigravity'].includes(props.platform),
))

watch(visibleTabs, tabs => {
  if (!tabs.includes(activeTab.value)) activeTab.value = 'general'
})

// 回顶在 DOM 更新后的同一刷新周期完成，先于 revealElement 的 nextTick 定位。
watch(activeTab, () => {
  if (contentRef.value) contentRef.value.scrollTop = 0
}, { flush: 'post' })

async function selectTab(tab: GroupFormTab) {
  activeTab.value = tab
  await nextTick()
  rootRef.value?.querySelector<HTMLElement>(`[data-group-tab-button="${tab}"]`)
    ?.scrollIntoView?.({ block: 'nearest', inline: 'nearest' })
}

function onTabKeydown(event: KeyboardEvent, tab: GroupFormTab) {
  const tabs = visibleTabs.value
  const index = tabs.indexOf(tab)
  let next: GroupFormTab | undefined
  if (event.key === 'ArrowRight') next = tabs[(index + 1) % tabs.length]
  if (event.key === 'ArrowLeft') next = tabs[(index + tabs.length - 1) % tabs.length]
  if (event.key === 'Home') next = tabs[0]
  if (event.key === 'End') next = tabs[tabs.length - 1]
  if (!next) return
  event.preventDefault()
  void selectTab(next)
  rootRef.value?.querySelector<HTMLElement>(`[data-group-tab-button="${next}"]`)?.focus()
}

// 原生校验和创建引导共用定位流程，先展示页签，再滚动及聚焦目标。
async function revealElement(element: HTMLElement, focus = true) {
  const tab = element.closest<HTMLElement>('[data-group-tab]')?.dataset.groupTab as GroupFormTab | undefined
  if (!tab || !visibleTabs.value.includes(tab)) return
  activeTab.value = tab
  element.dispatchEvent(new Event('form-field-reveal', { bubbles: true }))
  await nextTick()
  element.scrollIntoView?.({ block: 'center', inline: 'nearest' })
  if (focus) {
    const selector = 'input, textarea, button, select, [tabindex]'
    const target = element.matches(selector) ? element
      : element.querySelector<HTMLElement>(selector) ?? element.parentElement?.querySelector<HTMLElement>(selector)
    target?.focus({ preventScroll: true })
  }
}

async function revealField(selector: string) {
  const element = rootRef.value?.querySelector<HTMLElement>(selector)
  if (element) await revealElement(element)
}

async function validate(): Promise<boolean> {
  const fields = rootRef.value?.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(
    'input, textarea, select',
  )
  // 不依赖浏览器提交时自动聚焦，隐藏页签中的无效输入也能正常报告。
  const invalid = fields && Array.from(fields).find(field => field.willValidate && !field.validity.valid)
  if (!invalid) return true
  await revealElement(invalid)
  invalid.reportValidity()
  return false
}

function onTourReveal(event: Event) {
  if (event.target instanceof HTMLElement) void revealElement(event.target, false)
}

defineExpose({ validate, revealField })
</script>

<style scoped>
.group-form-tabs {
  display: flex;
  height: min(68dvh, 760px);
  max-height: calc(90dvh - 180px);
  min-height: 0;
  min-width: 0;
  flex-direction: column;
}

.group-tab-list {
  @apply flex shrink-0 gap-2 overflow-x-auto p-1 pb-2;
  border-bottom: 3px solid var(--bh-ink);
}

.group-tab {
  @apply inline-flex shrink-0 items-center gap-2 whitespace-nowrap px-4 py-2.5 text-sm font-extrabold;
  border: 2px solid var(--bh-ink);
  background: var(--bh-surface);
  color: var(--bh-ink);
  box-shadow: var(--bh-shadow-sm);
  transition: translate 150ms ease, box-shadow 150ms ease, background-color 150ms ease;
}

.group-tab-active {
  background: var(--bh-yellow);
  color: #141414;
}

/* 几何标记只承担装饰，页签语义继续由原有 ARIA 和文字表达。 */
.group-tab::before {
  content: '';
  width: 10px;
  height: 10px;
  flex: none;
  background: var(--bh-red);
}

.group-tab[data-group-tab-button='platform']::before {
  background: var(--bh-blue);
  border-radius: 50%;
}

.group-tab[data-group-tab-button='pricing']::before {
  background: var(--bh-red);
  clip-path: polygon(50% 0, 100% 100%, 0 100%);
}

.group-tab[data-group-tab-button='protocol']::before { background: var(--bh-blue); }

.group-tab:focus-visible {
  outline: 2px solid var(--bh-blue);
  outline-offset: 2px;
}

.group-tab:active {
  translate: 2px 2px;
  box-shadow: 2px 2px 0 var(--bh-shadow-ink);
}

@media (hover: hover) {
  .group-tab:hover { translate: -1px -1px; }
}

.group-tab-content {
  @apply min-h-0 min-w-0 flex-1 overflow-y-auto overscroll-contain pl-1 pr-2 pb-2 pt-5;
}

/* 仅省略普通分区首块的重复边线，独立包豪斯卡片保留完整边框。 */
.group-tab-panel :deep(> :first-child:not(.bh-policy-section):not(.bh-policy-card)) {
  border-top: 0;
  margin-top: 0;
  padding-top: 0;
}

@media (max-width: 639px) {
  .group-form-tabs { max-height: calc(95dvh - 180px); }
  /* 四个入口在手机上全部可见，长英文文案仍可在各自格子内换行。 */
  .group-tab-list { @apply grid grid-cols-2; }
  .group-tab { @apply min-w-0 whitespace-normal px-3 text-left; }
}

@media (prefers-reduced-motion: reduce) {
  .group-tab { transition: none; }
}
</style>
