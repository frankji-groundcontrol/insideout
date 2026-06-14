import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { SupabaseClient } from '@supabase/supabase-js'
import type { RoadmapNode, UserSubmission, Workshop } from '@/types'
import type { IWorkshopService } from '@/types/services'
import { createSupabaseWorkshopService } from '@/services/supabase/workshopService'

type QueryResult<T> = Promise<{ data: T; error: { message: string } | null }>

function createSelectBuilder<T>(selectResult: QueryResult<T>) {
  return {
    select: vi.fn().mockImplementation(() => selectResult),
  }
}

function createFilterBuilder<T>(
  singleResult: QueryResult<T>,
  maybeSingleResult: QueryResult<T>,
) {
  return {
    select: vi.fn().mockReturnThis(),
    eq: vi.fn().mockReturnThis(),
    single: vi.fn().mockImplementation(() => singleResult),
    maybeSingle: vi.fn().mockImplementation(() => maybeSingleResult),
    insert: vi.fn().mockReturnThis(),
    update: vi.fn().mockReturnThis(),
  }
}

const mockAuthGetUser = vi.fn()
const mockRpc = vi.fn()
const mockFrom = vi.fn()
const mockSchema = vi.fn()

// 注入式 stub 客户端：适配器工厂现在接收 SupabaseClient 参数。
function createStubClient(): SupabaseClient {
  return {
    auth: { getUser: mockAuthGetUser },
    schema: mockSchema,
  } as unknown as SupabaseClient
}

const FAKE_USER_ID = '550e8400-e29b-41d4-a716-446655440000'
const FAKE_WORKSHOP_ID = '550e8400-e29b-41d4-a716-446655440001'
const FAKE_NODE_ID = '550e8400-e29b-41d4-a716-446655440011'

