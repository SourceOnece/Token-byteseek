type FormControl = HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement
const disabledControls = new WeakMap<Element, FormControl[]>()

/** 退出画面不再参与交互和原生表单校验，避免待移除的必填字段阻止提交。 */
export function isolateLeavingElement(element: Element) {
  element.setAttribute('inert', '')
  if (disabledControls.has(element)) return
  const controls = Array.from(element.querySelectorAll<FormControl>('input, textarea, select, button'))
    .filter(control => !control.disabled)
  disabledControls.set(element, controls)
  controls.forEach(control => { control.disabled = true })
}

/** 退出被新一次展开取消时，只恢复由动效暂时禁用的控件。 */
export function restoreEnteringElement(element: Element) {
  element.removeAttribute('inert')
  if (element instanceof HTMLElement) element.style.removeProperty('--motion-list-width')
  disabledControls.get(element)?.forEach(control => { control.disabled = false })
  disabledControls.delete(element)
}

/** 列表项退出脱离文档流前固定当前宽度，避免多列表单或标签在退出时压缩。 */
export function prepareListLeave(element: Element) {
  if (element instanceof HTMLElement) {
    element.style.setProperty('--motion-list-width', `${element.getBoundingClientRect().width}px`)
  }
  isolateLeavingElement(element)
}
