// core/utils/time.js - 时间格式化工具。
//
// 当前只暴露 formatRelative:把 ISO 时间字符串 / Date 转成 "3 分钟前" 这种相对表达。
// 用于卡片 hover 显示"最近一次扫描"等场景,不重复引入 dayjs / date-fns。

/**
 * 把时间转成相对表达。
 *
 *   formatRelative('2026-06-29T10:30:00Z')   // -> "3 分钟前"(取决于当前时间)
 *   formatRelative(new Date())                // -> "刚刚"
 *   formatRelative('')                        // -> ""
 *
 * @param {string|Date|number|null|undefined} input 任意 Date 构造器可接受的输入。
 * @returns {string} 不可解析时返空串;≤1 分钟返"刚刚";< 1 小时返 N 分钟;< 1 天返 N 小时;< 7 天返 N 天;其余返本地日期。
 */
export function formatRelative(input) {
  if (input == null || input === '') return ''
  const t = input instanceof Date ? input.getTime() : new Date(input).getTime()
  if (Number.isNaN(t)) return ''
  const diffMs = Date.now() - t
  // 容忍几秒时钟漂移,把"将来一点点"也算"刚刚"
  if (diffMs < 60_000) return '刚刚'
  const m = Math.floor(diffMs / 60_000)
  if (m < 60) return `${m} 分钟前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h} 小时前`
  const d = Math.floor(h / 24)
  if (d < 7) return `${d} 天前`
  return new Date(t).toLocaleDateString()
}