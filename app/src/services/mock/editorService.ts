import type { IEditorService } from '@/types/services'
import type { EditorDraft } from '@/types'

const DRAFT_PREFIX = 'draft:'

export function createMockEditorService(): IEditorService {
  return {
    async loadDraft(key: EditorDraft['key']): Promise<EditorDraft | undefined> {
      const raw = localStorage.getItem(`${DRAFT_PREFIX}${key}`)
      return raw ? (JSON.parse(raw) as EditorDraft) : undefined
    },

    async saveDraft(draft: EditorDraft): Promise<EditorDraft> {
      localStorage.setItem(`${DRAFT_PREFIX}${draft.key}`, JSON.stringify(draft))
      return draft
    },

    async deleteDraft(key: EditorDraft['key']): Promise<void> {
      localStorage.removeItem(`${DRAFT_PREFIX}${key}`)
    },
  }
}
