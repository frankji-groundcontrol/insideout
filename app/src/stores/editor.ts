import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useServices } from '@/composables/useServices'
import type { EditorDraft } from '@/types'

export const useEditorStore = defineStore('editor', () => {
  const content = ref<unknown>(null)
  const isDirty = ref(false)
  const insertQueue = ref<string[]>([])
  const currentDraftKey = ref('')
  const draftRevision = ref(0)

  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function setContent(jsonContent: unknown) {
    content.value = jsonContent
    isDirty.value = true
    draftRevision.value += 1

    // 800ms 防抖，且只保存当前 revision 的最新快照
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }
    const revisionAtCall = draftRevision.value
    debounceTimer = setTimeout(() => {
      debounceTimer = null
      if (revisionAtCall === draftRevision.value) {
        void saveDraft()
      }
    }, 800)
  }

  async function saveDraft() {
    if (!currentDraftKey.value || !content.value) {
      return
    }

    const draft: EditorDraft = {
      key: currentDraftKey.value,
      content: content.value,
      updatedAt: new Date().toISOString(),
      revision: draftRevision.value,
    }

    const { editor } = useServices()
    await editor.saveDraft(draft)
    isDirty.value = false
  }

  async function flush() {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    await saveDraft()
  }

  async function loadDraft(key: string) {
    currentDraftKey.value = key
    const { editor } = useServices()
    const draft = await editor.loadDraft(key)

    if (draft) {
      content.value = draft.content
      draftRevision.value = draft.revision
    } else {
      content.value = null
      draftRevision.value = 0
    }

    isDirty.value = false
  }

  function enqueueInsert(text: string) {
    insertQueue.value.push(text)
  }

  function dequeueInsert() {
    return insertQueue.value.shift()
  }

  function reset() {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    content.value = null
    isDirty.value = false
    insertQueue.value = []
    currentDraftKey.value = ''
    draftRevision.value = 0
  }

  return {
    content,
    isDirty,
    insertQueue,
    currentDraftKey,
    draftRevision,
    setContent,
    saveDraft,
    flush,
    loadDraft,
    enqueueInsert,
    dequeueInsert,
    reset,
  }
})
