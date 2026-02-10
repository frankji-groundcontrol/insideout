import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import type { RoadmapNode, Workshop } from '@/types'
import { downloadMarkdown, generateMarkdown } from '../export'

const workshop: Workshop = {
  id: 'ws_test',
  title: '导出测试工作坊',
  description: '这是一个用于测试导出能力的工作坊。',
  code: '112233',
  creator_id: 'user_001',
  status: 'active',
  created_at: '2026-02-10T12:00:00.000Z',
  member_count: 3,
}

const nodes: RoadmapNode[] = [
  {
    id: 'node_1',
    workshop_id: 'ws_test',
    title: '问题定义',
    description: '明确你要解决的问题。',
    status: 'completed',
    order: 1,
    content: '聚焦真实用户在协作中的信息同步问题。',
  },
  {
    id: 'node_2',
    workshop_id: 'ws_test',
    title: '方案草图',
    description: '画出最小可行版本。',
    status: 'in_progress',
    order: 2,
    content: '',
  },
  {
    id: 'node_3',
    workshop_id: 'ws_test',
    title: '验证假设',
    description: '整理测试与反馈。',
    status: 'pending',
    order: 3,
  },
]

describe('export utils', () => {
  let createObjectURLMock: ReturnType<typeof vi.fn>
  let revokeObjectURLMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    createObjectURLMock = vi.fn(() => 'blob:mock-url')
    revokeObjectURLMock = vi.fn()
    Object.defineProperty(globalThis.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: createObjectURLMock,
    })
    Object.defineProperty(globalThis.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: revokeObjectURLMock,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('generateMarkdown 返回以工作坊标题为 H1 的 markdown', () => {
    const markdown = generateMarkdown(workshop, nodes)

    expect(markdown.startsWith(`# ${workshop.title}`)).toBe(true)
  })

  it('generateMarkdown 包含全部节点标题作为 H2', () => {
    const markdown = generateMarkdown(workshop, nodes)

    expect(markdown).toContain('## 1. 问题定义')
    expect(markdown).toContain('## 2. 方案草图')
    expect(markdown).toContain('## 3. 验证假设')
  })

  it('generateMarkdown 对空内容节点使用 [未完成] 占位', () => {
    const markdown = generateMarkdown(workshop, nodes)
    const placeholders = markdown.match(/\[未完成\]/g) ?? []

    expect(placeholders.length).toBe(2)
  })

  it('generateMarkdown 在有内容时包含节点正文', () => {
    const markdown = generateMarkdown(workshop, nodes)

    expect(markdown).toContain('聚焦真实用户在协作中的信息同步问题。')
  })

  it('downloadMarkdown 创建 blob 并触发下载', () => {
    const anchor = document.createElement('a')
    const clickSpy = vi.spyOn(anchor, 'click').mockImplementation(() => {})
    const originalCreateElement = document.createElement.bind(document)

    vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      if (tagName === 'a') {
        return anchor
      }
      return originalCreateElement(tagName)
    })

    downloadMarkdown('# 导出内容', 'workshop.md')

    expect(createObjectURLMock).toHaveBeenCalledTimes(1)
    const blobArg = createObjectURLMock.mock.calls[0]?.[0] as Blob
    expect(blobArg).toBeInstanceOf(Blob)
    expect(blobArg.type).toBe('text/markdown')
    expect(anchor.href).toBe('blob:mock-url')
    expect(anchor.download).toBe('workshop.md')
    expect(clickSpy).toHaveBeenCalledTimes(1)
    expect(revokeObjectURLMock).toHaveBeenCalledWith('blob:mock-url')
  })
})
