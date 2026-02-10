import { computed, type ComputedRef, type Ref } from 'vue'

interface UseWorkshopDraftKeyOptions {
  userId: ComputedRef<string>
  workshopId: ComputedRef<string>
  activeNodeId: ComputedRef<string>
}

export function useWorkshopDraftKey(options: UseWorkshopDraftKeyOptions): {
  activeDraftKey: Ref<string>
  createDraftKey: (nodeId: string) => string
} {
  function createDraftKey(nodeId: string): string {
    return `draft:${options.userId.value}:${options.workshopId.value}:${nodeId}:v1`
  }

  const activeDraftKey = computed(() => {
    if (!options.activeNodeId.value) {
      return ''
    }
    return createDraftKey(options.activeNodeId.value)
  })

  return {
    activeDraftKey,
    createDraftKey,
  }
}
