import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditorDraft } from '@/types'
import type { IEditorService } from '@/types/services'

type QueryResult<T> = Promise<{ data: T; error: { message: string } | null; count?: number | null }>

function createDocQueryBuilder<T>(
  maybeSingleResult: QueryResult<T>,
  singleResult: QueryResult<T>,
) {
  return {
    select: vi.fn().mockReturnThis(),
    eq: vi.fn().mockReturnThis(),
    maybeSingle: vi.fn().mockImplementation(() => maybeSingleResult),
    single: vi.fn().mockImplementation(() => singleResult),
    upsert: vi.fn().mockReturnThis(),
    delete: vi.fn().mockReturnThis(),
  }
}

function createCountBuilder<T>(result: QueryResult<T>) {
  return {
    select: vi.fn().mockReturnThis(),
    eq: vi.fn().mockImplementation(() => result),
  }
}

function createDeleteBuilder<T>(result: QueryResult<T>) {
  let eqCalls = 0
  const builder = {
    delete: vi.fn(),
    eq: vi.fn(),
  }
  builder.delete.mockReturnValue(builder)
  builder.eq.mockImplementation(() => {
    eqCalls += 1
    if (eqCalls >= 2) {
      return result
    }
    return builder
  })

  return {
    delete: builder.delete,
    eq: builder.eq,
  }
}

const mockAuthGetUser = vi.fn()
const mockFrom = vi.fn()
const mockSchema = vi.fn()

vi.mock('@/lib/supabase', () => ({
  getSupabase: () => ({
    auth: { getUser: mockAuthGetUser },
    schema: mockSchema,
  }),
}))

const FAKE_USER_ID = '550e8400-e29b-41d4-a716-446655440000'
const FAKE_NODE_ID = '550e8400-e29b-41d4-a716-446655440011'
const FAKE_DOCUMENT_ID = '550e8400-e29b-41d4-a716-446655440099'

describe('Supabase Editor adapter', () => {
  let service: IEditorService

  beforeEach(async () => {
    vi.clearAllMocks()

    mockSchema.mockReturnValue({ from: mockFrom })
    mockAuthGetUser.mockResolvedValue({
      data: { user: { id: FAKE_USER_ID } },
      error: null,
    })

    const mod = await import('@/services/supabase/editorService')
    service = mod.createSupabaseEditorService()
  })

  it('loadDraft returns EditorDraft with revision count', async () => {
    const docQuery = createDocQueryBuilder(
      Promise.resolve({
        data: {
          id: FAKE_DOCUMENT_ID,
          node_id: FAKE_NODE_ID,
          content: { t: 'draft content' },
          updated_at: '2026-02-03T00:00:00Z',
        },
        error: null,
      }),
      Promise.resolve({ data: null, error: null }),
    )

    const countBuilder = createCountBuilder(
      Promise.resolve({ data: [], error: null, count: 4 }),
    )

    mockFrom
      .mockReturnValueOnce(docQuery)
      .mockReturnValueOnce(countBuilder)

    const draft = await service.loadDraft(FAKE_NODE_ID)

    expect(draft).toEqual<EditorDraft>({
      key: FAKE_NODE_ID,
      content: { t: 'draft content' },
      updatedAt: '2026-02-03T00:00:00Z',
      revision: 4,
    })
  })

  it('saveDraft upserts documents row and returns draft shape', async () => {
    const docQuery = createDocQueryBuilder(
      Promise.resolve({ data: null, error: null }),
      Promise.resolve({
        data: {
          id: FAKE_DOCUMENT_ID,
          node_id: FAKE_NODE_ID,
          content: { t: 'saved' },
          updated_at: '2026-02-04T00:00:00Z',
        },
        error: null,
      }),
    )

    const countBuilder = createCountBuilder(
      Promise.resolve({ data: [], error: null, count: 2 }),
    )

    mockFrom
      .mockReturnValueOnce(docQuery)
      .mockReturnValueOnce(countBuilder)

    const draft = await service.saveDraft({
      key: FAKE_NODE_ID,
      content: { t: 'saved' },
      updatedAt: 'ignore-this',
      revision: 0,
    })

    expect(docQuery.upsert).toHaveBeenCalledWith(
      {
        node_id: FAKE_NODE_ID,
        user_id: FAKE_USER_ID,
        content: { t: 'saved' },
        status: 'draft',
      },
      { onConflict: 'node_id,user_id' },
    )
    expect(draft.revision).toBe(2)
    expect(draft.key).toBe(FAKE_NODE_ID)
  })

  it('deleteDraft deletes current user document by node_id', async () => {
    const docQuery = createDeleteBuilder(
      Promise.resolve({ data: null, error: null }),
    )
    mockFrom.mockReturnValueOnce(docQuery)

    await service.deleteDraft(FAKE_NODE_ID)

    expect(docQuery.delete).toHaveBeenCalled()
    expect(docQuery.eq).toHaveBeenCalledWith('node_id', FAKE_NODE_ID)
    expect(docQuery.eq).toHaveBeenCalledWith('user_id', FAKE_USER_ID)
  })
})
