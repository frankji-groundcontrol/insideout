import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { EditorDraft } from '@/types'

const { loadDraftMock, saveDraftMock } = vi.hoisted(() => ({
  loadDraftMock: vi.fn<(key: string) => Promise<EditorDraft | undefined>>(),
  saveDraftMock: vi.fn<(draft: EditorDraft) => Promise<EditorDraft>>(),
}))

vi.mock('@/composables/useServices', () => ({
  useServices: () => ({
    editor: {
      loadDraft: loadDraftMock,
      saveDraft: saveDraftMock,
    },
  }),
}))

import { useEditorStore } from '@/stores/editor'

describe('useEditorStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useRealTimers()
    vi.clearAllMocks()

    loadDraftMock.mockResolvedValue(undefined)
    saveDraftMock.mockImplementation(async (draft) => draft)
  })

  it('setContent 更新 content 状态', async () => {
    const store = useEditorStore()
    await store.loadDraft('draft:test')

    const json = { type: 'doc', content: [{ type: 'paragraph' }] }
    store.setContent(json)

    expect(store.content).toEqual(json)
    expect(store.isDirty).toBe(true)
  })

  it('setContent 在 800ms 后触发 saveDraft', async () => {
    vi.useFakeTimers()
    const store = useEditorStore()
    await store.loadDraft('draft:test')

    store.setContent({ type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'hello' }] }] })

    expect(saveDraftMock).not.toHaveBeenCalled()

    vi.advanceTimersByTime(799)
    expect(saveDraftMock).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    await vi.runAllTimersAsync()

    expect(saveDraftMock).toHaveBeenCalledTimes(1)
    expect(store.isDirty).toBe(false)
  })

  it('flush 立即保存，不等待防抖时间', async () => {
    vi.useFakeTimers()
    const store = useEditorStore()
    await store.loadDraft('draft:test')

    store.setContent({ type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'flush now' }] }] })
    await store.flush()

    expect(saveDraftMock).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    await vi.runAllTimersAsync()
    expect(saveDraftMock).toHaveBeenCalledTimes(1)
    expect(store.isDirty).toBe(false)
  })

  it('loadDraft 根据 key 恢复草稿内容', async () => {
    const draft: EditorDraft = {
      key: 'draft:demo',
      content: { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'restored' }] }] },
      updatedAt: new Date().toISOString(),
      revision: 3,
    }
    loadDraftMock.mockResolvedValueOnce(draft)

    const store = useEditorStore()
    await store.loadDraft('draft:demo')

    expect(loadDraftMock).toHaveBeenCalledWith('draft:demo')
    expect(store.content).toEqual(draft.content)
    expect(store.draftRevision).toBe(3)
    expect(store.isDirty).toBe(false)
  })

  it('enqueueInsert 后 dequeueInsert 返回同一文本', () => {
    const store = useEditorStore()

    store.enqueueInsert('hello')

    expect(store.dequeueInsert()).toBe('hello')
    expect(store.dequeueInsert()).toBeUndefined()
  })

  it('isDirty 跟踪未保存变更状态', async () => {
    vi.useFakeTimers()
    const store = useEditorStore()
    await store.loadDraft('draft:test')

    expect(store.isDirty).toBe(false)

    store.setContent({ type: 'doc', content: [] })
    expect(store.isDirty).toBe(true)

    vi.advanceTimersByTime(800)
    await vi.runAllTimersAsync()

    expect(store.isDirty).toBe(false)
  })
})
