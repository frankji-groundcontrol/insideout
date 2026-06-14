<script setup lang="ts">
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { useI18n } from 'vue-i18n'
import { useEditorStore } from '@/stores/editor'
import EditorToolbar from '@/components/workshop/EditorToolbar.vue'

interface Props {
  draftKey: string
}

const props = defineProps<Props>()
const editorStore = useEditorStore()
const { t } = useI18n()

const editor = useEditor({
  // Nuxt SSR：延迟到客户端再构建编辑器，避免服务端渲染时访问 document 报错与水合不一致
  immediatelyRender: false,
  extensions: [
    StarterKit,
    Placeholder.configure({
      placeholder: t('editor.placeholder'),
    }),
  ],
  content: '',
  onUpdate: ({ editor: currentEditor }) => {
    editorStore.setContent(currentEditor.getJSON())
  },
})

onMounted(async () => {
  await editorStore.loadDraft(props.draftKey)

  // 初次加载时只从 store 同步一次到编辑器，后续保持单向同步
  if (editor.value && editorStore.content) {
    editor.value.commands.setContent(editorStore.content)
  }
})

watch(
  () => editorStore.insertQueue.length,
  (queueLength) => {
    if (!editor.value || queueLength === 0) {
      return
    }

    let nextInsert = editorStore.dequeueInsert()
    while (nextInsert !== undefined) {
      editor.value.chain().focus().insertContent(nextInsert).run()
      nextInsert = editorStore.dequeueInsert()
    }
  },
)

onBeforeUnmount(async () => {
  await editorStore.flush()
})
</script>

<template>
  <section class="task-editor-shell w-full">
    <EditorToolbar :editor="editor" />
    <div class="editor-surface rounded-b-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900">
      <EditorContent :editor="editor" class="task-editor-content" />
    </div>
  </section>
</template>

<style scoped>
.task-editor-shell {
  width: 100%;
}

.editor-surface {
  min-height: 22rem;
  overflow: hidden;
}

.task-editor-content {
  width: 100%;
}

.task-editor-content :deep(.ProseMirror) {
  min-height: 22rem;
  padding: 1.2rem 1.1rem;
  color: rgb(17 24 39);
  line-height: 1.7;
  font-size: 1rem;
  background:
    radial-gradient(circle at 100% 0%, rgb(239 246 255) 0%, transparent 45%),
    radial-gradient(circle at 0% 100%, rgb(240 253 250) 0%, transparent 40%),
    rgb(255 255 255);
}

.task-editor-content :deep(.ProseMirror:focus) {
  outline: none;
}

.task-editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: rgb(156 163 175);
  pointer-events: none;
  height: 0;
}

.task-editor-content :deep(.ProseMirror h1) {
  font-size: 1.8rem;
  margin: 1.1rem 0 0.7rem;
}

.task-editor-content :deep(.ProseMirror h2) {
  font-size: 1.45rem;
  margin: 1rem 0 0.6rem;
}

.task-editor-content :deep(.ProseMirror h3) {
  font-size: 1.25rem;
  margin: 0.9rem 0 0.5rem;
}

.task-editor-content :deep(.ProseMirror code) {
  background-color: rgb(226 232 240);
  border-radius: 0.35rem;
  padding: 0.15rem 0.35rem;
}

.task-editor-content :deep(.ProseMirror pre) {
  background-color: rgb(15 23 42);
  color: rgb(241 245 249);
  border-radius: 0.75rem;
  padding: 0.9rem;
  overflow-x: auto;
}

.task-editor-content :deep(.ProseMirror hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid rgb(209 213 219);
}

:global(.dark) .task-editor-content :deep(.ProseMirror) {
  color: rgb(243 244 246);
  background:
    radial-gradient(circle at 100% 0%, rgb(30 41 59) 0%, transparent 45%),
    radial-gradient(circle at 0% 100%, rgb(12 74 110) 0%, transparent 42%),
    rgb(3 7 18);
}

:global(.dark) .task-editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  color: rgb(107 114 128);
}

:global(.dark) .task-editor-content :deep(.ProseMirror code) {
  background-color: rgb(30 41 59);
}

:global(.dark) .task-editor-content :deep(.ProseMirror hr) {
  border-top-color: rgb(75 85 99);
}

@media (max-width: 768px) {
  .task-editor-content :deep(.ProseMirror) {
    min-height: 18rem;
    padding: 0.95rem 0.85rem;
  }
}
</style>
