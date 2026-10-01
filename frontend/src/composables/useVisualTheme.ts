import { readonly, ref } from 'vue'

/** 页面视觉皮肤；明暗模式由 useTheme 独立维护。 */
export type VisualTheme = 'tokenflux' | 'bauhaus'

const visualTheme = ref<VisualTheme>('tokenflux')

function normalize(value: unknown): VisualTheme {
  return value === 'bauhaus' ? 'bauhaus' : 'tokenflux'
}

function applyVisualTheme(next: VisualTheme) {
  visualTheme.value = next
  const root = document.documentElement
  root.dataset.visualTheme = next
  root.classList.toggle('theme-bauhaus', next === 'bauhaus')
}

// 初始值来自服务端 HTML 注入，旧浏览器个人偏好不再覆盖站点配置。
export function initVisualTheme(siteTheme?: unknown) {
  applyVisualTheme(normalize(siteTheme))
}

// 仅由公开站点设置的应用流程更新，管理表单保存前不会改变生效主题。
export function setVisualTheme(siteTheme: unknown) {
  const next = normalize(siteTheme)
  if (next === visualTheme.value) return
  document.documentElement.classList.add('theme-switching')
  applyVisualTheme(next)
  window.requestAnimationFrame(() => {
    window.requestAnimationFrame(() => document.documentElement.classList.remove('theme-switching'))
  })
}

export function useVisualTheme() {
  return {
    visualTheme: readonly(visualTheme),
  }
}
