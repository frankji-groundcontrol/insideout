import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { Workshop, RoadmapNode } from '@/types'
import { createMockWorkshopService } from '@/services/mock/workshopService'
import type { ServiceRegistry } from '@/services/registry'

// 用真实的 mock workshop 服务构造一个 registry-shaped 对象，store 通过 useServices() 访问它。
// 单元测试直接 mock 该 composable，避免依赖 Nuxt 应用上下文（useNuxtApp）。
const services = { workshop: createMockWorkshopService() } as unknown as ServiceRegistry

vi.mock('@/composables/useServices', () => ({
  useServices: () => services,
}))

import { useWorkshopStore } from '@/stores/workshop'

const mockWorkshop: Workshop = {
  id: 'ws_001',
  title: '测试工作坊',
  description: '测试描述',
  code: '123456',
  creator_id: 'user_001',
  status: 'active',
  created_at: '2026-01-01T00:00:00.000Z',
  member_count: 12,
}

describe('useWorkshopStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('loadWorkshop 会填充 currentWorkshop 和 roadmapNodes', async () => {
    const nodes: RoadmapNode[] = [
      {
        id: 'node_1',
        workshop_id: 'ws_001',
        title: '节点 1',
        description: 'desc',
        status: 'completed',
        order: 1,
      },
      {
        id: 'node_2',
        workshop_id: 'ws_001',
        title: '节点 2',
        description: 'desc',
        status: 'in_progress',
        order: 2,
      },
      {
        id: 'node_3',
        workshop_id: 'ws_001',
        title: '节点 3',
        description: 'desc',
        status: 'pending',
        order: 3,
      },
    ]
    vi.spyOn(services.workshop, 'getWorkshop').mockResolvedValue(mockWorkshop)
    vi.spyOn(services.workshop, 'getRoadmap').mockResolvedValue(nodes)

    const store = useWorkshopStore()
    await store.loadWorkshop('ws_001')

    expect(store.currentWorkshop).toEqual(mockWorkshop)
    expect(store.roadmapNodes).toEqual(nodes)
    expect(store.currentNodeId).toBe('node_2')
  })

  it('loadWorkshop 保留 mock 节点状态（首节点可为 pending）', async () => {
    const nodes: RoadmapNode[] = [
      {
        id: 'node_1',
        workshop_id: 'ws_001',
        title: '节点 1',
        description: 'desc',
        status: 'pending',
        order: 1,
      },
      {
        id: 'node_2',
        workshop_id: 'ws_001',
        title: '节点 2',
        description: 'desc',
        status: 'locked',
        order: 2,
      },
    ]
    vi.spyOn(services.workshop, 'getWorkshop').mockResolvedValue(mockWorkshop)
    vi.spyOn(services.workshop, 'getRoadmap').mockResolvedValue(nodes)

    const store = useWorkshopStore()
    await store.loadWorkshop('ws_001')

    expect(store.roadmapNodes[0]?.status).toBe('pending')
    expect(store.roadmapNodes[1]?.status).toBe('locked')
  })

  it('selectNode 选中 pending 节点后状态变为 in_progress', () => {
    const store = useWorkshopStore()
    store.roadmapNodes = [
      {
        id: 'node_1',
        workshop_id: 'ws_001',
        title: '节点 1',
        description: 'desc',
        status: 'pending',
        order: 1,
      },
    ]

    store.selectNode('node_1')

    expect(store.currentNodeId).toBe('node_1')
    expect(store.roadmapNodes[0]?.status).toBe('in_progress')
  })

  it('completeNode 会完成当前节点并将下一个 locked 节点变为 pending', async () => {
    const store = useWorkshopStore()
    store.roadmapNodes = [
      {
        id: 'node_1',
        workshop_id: 'ws_001',
        title: '节点 1',
        description: 'desc',
        status: 'in_progress',
        order: 1,
      },
      {
        id: 'node_2',
        workshop_id: 'ws_001',
        title: '节点 2',
        description: 'desc',
        status: 'locked',
        order: 2,
      },
    ]
    store.currentNodeId = 'node_1'

    await store.completeNode('node_1')

    expect(store.roadmapNodes[0]?.status).toBe('completed')
    expect(store.roadmapNodes[1]?.status).toBe('pending')
  })

  it('无法选中 locked 节点，状态和当前节点都不变', () => {
    const store = useWorkshopStore()
    store.roadmapNodes = [
      {
        id: 'node_1',
        workshop_id: 'ws_001',
        title: '节点 1',
        description: 'desc',
        status: 'in_progress',
        order: 1,
      },
      {
        id: 'node_2',
        workshop_id: 'ws_001',
        title: '节点 2',
        description: 'desc',
        status: 'locked',
        order: 2,
      },
    ]
    store.currentNodeId = 'node_1'

    store.selectNode('node_2')

    expect(store.currentNodeId).toBe('node_1')
    expect(store.roadmapNodes[1]?.status).toBe('locked')
  })
})
