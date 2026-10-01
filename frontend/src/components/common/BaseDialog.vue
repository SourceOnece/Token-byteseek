<template>
  <Teleport to="body">
    <Transition name="modal" @after-leave="afterLeave">
      <div
        v-if="show"
        class="modal-overlay h-[100dvh] w-[100dvw] min-w-0 overflow-hidden"
        :style="zIndexStyle"
        :aria-labelledby="dialogId"
        role="dialog"
        aria-modal="true"
        @click.self="handleClose"
      >
        <!-- 动态视口单位避开移动端浏览器工具栏，vh/vw 规则由公共样式作为旧浏览器兜底。 -->
        <div
          ref="dialogRef"
          tabindex="-1"
          :class="['modal-content min-h-0 min-w-0 max-h-[95dvh] sm:max-h-[90dvh]', widthClasses]"
          @click.stop
        >
          <!-- 头部 -->
          <div class="modal-header min-w-0 max-w-full">
            <h3 :id="dialogId" class="modal-title min-w-0 break-words">
              {{ title }}
            </h3>
            <button
              v-if="showCloseButton"
              @click="emit('close')"
              class="-mr-2 border-2 border-transparent p-2 text-gray-600 transition-colors hover:border-gray-950 hover:bg-bh-red hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:text-dark-300 dark:hover:border-dark-100"
              aria-label="Close modal"
            >
              <Icon name="x" size="md" :stroke-width="2.5" />
            </button>
          </div>

          <!-- 内容区 -->
          <div
            ref="modalBodyRef"
            class="modal-body min-h-0 min-w-0 max-w-full"
            :class="{ 'modal-body-contained': !bodyScroll }"
          >
            <slot></slot>
          </div>

          <!-- 底部 -->
          <div v-if="$slots.footer" class="modal-footer min-w-0 max-w-full">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts">
let dialogIdCounter = 0
</script>

<script setup lang="ts">
import { computed, watch, onMounted, onUnmounted, ref, nextTick } from 'vue'
import { useDialogLifecycle } from '@/composables/useDialogLifecycle'
import Icon from '@/components/icons/Icon.vue'
import { Z_INDEX } from '@/constants/overlay'

// 生成唯一ID以避免多个对话框时ID冲突
const dialogId = `modal-title-${++dialogIdCounter}`

// 焦点管理
const dialogRef = ref<HTMLElement | null>(null)
const modalBodyRef = ref<HTMLElement | null>(null)

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'

interface Props {
  show: boolean
  title: string
  width?: DialogWidth
  bodyScroll?: boolean
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
  zIndex?: number
}

interface Emits {
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  width: 'normal',
  bodyScroll: true,
  closeOnEscape: true,
  closeOnClickOutside: false,
  showCloseButton: true,
  zIndex: Z_INDEX.MODAL
})

const emit = defineEmits<Emits>()

// 自定义层级会覆盖 CSS 中默认的 z-50。
const zIndexStyle = computed(() => {
  return props.zIndex !== Z_INDEX.MODAL ? { zIndex: props.zIndex } : undefined
})

const widthClasses = computed(() => {
  // 移动端统一保留遮罩边距，避免弹窗内容的最小宽度撑开页面。
  const widths: Record<DialogWidth, string> = {
    narrow: 'max-w-[calc(100vw-1rem)] sm:max-w-md',
    normal: 'max-w-[calc(100vw-1rem)] sm:max-w-lg',
    wide: 'max-w-[calc(100vw-1rem)] sm:max-w-2xl md:max-w-3xl lg:max-w-4xl',
    'extra-wide': 'max-w-[calc(100vw-1rem)] sm:max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl',
    full: 'max-w-[calc(100vw-1rem)] sm:max-w-4xl md:max-w-5xl lg:max-w-6xl xl:max-w-7xl'
  }
  return widths[props.width]
})

const handleClose = () => {
  if (props.closeOnClickOutside) {
    emit('close')
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (props.show && isTop() && props.closeOnEscape && event.key === 'Escape') {
    emit('close')
  }
}

// 嵌套弹窗只让顶层响应关闭，滚动锁与焦点在退出完成后释放。
const { afterLeave, isTop } = useDialogLifecycle(() => props.show, dialogRef)
watch(() => props.show, async (open) => {
  if (!open) return
  await nextTick()
  if (props.show && modalBodyRef.value) modalBodyRef.value.scrollTop = 0
})

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
/* 分页表单自行管理滚动，外壳只分配标题和按钮之间的剩余高度。 */
.modal-body-contained {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
