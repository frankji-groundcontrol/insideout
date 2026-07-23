import type { IPrdService } from '@/types/services'
import { apiFetch } from './http'

export function createApiPrdService(): IPrdService {
  return {
    get(id) {
      return apiFetch(`/prds/${id}`)
    },
    updateSections(id, title, sections) {
      return apiFetch(`/prds/${id}`, { method: 'PATCH', body: { title, sections } })
    },
    listRevisions(id) {
      return apiFetch(`/prds/${id}/revisions`)
    },
    createRevision(id, note) {
      return apiFetch(`/prds/${id}/revisions`, { method: 'POST', body: { note } })
    },
    updateStatus(id, status) {
      return apiFetch(`/prds/${id}/status`, { method: 'POST', body: { status } })
    },
    build(id) {
      return apiFetch(`/prds/${id}/build`, { method: 'POST' })
    },
  }
}
