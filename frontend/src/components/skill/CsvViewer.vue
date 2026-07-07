<script setup>
// CsvViewer - 表格化 CSV 预览
//
// 2026-07-08 增:CSV 文件不再走"二进制兜底"或纯文本 monaco,而是解析成二维数组,
// 渲染成 <table>,首行作为表头加粗 + 斑马纹 + sticky 横向滚动条,大文件按需虚拟滚动。
//
// 数据流:后端把 csv 内容当 string 返回 → papaparse.parse(string, { skipEmptyLines })
// → 二维数组 [[cell, cell, ...], ...]。
//
// 性能:1MB csv 通常 ~ 10000 行,Vue 渲染 10000 个 <tr> 会卡,所以限制只渲染前 N=2000 行,
// 提示"仅预览前 N 行"。monaco 编辑模式不限行(虚拟滚动原生支持)。

import { computed, ref } from 'vue'
import Papa from 'papaparse'

const props = defineProps({
  content: { type: String, default: '' },
})

const PREVIEW_LIMIT = 2000

const parsed = computed(() => {
  const text = props.content || ''
  if (!text) return { rows: [], headers: [], truncated: false, total: 0 }
  // skipEmptyLines:true 跳过空行(常见 csv 末尾有空行)
  const result = Papa.parse(text, {
    skipEmptyLines: true,
    // 不自动检测 header,统一把第一行当 header,后面当 data
  })
  const data = (result.data || []).filter((row) => Array.isArray(row) && row.length > 0)
  const total = data.length
  const truncated = total > PREVIEW_LIMIT
  const rows = truncated ? data.slice(0, PREVIEW_LIMIT) : data
  const headers = rows[0] || []
  const body = rows.slice(1)
  return { rows, headers, body, truncated, total }
})

// 表格列宽:统一 160px,内容超长自动截断 + tooltip 显示完整
const COLUMN_WIDTH = 160
</script>

<template>
  <div class="cv-csv">
    <div v-if="parsed.truncated" class="cv-csv-banner">
      文件共 {{ parsed.total }} 行,仅预览前 {{ PREVIEW_LIMIT }} 行;编辑模式(monaco)看完整内容。
    </div>
    <div v-else-if="parsed.total === 0" class="cv-csv-empty">空文件或无可解析行</div>
    <div v-else class="cv-csv-table-wrap">
      <table class="cv-csv-table">
        <thead>
          <tr>
            <th class="cv-csv-rownum">#</th>
            <th
              v-for="(h, idx) in parsed.headers"
              :key="`h-${idx}`"
              :style="{ width: COLUMN_WIDTH + 'px', minWidth: COLUMN_WIDTH + 'px' }"
              :title="String(h)"
            >
              {{ h }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rIdx) in parsed.body" :key="`r-${rIdx}`">
            <td class="cv-csv-rownum">{{ rIdx + 1 }}</td>
            <td
              v-for="(cell, cIdx) in row"
              :key="`c-${rIdx}-${cIdx}`"
              :style="{ width: COLUMN_WIDTH + 'px', minWidth: COLUMN_WIDTH + 'px' }"
              :title="String(cell)"
            >
              {{ cell }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.cv-csv {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
}
.cv-csv-banner {
  flex-shrink: 0;
  padding: 8px 12px;
  background: #fef3c7;
  color: #92400e;
  border-bottom: 1px solid #fde68a;
  font-size: 12px;
}
.cv-csv-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-faint, #94a3b8);
  font-size: 13px;
  font-style: italic;
}
.cv-csv-table-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.cv-csv-table {
  border-collapse: separate;
  border-spacing: 0;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text, #0f172a);
}
.cv-csv-table th,
.cv-csv-table td {
  padding: 4px 10px;
  border-bottom: 1px solid var(--border, #e2e8f0);
  border-right: 1px solid var(--border, #e2e8f0);
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 240px;
}
.cv-csv-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: #f6f8fa;
  font-weight: 600;
  border-bottom: 2px solid var(--border, #e2e8f0);
}
.cv-csv-rownum {
  position: sticky;
  left: 0;
  z-index: 1;
  background: #f6f8fa;
  color: var(--text-faint, #94a3b8);
  text-align: right;
  width: 48px;
  min-width: 48px;
  user-select: none;
  font-weight: 500;
}
.cv-csv-table thead th.cv-csv-rownum { z-index: 3; }
.cv-csv-table tbody tr:nth-child(even) .cv-csv-rownum,
.cv-csv-table tbody tr:nth-child(even) td:not(.cv-csv-rownum) { background: #fafbfc; }
.cv-csv-table tbody tr:hover td { background: #f1f5f9; }
</style>