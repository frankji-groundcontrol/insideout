import type { IExportService } from '@/types/services'
import { apiBase, toApiError, rawFetchInit } from './http'

// Export is a direct-download GET, not JSON — plain fetch, not $fetch,
// so we control the raw body/content-type (D8: on-demand render, no
// object storage, no job model).
export function createApiExportService(): IExportService {
  return {
    async download(prdId, format) {
      const { cookieHeaders } = rawFetchInit()
      const res = await fetch(`${apiBase()}/prds/${prdId}/export?format=${format}`, {
        credentials: 'include',
        headers: cookieHeaders,
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw toApiError(res.status, body)
      }
      const contentType = res.headers.get('Content-Type') || 'text/plain'
      const content = await res.text()
      return { content, contentType }
    },
  }
}
