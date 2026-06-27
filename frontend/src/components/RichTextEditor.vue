<!--
  RichTextEditor.vue
  基于 Tiptap 的所见即所得编辑器(简化版,无图片上传/无 caption 扩展)
  - 输入输出都是 markdown 字符串(与现有 body 字段保持一致)
  - 内部:markdown → HTML 喂给 Tiptap;onUpdate 时 HTML → markdown 回写
  - 适用场景:技能首页 body 编辑、新建/编辑弹窗 body 编辑
-->
<script setup>
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import { Icon } from '@iconify/vue'
import { renderMarkdown } from '@/core/utils/markdown.js'
import { htmlToMarkdown } from '@/core/utils/html_to_markdown.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '开始输入内容...' },
  minHeight: { type: String, default: '320px' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

// 编辑器实例
const editor = useEditor({
  content: renderMarkdown(props.modelValue || ''),
  extensions: [
    StarterKit.configure({
      heading: { levels: [1, 2, 3] },
      // 不允许 horizontalRule / hardBreak 的快捷键(本项目 markdown 不渲染 <hr>)
    }),
    Placeholder.configure({ placeholder: props.placeholder }),
    Link.configure({
      openOnClick: false,
      autolink: false,
      HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
    }),
  ],
  editorProps: {
    attributes: {
      class: 'rte-prose',
      spellcheck: 'false',
    },
  },
  onUpdate({ editor }) {
    // Tiptap 输出 HTML → 转回 markdown 写回 v-model
    const html = editor.getHTML()
    const md = htmlToMarkdown(html)
    emit('update:modelValue', md)
  },
})

// 外部 modelValue 变化时同步进编辑器(避免循环写)
watch(
  () => props.modelValue,
  (newMd) => {
    if (!editor.value) return
    const currentMd = htmlToMarkdown(editor.value.getHTML())
    if (currentMd !== newMd) {
      // 第二个参数 false:不触发 onUpdate
      editor.value.commands.setContent(renderMarkdown(newMd || ''), false)
    }
  },
)

// disabled 切换
watch(
  () => props.disabled,
  (v) => editor.value?.setEditable(!v),
)

onBeforeUnmount(() => {
  editor.value?.destroy()
})

// ====== 工具栏操作 ======
const showLinkInput = ref(false)
const linkUrl = ref('')
const linkInputRef = ref(null)

function openLinkInput() {
  const e = editor.value
  if (!e) return
  if (e.isActive('link')) {
    const attrs = e.getAttributes('link')
    linkUrl.value = attrs.href || ''
  } else {
    linkUrl.value = ''
  }
  showLinkInput.value = true
  nextTick(() => linkInputRef.value?.focus())
}

function closeLinkInput() {
  showLinkInput.value = false
  linkUrl.value = ''
}

function setLink() {
  const e = editor.value
  if (!e) { closeLinkInput(); return }
  const url = linkUrl.value.trim()
  if (!url) { closeLinkInput(); return }
  const finalUrl = /^(https?:|mailto:|tel:|\/|#)/i.test(url) ? url : 'https://' + url
  e.chain().focus().extendMarkRange('link').setLink({ href: finalUrl }).run()
  closeLinkInput()
}

function unsetLink() {
  editor.value?.chain().focus().unsetLink().run()
}

const showImageInput = ref(false)
const imageUrl = ref('')
const imageAlt = ref('')
const imageInputRef = ref(null)

function openImageInput() {
  imageUrl.value = ''
  imageAlt.value = ''
  showImageInput.value = true
  nextTick(() => imageInputRef.value?.focus())
}

function closeImageInput() {
  showImageInput.value = false
  imageUrl.value = ''
  imageAlt.value = ''
}

function insertImage() {
  const e = editor.value
  if (!e) { closeImageInput(); return }
  const url = imageUrl.value.trim()
  if (!url) { closeImageInput(); return }
  e.chain().focus().setImage({ src: url, alt: imageAlt.value.trim() }).run()
  closeImageInput()
}

function isBtn(name, attrs = undefined) {
  return editor.value?.isActive(name, attrs) ? 'rte-btn rte-btn-active' : 'rte-btn'
}
</script>

<template>
  <div class="rte" :style="{ '--rte-min-h': minHeight }">
    <!-- 工具栏 -->
    <div v-if="editor" class="rte-toolbar">
      <div class="rte-group">
        <button type="button" :class="isBtn('heading', { level: 1 })" data-tip="H1 标题" @click="editor.chain().focus().toggleHeading({ level: 1 }).run()">
          <span class="rte-btn-text">H1</span>
        </button>
        <button type="button" :class="isBtn('heading', { level: 2 })" data-tip="H2 标题" @click="editor.chain().focus().toggleHeading({ level: 2 }).run()">
          <span class="rte-btn-text">H2</span>
        </button>
        <button type="button" :class="isBtn('heading', { level: 3 })" data-tip="H3 标题" @click="editor.chain().focus().toggleHeading({ level: 3 }).run()">
          <span class="rte-btn-text">H3</span>
        </button>
      </div>

      <div class="rte-divider" />

      <div class="rte-group">
        <button type="button" :class="isBtn('bold')" data-tip="加粗" @click="editor.chain().focus().toggleBold().run()">
          <Icon icon="mdi:format-bold" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('italic')" data-tip="斜体" @click="editor.chain().focus().toggleItalic().run()">
          <Icon icon="mdi:format-italic" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('strike')" data-tip="删除线" @click="editor.chain().focus().toggleStrike().run()">
          <Icon icon="mdi:format-strikethrough" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('code')" data-tip="行内代码" @click="editor.chain().focus().toggleCode().run()">
          <Icon icon="mdi:code-tags" width="14" height="14" />
        </button>
      </div>

      <div class="rte-divider" />

      <div class="rte-group">
        <button type="button" :class="isBtn('bulletList')" data-tip="无序列表" @click="editor.chain().focus().toggleBulletList().run()">
          <Icon icon="mdi:format-list-bulleted" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('orderedList')" data-tip="有序列表" @click="editor.chain().focus().toggleOrderedList().run()">
          <Icon icon="mdi:format-list-numbered" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('blockquote')" data-tip="引用" @click="editor.chain().focus().toggleBlockquote().run()">
          <Icon icon="mdi:format-quote-close" width="14" height="14" />
        </button>
        <button type="button" :class="isBtn('codeBlock')" data-tip="代码块" @click="editor.chain().focus().toggleCodeBlock().run()">
          <Icon icon="mdi:code-braces" width="14" height="14" />
        </button>
      </div>

      <div class="rte-divider" />

      <div class="rte-group">
        <div class="rte-popwrap">
          <button
            type="button"
            :class="isBtn('link')"
            data-tip="链接"
            @click="openLinkInput"
          >
            <Icon icon="mdi:link-variant" width="14" height="14" />
          </button>
          <button
            v-if="editor.isActive('link')"
            type="button"
            class="rte-btn rte-btn-danger"
            data-tip="取消链接"
            @click="unsetLink"
          >
            <Icon icon="mdi:link-variant-off" width="14" height="14" />
          </button>
          <div v-if="showLinkInput" class="rte-popover">
            <input
              ref="linkInputRef"
              v-model="linkUrl"
              type="url"
              class="rte-input"
              placeholder="https://example.com"
              @keyup.enter="setLink"
              @keyup.escape="closeLinkInput"
            />
            <div class="rte-popover-actions">
              <button type="button" class="rte-btn-sm" @click="closeLinkInput">取消</button>
              <button type="button" class="rte-btn-sm rte-btn-primary" @click="setLink">确定</button>
            </div>
          </div>
        </div>

        <div class="rte-popwrap">
          <button
            type="button"
            class="rte-btn"
            data-tip="插入图片(填 URL)"
            @click="openImageInput"
          >
            <Icon icon="mdi:image-outline" width="14" height="14" />
          </button>
          <div v-if="showImageInput" class="rte-popover">
            <input
              ref="imageInputRef"
              v-model="imageUrl"
              type="url"
              class="rte-input"
              placeholder="图片 URL"
              @keyup.enter="insertImage"
              @keyup.escape="closeImageInput"
            />
            <input
              v-model="imageAlt"
              type="text"
              class="rte-input"
              placeholder="替代文本(可选)"
              @keyup.enter="insertImage"
              @keyup.escape="closeImageInput"
            />
            <div class="rte-popover-actions">
              <button type="button" class="rte-btn-sm" @click="closeImageInput">取消</button>
              <button type="button" class="rte-btn-sm rte-btn-primary" @click="insertImage">插入</button>
            </div>
          </div>
        </div>
      </div>

      <div class="rte-divider" />

      <div class="rte-group">
        <button
          type="button"
          class="rte-btn"
          data-tip="撤销"
          :disabled="!editor.can().undo()"
          @click="editor.chain().focus().undo().run()"
        >
          <Icon icon="mdi:undo" width="14" height="14" />
        </button>
        <button
          type="button"
          class="rte-btn"
          data-tip="重做"
          :disabled="!editor.can().redo()"
          @click="editor.chain().focus().redo().run()"
        >
          <Icon icon="mdi:redo" width="14" height="14" />
        </button>
      </div>
    </div>

    <!-- 编辑区 -->
    <EditorContent :editor="editor" class="rte-content" />
  </div>
</template>

<style scoped>
.rte {
  border: 1px solid var(--border);
  border-radius: var(--radius, 6px);
  background: var(--bg-card);
  overflow: hidden;
}

.rte-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  background: var(--bg-soft, #f7f7f8);
  border-bottom: 1px solid var(--border);
  position: relative;
}

.rte-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.rte-divider {
  width: 1px;
  height: 18px;
  background: var(--border);
  margin: 0 4px;
}

.rte-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  height: 26px;
  padding: 0 6px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  cursor: pointer;
  transition: background 0.12s, border-color 0.12s, color 0.12s;
  font-size: 12px;
}
.rte-btn:hover:not(:disabled) {
  background: var(--bg-hover, rgba(0, 0, 0, 0.05));
  border-color: var(--border);
}
.rte-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.rte-btn-active {
  background: var(--primary, #3b82f6);
  color: #fff;
  border-color: var(--primary, #3b82f6);
}
.rte-btn-active:hover:not(:disabled) {
  background: var(--primary, #3b82f6);
  border-color: var(--primary, #3b82f6);
}
.rte-btn-danger {
  color: #ef4444;
}
.rte-btn-danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.3);
}
.rte-btn-text {
  font-weight: 600;
  font-size: 11px;
}

.rte-popwrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.rte-popover {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 20;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 8px;
  min-width: 260px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rte-input {
  width: 100%;
  padding: 5px 8px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg);
  color: var(--text);
  font-size: 12px;
  outline: none;
}
.rte-input:focus {
  border-color: var(--primary, #3b82f6);
}

.rte-popover-actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 2px;
}
.rte-btn-sm {
  padding: 4px 10px;
  font-size: 12px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg);
  color: var(--text);
  cursor: pointer;
}
.rte-btn-sm:hover {
  background: var(--bg-hover, rgba(0, 0, 0, 0.05));
}
.rte-btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
  border-color: var(--primary, #3b82f6);
}
.rte-btn-primary:hover {
  background: var(--primary, #3b82f6);
  border-color: var(--primary, #3b82f6);
  opacity: 0.92;
}

.rte-content {
  min-height: var(--rte-min-h, 320px);
  max-height: 70vh;
  overflow-y: auto;
  background: var(--bg-card);
}

/* Tiptap ProseMirror 基础样式 — 颜色用项目变量,排版贴近自研 markdown 渲染 */
.rte-content :deep(.ProseMirror) {
  min-height: var(--rte-min-h, 320px);
  padding: 12px 16px;
  outline: none;
  color: var(--text);
  font-size: 13px;
  line-height: 1.6;
}
.rte-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: var(--text-muted, #9ca3af);
  pointer-events: none;
  height: 0;
}
.rte-content :deep(.ProseMirror h1) {
  font-size: 1.6rem;
  font-weight: 700;
  margin: 0.6em 0 0.4em;
  color: var(--text);
}
.rte-content :deep(.ProseMirror h2) {
  font-size: 1.3rem;
  font-weight: 600;
  margin: 0.6em 0 0.4em;
  color: var(--text);
}
.rte-content :deep(.ProseMirror h3) {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0.5em 0 0.3em;
  color: var(--text);
}
.rte-content :deep(.ProseMirror p) {
  margin: 0 0 0.5em;
}
.rte-content :deep(.ProseMirror ul),
.rte-content :deep(.ProseMirror ol) {
  padding-left: 1.5em;
  margin: 0 0 0.5em;
}
.rte-content :deep(.ProseMirror li) {
  margin: 0 0 0.15em;
}
.rte-content :deep(.ProseMirror blockquote) {
  border-left: 3px solid var(--primary, #3b82f6);
  padding-left: 0.8em;
  margin: 0 0 0.5em;
  color: var(--text-muted, #6b7280);
  font-style: italic;
}
.rte-content :deep(.ProseMirror code) {
  background: var(--bg-soft, #f3f4f6);
  border-radius: 3px;
  padding: 1px 5px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 0.9em;
  color: #ef4444;
}
.rte-content :deep(.ProseMirror pre) {
  background: #1f2937;
  color: #f9fafb;
  padding: 0.8em 1em;
  border-radius: 6px;
  margin: 0 0 0.5em;
  overflow-x: auto;
}
.rte-content :deep(.ProseMirror pre code) {
  background: none;
  color: inherit;
  padding: 0;
  font-size: 0.85em;
}
.rte-content :deep(.ProseMirror a) {
  color: var(--primary, #3b82f6);
  text-decoration: underline;
}
.rte-content :deep(.ProseMirror hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 1em 0;
}
.rte-content :deep(.ProseMirror img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  display: block;
  margin: 0.5em 0;
}
.rte-content :deep(.ProseMirror s) {
  text-decoration: line-through;
}
</style>
