import type { ObjectDirective } from 'vue'

interface RevealState {
  animation?: Animation
  media: MediaQueryList
  stop: () => void
}

const states = new WeakMap<HTMLElement, RevealState>()

/** 内容立即可用；动画只改变透明度，不改变定位、挂载或请求时机。 */
function reveal(element: HTMLElement) {
  const state = states.get(element)
  state?.stop()
  if (!state || state.media.matches || !element.animate) return
  const style = getComputedStyle(element)
  const duration = Number.parseFloat(style.getPropertyValue('--motion-fast'))
  if (!Number.isFinite(duration) || duration <= 0) return
  const animation = element.animate([{ opacity: 0 }, { opacity: 1 }], {
    duration,
    easing: style.getPropertyValue('--motion-ease').trim() || 'ease-out',
  })
  state.animation = animation
  // 完成后释放动画，不留下会覆盖后续主题或显隐状态的填充样式。
  void animation.finished.then(() => {
    if (state.animation === animation) state.animation = undefined
  }, () => {})
}

// @project-doc docs/architecture/frontend_ui_conventions.md#ui_motion
export const vContentReveal: ObjectDirective<HTMLElement, unknown> = {
  mounted(element, binding) {
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    const state: RevealState = {
      media,
      stop: () => {
        state.animation?.cancel()
        state.animation = undefined
      },
    }
    states.set(element, state)
    media.addEventListener('change', state.stop)
    if (binding.value !== false) reveal(element)
  },
  updated(element, binding) {
    if (Object.is(binding.value, binding.oldValue)) return
    if (binding.value === false) states.get(element)?.stop()
    else reveal(element)
  },
  beforeUnmount(element) {
    const state = states.get(element)
    state?.stop()
    state?.media.removeEventListener('change', state.stop)
    states.delete(element)
  },
}
