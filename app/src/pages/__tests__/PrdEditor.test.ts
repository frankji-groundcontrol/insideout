import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import zhCN from '@/i18n/locales/zh-CN'
import enUS from '@/i18n/locales/en-US'
import { PRD_SECTION_KEYS, type Prd, type PrdSectionKey, type Workspace } from '@/types'
import { useUserStore } from '@/stores/user'

// F8: the PRD editor must be read-only for anyone who isn't the author or a
// workspace admin — the same rule the backend's handleUpdatePrd enforces with a
// 403. Before the fix every viewer could type into the textareas, and a rejected
// save left their phantom text stuck in the box (the one-way :value binding never
// reverted it). These tests pin the read-only gate and the revert-on-failed-save.

import PrdEditor from '@/pages/prd/[id]/index.vue'

const ME = 'user_me'

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return { ...actual, useRoute: () => ({ params: { id: 'prd_1' } }) }
})

const prdMock = {
  get: vi.fn(),
  updateSections: vi.fn(),
  updateStatus: vi.fn(),
  createRevision: vi.fn(),
  listRevisions: vi.fn(),
  build: vi.fn(),
}
const workspaceMock = { get: vi.fn() }
vi.mock('@/composables/useServices', () => ({
  useServices: () => ({ prd: prdMock, workspace: workspaceMock }),
}))

const createTestI18n = () =>
  createI18n({ legacy: false, locale: 'zh-CN', fallbackLocale: 'zh-CN', messages: { 'zh-CN': zhCN, 'en-US': enUS } })

function makePrd(authorId: string): Prd {
  const sections = {} as Record<PrdSectionKey, string>
  for (const key of PRD_SECTION_KEYS) sections[key] = `${key} body`
  return {
    id: 'prd_1', workspaceId: 'ws_1', authorId, title: 'My PRD',
    sections, status: 'draft', currentRevision: 1, updatedAt: '2026-07-27T00:00:00Z',
  }
}

function makeWorkspace(myRole: 'admin' | 'member'): Workspace {
  return {
    id: 'ws_1', title: 'WS', description: '', code: 'ABC123',
    status: 'active', memberCount: 1, myRole, createdAt: '2026-07-27T00:00:00Z',
  }
}

async function mountEditor(prd: Prd, ws: Workspace) {
  prdMock.get.mockResolvedValue(prd)
  workspaceMock.get.mockResolvedValue(ws)
  const wrapper = mount(PrdEditor, {
    global: {
      plugins: [createTestI18n()],
      stubs: { CoachPanel: true, NuxtLink: true },
    },
  })
  await flushPromises() // let load() resolve prd + workspace
  return wrapper
}

describe('PrdEditor read-only gate (F8)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    const userStore = useUserStore()
    userStore.user = { id: ME, email: 'me@example.com', username: 'me', bio: '', keywords: [] }
  })

  it('locks the title and every section for a non-author member', async () => {
    const wrapper = await mountEditor(makePrd('someone_else'), makeWorkspace('member'))
    expect(wrapper.get('input').attributes('readonly')).toBeDefined()
    const textareas = wrapper.findAll('textarea')
    expect(textareas.length).toBe(PRD_SECTION_KEYS.length)
    for (const ta of textareas) expect(ta.attributes('readonly')).toBeDefined()
  })

  it('lets the author edit', async () => {
    const wrapper = await mountEditor(makePrd(ME), makeWorkspace('member'))
    expect(wrapper.get('input').attributes('readonly')).toBeUndefined()
    for (const ta of wrapper.findAll('textarea')) expect(ta.attributes('readonly')).toBeUndefined()
  })

  it('lets a workspace admin edit even when not the author', async () => {
    const wrapper = await mountEditor(makePrd('someone_else'), makeWorkspace('admin'))
    expect(wrapper.get('input').attributes('readonly')).toBeUndefined()
    for (const ta of wrapper.findAll('textarea')) expect(ta.attributes('readonly')).toBeUndefined()
  })

  it('reverts a section to the server value when the save is rejected', async () => {
    prdMock.updateSections.mockRejectedValue(new Error('409'))
    const wrapper = await mountEditor(makePrd(ME), makeWorkspace('member'))

    const first = wrapper.findAll('textarea')[0]!
    await first.setValue('hijacked text')
    await first.trigger('blur')
    await flushPromises()

    expect(prdMock.updateSections).toHaveBeenCalledTimes(1)
    // The rejected save must not leave the phantom edit in the box.
    expect((first.element as HTMLTextAreaElement).value).toBe(`${PRD_SECTION_KEYS[0]} body`)
  })
})
