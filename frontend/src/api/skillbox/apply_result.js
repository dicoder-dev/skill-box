// 2026-07-03 增:统一 apply / batch 响应判定工具。
//
// 背景:Service.Apply / Service.BatchApply 设计为"宽容"语义——逐 tool 失败不阻断
// 其他 tool,即便有失败仍 return out, nil → controller 永远 200。
// 前端 await applySkill 不抛异常,只 toast.success 把失败静默吞掉。
//
// 这里抽出 inspectApplyResult,把响应规整成 { allOk, partial, failedItems },
// 供调用方判断后弹 partialFailed / allOk 两种 toast。
//
// 用法:
//   import { inspectApplyResult } from '@/api/skillbox/apply_result.js'
//   const res = await applySkill({ ... })
//   const ins = inspectApplyResult(res)
//   if (ins.allOk) toast.success(t('skills.apply.allOk', { n: ... }))
//   else toast.error(t('skills.apply.partialFailed', { ok, total, detail }), 6000)

const MAX_DETAIL = 5 // toast 失败明细最多展示条数,超出截断

/**
 * 解析 apply / batch 响应,提取失败条目。
 *
 * @param {object} res ApplyResult(name/version/applies[]/all_ok) | BatchOutput(items[]/all_ok)
 *                       | market.pull 的 res.apply_result 嵌套对象
 * @returns {{ allOk: boolean, partial: boolean, failedItems: Array<{tool:string, error:string}> }}
 */
export function inspectApplyResult(res) {
  // 兼容三种形态:
  //   1) 单 apply:  res.applies = [{tool, target_path, status, error}]
  //   2) 批量:      res.items   = [{tool, ..., result: {status, error}, error}]
  //   3) market.pull: res.apply_result.applies (嵌套)
  const items = res?.applies || res?.items || res?.apply_result?.applies || []
  const failed = []
  for (const it of items) {
    // 批量条目 result 是嵌套 .result,单 apply 条目 status/error 在自身
    const inner = it?.result || it
    const isFailed = inner?.status === 'failed' || (it?.error && it.error.length)
    if (!isFailed) continue
    failed.push({
      tool: inner?.tool || it?.tool || '(unknown)',
      error: inner?.error || it?.error || '',
    })
  }
  // outerAllOK 兼容嵌套 res.apply_result.all_ok
  const outerAllOK = res?.all_ok ?? res?.apply_result?.all_ok
  return {
    allOk: !!outerAllOK && failed.length === 0,
    partial: !!res?.partial_failure || (!outerAllOK && failed.length > 0),
    failedItems: failed,
  }
}

/**
 * 把失败条目格式化成可读的多行 detail(toast 用)。
 * 超出 MAX_DETAIL 自动截断,末尾追加 "... (N more)"。
 *
 * @param {Array<{tool:string, error:string}>} failedItems
 * @returns {string}
 */
export function formatFailedDetail(failedItems) {
  if (!failedItems?.length) return ''
  const head = failedItems.slice(0, MAX_DETAIL).map((f) => `  • ${f.tool}: ${f.error}`).join('\n')
  if (failedItems.length > MAX_DETAIL) {
    return head + `\n  ... (${failedItems.length - MAX_DETAIL} more)`
  }
  return head
}