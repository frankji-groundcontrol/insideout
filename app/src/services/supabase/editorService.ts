import { getSupabase } from '@/lib/supabase'
import type { EditorDraft } from '@/types'
import type { IEditorService } from '@/types/services'

type DbDocumentRow = {
  id: string
  node_id: string
  content: unknown
  updated_at: string
}

async function getCurrentUserId(): Promise<string> {
  const { data, error } = await getSupabase().auth.getUser()
  if (error || !data.user) {
    throw new Error(error?.message ?? '未登录')
  }
  return data.user.id
}

async function getRevisionCount(documentId: string): Promise<number> {
  const { count, error } = await getSupabase()
    .schema('juanleme')
    .from('document_revisions')
    .select('*', { count: 'exact', head: true })
    .eq('document_id', documentId)

  if (error) {
    throw new Error(error.message)
  }

  return count ?? 0
}

function toEditorDraft(row: DbDocumentRow, revision: number): EditorDraft {
  return {
    key: row.node_id,
    content: row.content,
    updatedAt: row.updated_at,
    revision,
  }
}

export function createSupabaseEditorService(): IEditorService {
  return {
    async loadDraft(key: EditorDraft['key']): Promise<EditorDraft | undefined> {
      const userId = await getCurrentUserId()
      const { data, error } = await getSupabase()
        .schema('juanleme')
        .from('documents')
        .select('id, node_id, content, updated_at')
        .eq('node_id', key)
        .eq('user_id', userId)
        .maybeSingle()

      if (error) {
        throw new Error(error.message)
      }

      if (!data) {
        return undefined
      }

      const row = data as DbDocumentRow
      const revision = await getRevisionCount(row.id)
      return toEditorDraft(row, revision)
    },

    async saveDraft(draft: EditorDraft): Promise<EditorDraft> {
      const userId = await getCurrentUserId()
      const { data, error } = await getSupabase()
        .schema('juanleme')
        .from('documents')
        .upsert(
          {
            node_id: draft.key,
            user_id: userId,
            content: draft.content,
            status: 'draft',
          },
          { onConflict: 'node_id,user_id' },
        )
        .select('id, node_id, content, updated_at')
        .single()

      if (error || !data) {
        throw new Error(error?.message ?? '保存草稿失败')
      }

      const row = data as DbDocumentRow
      const revision = await getRevisionCount(row.id)
      return toEditorDraft(row, revision)
    },

    async deleteDraft(key: EditorDraft['key']): Promise<void> {
      const userId = await getCurrentUserId()
      const { error } = await getSupabase()
        .schema('juanleme')
        .from('documents')
        .delete()
        .eq('node_id', key)
        .eq('user_id', userId)

      if (error) {
        throw new Error(error.message)
      }
    },
  }
}
