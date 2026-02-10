import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { RoadmapNode, Workshop } from '@/types'
import { services } from '@/services/registry'

export const useWorkshopStore = defineStore('workshop', () => {
  const currentWorkshop = ref<Workshop | null>(null)
  const roadmapNodes = ref<RoadmapNode[]>([])
  const currentNodeId = ref('')
  const loading = ref(false)

  const currentNode = computed(() =>
    roadmapNodes.value.find((node) => node.id === currentNodeId.value),
  )

  async function loadWorkshop(id: string) {
    loading.value = true
    try {
      const [workshop, nodes] = await Promise.all([
        services.workshop.getWorkshop(id),
        services.workshop.getRoadmap(id),
      ])

      if (workshop) {
        currentWorkshop.value = workshop
      }
      roadmapNodes.value = nodes

      // 默认选中第一个进行中节点，否则选中第一个待开始节点。
      const firstActive =
        nodes.find((node) => node.status === 'in_progress') ??
        nodes.find((node) => node.status === 'pending')

      if (firstActive) {
        currentNodeId.value = firstActive.id
      }
    } finally {
      loading.value = false
    }
  }

  function selectNode(nodeId: string) {
    const node = roadmapNodes.value.find((item) => item.id === nodeId)
    if (!node || node.status === 'locked') {
      return
    }

    if (node.status === 'pending') {
      node.status = 'in_progress'
    }

    currentNodeId.value = nodeId
  }

  async function completeNode(nodeId: string) {
    const nodeIndex = roadmapNodes.value.findIndex((node) => node.id === nodeId)
    if (nodeIndex === -1) {
      return
    }

    const node = roadmapNodes.value[nodeIndex]
    if (!node || node.status !== 'in_progress') {
      return
    }

    node.status = 'completed'

    // 线性解锁：仅将下一个 locked 节点变为 pending。
    const nextNode = roadmapNodes.value[nodeIndex + 1]
    if (nextNode && nextNode.status === 'locked') {
      nextNode.status = 'pending'
    }
  }

  return {
    currentWorkshop,
    roadmapNodes,
    currentNodeId,
    currentNode,
    loading,
    loadWorkshop,
    selectNode,
    completeNode,
  }
})
