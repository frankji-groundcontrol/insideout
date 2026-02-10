<script setup lang="ts">
/* biome-ignore-all lint/correctness/noUnusedVariables: 模板中会使用变量与方法 */
import type { Editor } from '@tiptap/vue-3'
import { useI18n } from 'vue-i18n'

interface Props {
  editor: Editor | null | undefined
}

const props = defineProps<Props>()
const { t } = useI18n()

function run(action: (editor: Editor) => void) {
  if (!props.editor) {
    return
  }
  action(props.editor)
}

function canUndo() {
  return props.editor?.can().chain().focus().undo().run() ?? false
}

function canRedo() {
  return props.editor?.can().chain().focus().redo().run() ?? false
}
</script>

<template>
  <div class="editor-toolbar rounded-t-xl border border-b-0 border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900">
    <div class="flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('bold') }"
        :title="t('editor.toolbar.bold')"
        @click="run((e) => e.chain().focus().toggleBold().run())"
      >
        B
      </button>
      <button
        type="button"
        class="toolbar-btn italic-font"
        :class="{ 'is-active': editor?.isActive('italic') }"
        :title="t('editor.toolbar.italic')"
        @click="run((e) => e.chain().focus().toggleItalic().run())"
      >
        I
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('heading', { level: 1 }) }"
        :title="t('editor.toolbar.h1')"
        @click="run((e) => e.chain().focus().toggleHeading({ level: 1 }).run())"
      >
        H1
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('heading', { level: 2 }) }"
        :title="t('editor.toolbar.h2')"
        @click="run((e) => e.chain().focus().toggleHeading({ level: 2 }).run())"
      >
        H2
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('heading', { level: 3 }) }"
        :title="t('editor.toolbar.h3')"
        @click="run((e) => e.chain().focus().toggleHeading({ level: 3 }).run())"
      >
        H3
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('bulletList') }"
        :title="t('editor.toolbar.bulletList')"
        @click="run((e) => e.chain().focus().toggleBulletList().run())"
      >
        Bul
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('orderedList') }"
        :title="t('editor.toolbar.orderedList')"
        @click="run((e) => e.chain().focus().toggleOrderedList().run())"
      >
        1. List
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('codeBlock') }"
        :title="t('editor.toolbar.codeBlock')"
        @click="run((e) => e.chain().focus().toggleCodeBlock().run())"
      >
        &lt;/&gt;
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :class="{ 'is-active': editor?.isActive('blockquote') }"
        :title="t('editor.toolbar.blockquote')"
        @click="run((e) => e.chain().focus().toggleBlockquote().run())"
      >
        Quote
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :title="t('editor.toolbar.horizontalRule')"
        @click="run((e) => e.chain().focus().setHorizontalRule().run())"
      >
        —
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :disabled="!canUndo()"
        :title="t('editor.toolbar.undo')"
        @click="run((e) => e.chain().focus().undo().run())"
      >
        Undo
      </button>
      <button
        type="button"
        class="toolbar-btn"
        :disabled="!canRedo()"
        :title="t('editor.toolbar.redo')"
        @click="run((e) => e.chain().focus().redo().run())"
      >
        Redo
      </button>
    </div>
  </div>
</template>

<style scoped>
.editor-toolbar {
  backdrop-filter: blur(6px);
}

.toolbar-btn {
  border: 1px solid rgb(209 213 219);
  border-radius: 0.55rem;
  color: rgb(31 41 55);
  background: rgb(249 250 251);
  font-size: 0.8125rem;
  line-height: 1;
  min-width: 2.25rem;
  height: 2.25rem;
  padding: 0 0.55rem;
  transition: all 160ms ease;
}

.toolbar-btn:hover:not(:disabled) {
  background: rgb(238 242 255);
  border-color: rgb(129 140 248);
  color: rgb(55 48 163);
}

.toolbar-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.toolbar-btn.is-active {
  background: linear-gradient(135deg, rgb(79 70 229), rgb(56 189 248));
  border-color: rgb(79 70 229);
  color: rgb(255 255 255);
  box-shadow: 0 8px 20px rgb(79 70 229 / 0.24);
}

.italic-font {
  font-style: italic;
}

:global(.dark) .toolbar-btn {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
  color: rgb(243 244 246);
}

:global(.dark) .toolbar-btn:hover:not(:disabled) {
  background: rgb(30 41 59);
  border-color: rgb(96 165 250);
  color: rgb(191 219 254);
}

:global(.dark) .toolbar-btn.is-active {
  background: linear-gradient(135deg, rgb(129 140 248), rgb(20 184 166));
  border-color: rgb(129 140 248);
  color: rgb(255 255 255);
}
</style>