describe('Supabase Workshop adapter', () => {
  let service: IWorkshopService

  beforeEach(() => {
    vi.clearAllMocks()

    mockSchema.mockReturnValue({
      from: mockFrom,
      rpc: mockRpc,
    })

    mockAuthGetUser.mockResolvedValue({
      data: { user: { id: FAKE_USER_ID } },
      error: null,
    })

    service = createSupabaseWorkshopService(createStubClient())
  })

  it('getWorkshops returns Workshop[] with summary fields', async () => {
    const dbRows = [
      {
        id: FAKE_WORKSHOP_ID,
        title: 'Workshop A',
        description: 'Desc',
        cover_url: 'https://example.com/a.png',
        code: '123456',
        creator_id: FAKE_USER_ID,
        status: 'active',
        created_at: '2026-02-01T00:00:00Z',
        member_count: 3,
        is_joined: true,
      },
    ]

    const query = createSelectBuilder(Promise.resolve({ data: dbRows, error: null }))
    mockFrom.mockReturnValue(query)

    const workshops = await service.getWorkshops()

    expect(mockSchema).toHaveBeenCalledWith('juanleme')
    expect(mockFrom).toHaveBeenCalledWith('v_workshop_summary')
    expect(workshops[0]).toEqual<Workshop>({
      id: FAKE_WORKSHOP_ID,
      title: 'Workshop A',
      description: 'Desc',
      cover_url: 'https://example.com/a.png',
      code: '123456',
      creator_id: FAKE_USER_ID,
      status: 'active',
      created_at: '2026-02-01T00:00:00Z',
      member_count: 3,
      is_joined: true,
    })
  })

  it('getRoadmap maps workspace_id to workshop_id and computes locked', async () => {
    mockRpc.mockResolvedValue({
      data: [
        {
          id: 'n1',
          workspace_id: FAKE_WORKSHOP_ID,
          title: 'Node 1',
          description: 'D1',
          order: 1,
          status: 'completed',
          content: 'done',
        },
        {
          id: 'n2',
          workspace_id: FAKE_WORKSHOP_ID,
          title: 'Node 2',
          description: 'D2',
          order: 2,
          status: 'pending',
          content: undefined,
        },
        {
          id: 'n3',
          workspace_id: FAKE_WORKSHOP_ID,
          title: 'Node 3',
          description: 'D3',
          order: 3,
          status: 'pending',
          content: undefined,
        },
      ],
      error: null,
    })

    const roadmap = await service.getRoadmap(FAKE_WORKSHOP_ID)

    expect(mockRpc).toHaveBeenCalledWith('get_workshop_roadmap', {
      p_workspace_id: FAKE_WORKSHOP_ID,
    })
    expect(roadmap).toEqual<RoadmapNode[]>([
      {
        id: 'n1',
        workshop_id: FAKE_WORKSHOP_ID,
        title: 'Node 1',
        description: 'D1',
        order: 1,
        status: 'completed',
        content: 'done',
      },
      {
        id: 'n2',
        workshop_id: FAKE_WORKSHOP_ID,
        title: 'Node 2',
        description: 'D2',
        order: 2,
        status: 'pending',
        content: undefined,
      },
      {
        id: 'n3',
        workshop_id: FAKE_WORKSHOP_ID,
        title: 'Node 3',
        description: 'D3',
        order: 3,
        status: 'locked',
        content: undefined,
      },
    ])
  })

  it('joinWorkshop returns true on success and throws on invalid code', async () => {
    mockRpc.mockResolvedValueOnce({
      data: 'membership-id',
      error: null,
    })

    await expect(service.joinWorkshop('123456')).resolves.toBe(true)
    expect(mockRpc).toHaveBeenCalledWith('join_workshop', { p_code: '123456' })

    mockRpc.mockResolvedValueOnce({
      data: null,
      error: { message: 'Invalid invite code' },
    })

    await expect(service.joinWorkshop('000000')).rejects.toThrow('Invalid invite code')
  })

  it('createWorkshop calls create_workspace RPC', async () => {
    mockRpc.mockResolvedValueOnce({
      data: FAKE_WORKSHOP_ID,
      error: null,
    })

    const summaryRow = {
      id: FAKE_WORKSHOP_ID,
      title: 'New WS',
      description: 'New Desc',
      cover_url: null,
      code: '654321',
      creator_id: FAKE_USER_ID,
      status: 'draft',
      created_at: '2026-02-02T00:00:00Z',
      member_count: 1,
      is_joined: true,
    }
    const query = createFilterBuilder(
      Promise.resolve({ data: summaryRow, error: null }),
      Promise.resolve({ data: summaryRow, error: null }),
    )
    mockFrom.mockReturnValue(query)

    const created = await service.createWorkshop({
      title: 'New WS',
      description: 'New Desc',
    })

    expect(mockRpc).toHaveBeenCalledWith('create_workspace', {
      p_title: 'New WS',
      p_description: 'New Desc',
      p_cover_url: null,
      p_status: 'draft',
    })
    expect(created.id).toBe(FAKE_WORKSHOP_ID)
  })

  it('getSubmission maps row to UserSubmission and saveSubmission calls complete_node', async () => {
    const submissionRow = {
      id: 'doc-1',
      node_id: FAKE_NODE_ID,
      user_id: FAKE_USER_ID,
      content: { text: 'hello' },
      updated_at: '2026-02-03T00:00:00Z',
      status: 'draft',
    }
    const query = createFilterBuilder(
      Promise.resolve({ data: submissionRow, error: null }),
      Promise.resolve({ data: submissionRow, error: null }),
    )
    mockFrom.mockReturnValue(query)

    const loaded = await service.getSubmission(FAKE_NODE_ID)
    expect(loaded).toEqual<UserSubmission>({
      nodeId: FAKE_NODE_ID,
      userId: FAKE_USER_ID,
      content: { text: 'hello' },
      submittedAt: '2026-02-03T00:00:00Z',
    })

    mockRpc.mockResolvedValueOnce({
      data: { document_id: 'doc-1', revision: 2, status: 'completed' },
      error: null,
    })

    const saved = await service.saveSubmission({
      nodeId: FAKE_NODE_ID,
      userId: FAKE_USER_ID,
      content: { text: 'done' },
      submittedAt: '2026-02-04T00:00:00Z',
    })
    expect(mockRpc).toHaveBeenCalledWith('complete_node', {
      p_node_id: FAKE_NODE_ID,
      p_content: { text: 'done' },
    })
    expect(saved.nodeId).toBe(FAKE_NODE_ID)
  })
})
