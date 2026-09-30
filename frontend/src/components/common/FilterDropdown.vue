<template>
  <div ref="dropdownRef" class="relative shrink-0" @keydown.esc.stop.prevent="closeWithFocus">
    <button
      ref="triggerRef"
      type="button"
      class="btn btn-secondary relative btn-icon"
      :aria-label="t('common.filter')"
      :title="t('common.filter')"
      :aria-expanded="open"
      @click="toggle"
    >
      <Icon name="filter" size="sm" />
      <span v-if="activeCount" class="absolute -right-1 -top-1 inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-primary-100 px-1.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
        {{ activeCount }}
      </span>
    </button>

    <Teleport to="body"><Transition name="dropdown-fade">
      <div
        v-if="open"
        ref="panelRef"
        :inert="!open || undefined"
        class="dropdown overflow-auto p-4"
        :style="panelStyle"
        @click.stop
        @keydown.esc.stop.prevent="closeWithFocus"
      >
        <div class="mb-3 flex items-center justify-between">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('common.filter') }}</div>
          <button v-if="activeCount" type="button" class="text-xs font-medium text-primary-600 dark:text-primary-400" @click="emit('reset')">
            {{ t('common.reset') }}
          </button>
        </div>
        <div class="space-y-3">
          <slot />
        </div>
      </div>
    </Transition></Teleport>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getFloatingPanelPosition } from '@/utils/floatingPanel'
import { Z_INDEX } from '@/constants/overlay'

// wide 用于条件较多、需要两列排列的面板，按可用空间决定展开方向。
const props = defineProps<{ activeCount: number; wide?: boolean }>()
const emit = defineEmits<{ (event: 'reset'): void }>()
const { t } = useI18n()
const open = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const panelStyle = ref<CSSProperties>({})

// 加宽面板默认从按钮左缘向右展开，右侧放不下时改为与按钮右缘对齐向左展开。
function updateAlignment() {
  const rect = dropdownRef.value?.getBoundingClientRect()
  if (!rect) return
  const rem = Number.parseFloat(getComputedStyle(document.documentElement).fontSize) || 16
  const position = getFloatingPanelPosition(rect, innerWidth, innerHeight, { maxWidth: (props.wide ? 34 : 18) * rem, align: 'left' })
  panelStyle.value = { position: 'fixed', zIndex: Z_INDEX.TELEPORT_DROPDOWN,
    left: `${position.left}px`, width: `${position.width}px`, maxHeight: `${position.maxHeight}px`,
    top: position.top === null ? undefined : `${position.top}px`, bottom: position.bottom === null ? undefined : `${position.bottom}px` }
}

function toggle() {
  open.value = !open.value
  if (open.value) updateAlignment()
}

// 键盘关闭后回到筛选按钮，便于继续操作工具栏。
function closeWithFocus() {
  open.value = false
  triggerRef.value?.focus()
}

// Select 的 Teleport 面板会阻止点击冒泡，选择条件时不会误关外层筛选面板。
function closeOutside(event: MouseEvent) {
  if (event.target instanceof Node && !dropdownRef.value?.contains(event.target) && !panelRef.value?.contains(event.target)) {
    open.value = false
  }
}

onMounted(() => { document.addEventListener('click', closeOutside); window.addEventListener('resize', updateAlignment) })
onBeforeUnmount(() => { document.removeEventListener('click', closeOutside); window.removeEventListener('resize', updateAlignment) })
</script>
