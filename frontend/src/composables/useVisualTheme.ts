import { ref } from 'vue'

/** 页面视觉皮肤；明暗模式由 useTheme 独立维护。 */
export type VisualTheme = 'tokenflux' | 'bauhaus'

const STORAGE_KEY = 'visual-theme'
const visualTheme = ref<VisualTheme>('tokenflux')

function normalize(value: string | null): VisualTheme {
  return value === 'bauhaus' ? 'bauhaus' : 'tokenflux'
}

function applyVisualTheme(next: VisualTheme) {
  visualTheme.value = next
  const root = document.documentElement
  root.dataset.visualTheme = next
  root.classList.toggle('theme-bauhaus', next === 'bauhaus')
}

export function initVisualTheme() {
  let saved: string | null = null
  try { saved = localStorage.getItem(STORAGE_KEY) } catch { /* 禁用存储时仍可使用默认皮肤。 */ }
  applyVisualTheme(normalize(saved))
}

export function setVisualTheme(next: VisualTheme) {
  if (next === visualTheme.value) return
  document.documentElement.classList.add('theme-switching')
  applyVisualTheme(next)
  try { localStorage.setItem(STORAGE_KEY, next) } catch { /* 只在当前页面生效。 */ }
  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => document.documentElement.classList.remove('theme-switching'))
  })
}

export function useVisualTheme() {
  return {
    visualTheme,
    setVisualTheme,
  }
}
