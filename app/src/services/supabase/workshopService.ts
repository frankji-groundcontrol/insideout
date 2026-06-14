import type { SupabaseClient } from '@supabase/supabase-js'
import type { RoadmapNode, UserSubmission, Workshop } from '@/types'
import type { IWorkshopService } from '@/types/services'

type DbWorkshopSummaryRow = {
  id: string
  title: string
  description: string
  cover_url: string | null
  code: string
  creator_id: string
  status: Workshop['status']
  created_at: string
  member_count: number
  is_joined: boolean | null
}

type DbRoadmapRow = {
  id: string
  workspace_id: string
  title: string
  description: string
  order: number
  status: 'pending' | 'in_progress' | 'completed'
  content: unknown
}

type DbSubmissionRow = {
  node_id: string
  user_id: string
  content: unknown
  updated_at: string
}

type DbNodeRow = {
  id: string
  workspace_id: string
  title: string
  description: string
  order: number
}

function toWorkshop(row: DbWorkshopSummaryRow): Workshop {
  return {
    id: row.id,
    title: row.title,
    description: row.description,
    cover_url: row.cover_url ?? undefined,
    code: row.code,
    creator_id: row.creator_id,
    status: row.status,
    created_at: row.created_at,
    member_count: row.member_count,
    is_joined: row.is_joined ?? undefined,
  }
}

function mapRoadmapWithLockStatus(rows: DbRoadmapRow[]): RoadmapNode[] {
  const sortedRows = [...rows].sort((a, b) => a.order - b.order)
  const result: RoadmapNode[] = []

  for (let index = 0; index < sortedRows.length; index += 1) {
    const row = sortedRows[index]
    if (!row) {
      continue
    }
    const previous = result[index - 1]
    const shouldLock =
      row.status === 'pending' &&
      previous !== undefined &&
      previous.status !== 'completed'

    result.push({
      id: row.id,
      workshop_id: row.workspace_id,
      title: row.title,
      description: row.description,
      order: row.order,
      status: shouldLock ? 'locked' : row.status,
      content: row.content as string | undefined,
    })
  }

  return result
}

async function getCurrentUserId(supabase: SupabaseClient): Promise<string> {
  const { data, error } = await supabase.auth.getUser()
  if (error || !data.user) {
    throw new Error(error?.message ?? '未登录')
  }
  return data.user.id
}

async function getNodeById(
  supabase: SupabaseClient,
  nodeId: string,
): Promise<DbNodeRow> {
  const { data, error } = await supabase
    .schema('juanleme')
    .from('workshop_nodes')
    .select('id, workspace_id, title, description, order')
    .eq('id', nodeId)
    .single()

  if (error || !data) {
    throw new Error(error?.message ?? '节点不存在')
  }

  return data as DbNodeRow
}

export function createSupabaseWorkshopService(
  supabase: SupabaseClient,
): IWorkshopService {
  return {
    async getWorkshops(): Promise<Workshop[]> {
      const { data, error } = await supabase
        .schema('juanleme')
        .from('v_workshop_summary')
        .select('*')

      if (error) {
        throw new Error(error.message)
      }

      return (data ?? []).map(row => toWorkshop(row as DbWorkshopSummaryRow))
    },

    async getWorkshop(id: string): Promise<Workshop | undefined> {
      const { data, error } = await supabase
        .schema('juanleme')
        .from('v_workshop_summary')
        .select('*')
        .eq('id', id)
        .maybeSingle()

      if (error) {
        throw new Error(error.message)
      }

      return data ? toWorkshop(data as DbWorkshopSummaryRow) : undefined
    },

    async getRoadmap(workshopId: string): Promise<RoadmapNode[]> {
      const { data, error } = await supabase
        .schema('juanleme')
        .rpc('get_workshop_roadmap', { p_workspace_id: workshopId })

      if (error) {
        throw new Error(error.message)
      }

      return mapRoadmapWithLockStatus((data ?? []) as DbRoadmapRow[])
    },

    async joinWorkshop(code: string): Promise<boolean> {
      const { error } = await supabase
        .schema('juanleme')
        .rpc('join_workshop', { p_code: code })

      if (error) {
        throw new Error(error.message)
      }

      return true
    },

    async createWorkshop(data: Partial<Workshop>): Promise<Workshop> {
      const title = data.title?.trim()
      if (!title) {
        throw new Error('工作坊标题不能为空')
      }

      const { data: workspaceId, error } = await supabase
        .schema('juanleme')
        .rpc('create_workspace', {
          p_title: title,
          p_description: data.description ?? '',
          p_cover_url: data.cover_url ?? null,
          p_status: data.status ?? 'draft',
        })

      if (error || !workspaceId) {
        throw new Error(error?.message ?? '创建工作坊失败')
      }

      const created = await this.getWorkshop(workspaceId as string)
      if (!created) {
        throw new Error('创建成功但读取工作坊失败')
      }

      return created
    },

    async updateNodeStatus(
      nodeId: string,
      status: RoadmapNode['status'],
    ): Promise<RoadmapNode> {
      const userId = await getCurrentUserId(supabase)
      const node = await getNodeById(supabase, nodeId)
      const client = supabase.schema('juanleme')

      if (status === 'completed') {
        const { error } = await client.rpc('complete_node', {
          p_node_id: nodeId,
          p_content: null,
        })
        if (error) {
          throw new Error(error.message)
        }
      } else if (status === 'in_progress') {
        const { error } = await client
          .from('documents')
          .upsert(
            {
              node_id: nodeId,
              user_id: userId,
              content: null,
              status: 'draft',
            },
            { onConflict: 'node_id,user_id' },
          )
        if (error) {
          throw new Error(error.message)
        }
      } else {
        const { error } = await client
          .from('documents')
          .delete()
          .eq('node_id', nodeId)
          .eq('user_id', userId)
        if (error) {
          throw new Error(error.message)
        }
      }

      const roadmap = await this.getRoadmap(node.workspace_id)
      const updated = roadmap.find(item => item.id === nodeId)
      if (updated) {
        return updated
      }

      return {
        id: node.id,
        workshop_id: node.workspace_id,
        title: node.title,
        description: node.description,
        order: node.order,
        status,
      }
    },

    async getSubmission(nodeId: string): Promise<UserSubmission | undefined> {
      const userId = await getCurrentUserId(supabase)
      const { data, error } = await supabase
        .schema('juanleme')
        .from('documents')
        .select('node_id, user_id, content, updated_at')
        .eq('node_id', nodeId)
        .eq('user_id', userId)
        .maybeSingle()

      if (error) {
        throw new Error(error.message)
      }

      if (!data) {
        return undefined
      }

      const row = data as DbSubmissionRow
      return {
        nodeId: row.node_id,
        userId: row.user_id,
        content: row.content,
        submittedAt: row.updated_at,
      }
    },

    async saveSubmission(submission: UserSubmission): Promise<UserSubmission> {
      const { error } = await supabase
        .schema('juanleme')
        .rpc('complete_node', {
          p_node_id: submission.nodeId,
          p_content: submission.content,
        })

      if (error) {
        throw new Error(error.message)
      }

      const latest = await this.getSubmission(submission.nodeId)
      if (latest) {
        return latest
      }

      return {
        ...submission,
        submittedAt: new Date().toISOString(),
      }
    },
  }
}
