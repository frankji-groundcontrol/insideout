import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import RoadmapCanvasNode from '@/components/roadmap/RoadmapCanvasNode.vue'
import type { RoadmapTreeNode } from '@/types'
import zhCN from '@/i18n/locales/zh-CN'

// These tests pin the D9 sparse-payload contract: status-cycle dispatches only
// {status}, and the edit form dispatches only {title, description}. If either
// ever sends the full node, the COALESCE partial PATCH no longer protects the
// other fields from being clobbered.

vi.mock('@/composables/useServices', () => ({
  useServices: () => ({ roadmap: roadmapMock }),
}))

const roadmapMock = {
  update: vi.fn(async () => ({})),
  create: vi.fn(async () => ({})),
  remove: vi.fn(async () => undefined),
  expand: vi.fn(async () => []),
}

const i18n = createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': zhCN } })

function makeNode(over: Partial<RoadmapTreeNode> = {}): RoadmapTreeNode {
  return {
    id: 'n1',
    projectId: 'p1',
    parentId: null,
    title: 'Ship MVP',
    description: 'the first cut',
    status: 'pending',
    position: 0,
    children: [],
    ...over,
  } as RoadmapTreeNode
}

function mountNode(node: RoadmapTreeNode) {
  return mount(RoadmapCanvasNode, {
    props: { node, projectId: 'p1' },
    global: { plugins: [i18n] },
  })
}

describe('RoadmapCanvasNode sparse updates (D9)', () => {
  it('status cycle dispatches only { status }', async () => {
    roadmapMock.update.mockClear()
    const wrapper = mountNode(makeNode({ status: 'pending' }))
    await wrapper.get('button[title]').trigger('click')
    expect(roadmapMock.update).toHaveBeenCalledTimes(1)
    expect(roadmapMock.update).toHaveBeenCalledWith('n1', { status: 'in_progress' })
  })

  it('edit form dispatches only { title, description }', async () => {
    roadmapMock.update.mockClear()
    const wrapper = mountNode(makeNode())
    await wrapper.get('button[title="' + zhCN.roadmap.edit + '"]').trigger('click')
    await wrapper.get('form').trigger('submit')
    expect(roadmapMock.update).toHaveBeenCalledTimes(1)
    expect(roadmapMock.update).toHaveBeenCalledWith('n1', {
      title: 'Ship MVP',
      description: 'the first cut',
    })
  })
})
