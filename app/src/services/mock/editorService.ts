import type { IEditorService } from '@/types/services'
import type { EditorDraft } from '@/types'
import { safeStorage } from '@/lib/safeStorage'

const DRAFT_PREFIX = 'draft:'

export function createMockEditorService(): IEditorService {
  return {
    async loadDraft(key: EditorDraft['key']): Promise<EditorDraft | undefined> {
      const raw = safeStorage.getItem(`${DRAFT_PREFIX}${key}`)
      return raw ? (JSON.parse(raw) as EditorDraft) : undefined
    },

    async saveDraft(draft: EditorDraft): Promise<EditorDraft> {
      safeStorage.setItem(`${DRAFT_PREFIX}${draft.key}`, JSON.stringify(draft))
      return draft
    },

    async deleteDraft(key: EditorDraft['key']): Promise<void> {
      safeStorage.removeItem(`${DRAFT_PREFIX}${key}`)
    },
  }
}
