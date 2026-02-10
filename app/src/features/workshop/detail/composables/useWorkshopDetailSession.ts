import { computed, onBeforeUnmount, ref, watch, type ComputedRef } from 'vue'
import type { Router } from 'vue-router'
import { useEditorStore } from '@/stores/editor'
import { useWorkshopStore } from '@/stores/workshop'

interface UseWorkshopDetailSessionOptions {
  workshopId: ComputedRef<string>
  activeNodeId: ComputedRef<string>
  createDraftKey: (nodeId: string) => string
  router: Router
}

function toPlainText(content: unknown): string {
  if (!content || typeof content !== 'object') {
    return ''
  }

  const lines: string[] = []

  const visit = (node: unknown) => {
    if (!node || typeof node !== 'object') {
      return
    }

    const entry = node as { text?: string; content?: unknown[] }
    if (typeof entry.text === 'string') {
      lines.push(entry.text)
    }

    if (Array.isArray(entry.content)) {
      for (const child of entry.content) {
        visit(child)
      }
      lines.push('\n')
    }
  }

  visit(content)

  return lines
    .join('')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

export function useWorkshopDetailSession(options: UseWorkshopDetailSessionOptions) {
  const workshopStore = useWorkshopStore()
  const editorStore = useEditorStore()
  const isSwitchingNode = ref(false)

  const nodes = computed(() => workshopStore.roadmapNodes)
  const activeNode = computed(() => workshopStore.currentNode)
  const loading = computed(() => workshopStore.loading)

  async function loadWorkshopAndDraft(nextWorkshopId: string) {
    if (!nextWorkshopId) {
      return
    }

    await workshopStore.loadWorkshop(nextWorkshopId)
    if (workshopStore.currentNodeId) {
      await editorStore.loadDraft(options.createDraftKey(workshopStore.currentNodeId))
    }
  }

  async function switchNodeWithDraft(nodeId: string) {
    if (!nodeId || nodeId === options.activeNodeId.value || isSwitchingNode.value) {
      return
    }

    isSwitchingNode.value = true
    try {
      // 切换节点前先落盘，避免当前编辑内容丢失。
      await editorStore.flush()
      workshopStore.selectNode(nodeId)
    } finally {
      isSwitchingNode.value = false
    }
  }

  async function handleCompleteNode() {
    const nodeId = options.activeNodeId.value
    if (!nodeId) {
      return
    }

    await editorStore.flush()

    const currentContent = toPlainText(editorStore.content)
    if (currentContent) {
      workshopStore.setNodeContent(nodeId, currentContent)
    }

    await workshopStore.completeNode(nodeId)

    // 若 store 尚未切到下一节点，则兜底激活下一个 pending 节点。
    if (workshopStore.currentNodeId === nodeId) {
      const nextPending = workshopStore.roadmapNodes.find((node) => node.status === 'pending')
      if (nextPending) {
        workshopStore.selectNode(nextPending.id)
      }
    }
  }

  function goToExportPage() {
    if (!options.workshopId.value) {
      return
    }
    void options.router.push(`/workshop/${options.workshopId.value}/export`)
  }

  watch(
    () => workshopStore.currentNodeId,
    async (nodeId, previousNodeId) => {
      if (!nodeId || nodeId === previousNodeId) {
        return
      }
      await editorStore.loadDraft(options.createDraftKey(nodeId))
    },
  )

  watch(
    () => options.workshopId.value,
    async (nextWorkshopId) => {
      await loadWorkshopAndDraft(nextWorkshopId)
    },
    { immediate: true },
  )

  onBeforeUnmount(async () => {
    await editorStore.flush()
  })

  return {
    nodes,
    activeNode,
    loading,
    switchNodeWithDraft,
    handleCompleteNode,
    goToExportPage,
  }
}
