import { ref, watch } from 'vue'

/** 按需创建弹窗，并把组件卸载延后到外壳退出完成，避免提前截断动画。 */
export function useLeavingPresence(visible: () => boolean) {
  const present = ref(visible())
  watch(visible, (open) => {
    if (open) present.value = true
  }, { flush: 'sync' })
  function afterLeave() {
    if (!visible()) present.value = false
  }
  return { present, afterLeave }
}
