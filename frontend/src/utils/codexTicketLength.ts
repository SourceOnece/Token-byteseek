// 所有票据长度显示复用同一语义：异常红优先，合格绿，其它黄；仅复验响应允许0绿。
export function codexTicketLengthColor(actual: number, target: number, signal = false, allowEmpty = false) {
  if (signal) return 'text-bh-red dark:text-red-400'
  return actual === target || (allowEmpty && actual === 0)
    ? 'text-emerald-700 dark:text-emerald-400'
    : 'text-yellow-700 dark:text-bh-yellow'
}
