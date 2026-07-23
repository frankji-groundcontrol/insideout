import type { ICoachService } from '@/types/services'
import { apiFetch, apiBase, toApiError, rawFetchInit } from './http'

interface SseEvent {
  event: string
  data: string
}

/** Splits an SSE byte stream into `{event, data}` frames as they complete. */
function parseSseChunk(buffer: string): { events: SseEvent[]; rest: string } {
  const events: SseEvent[] = []
  let rest = buffer
  let sep: number
  while ((sep = rest.indexOf('\n\n')) !== -1) {
    const raw = rest.slice(0, sep)
    rest = rest.slice(sep + 2)
    let eventName = 'message'
    let data = ''
    for (const line of raw.split('\n')) {
      if (line.startsWith('event: ')) eventName = line.slice(7)
      else if (line.startsWith('data: ')) data = line.slice(6)
    }
    if (data) events.push({ event: eventName, data })
  }
  return { events, rest }
}

export function createApiCoachService(): ICoachService {
  return {
    getConversation(id) {
      return apiFetch(`/conversations/${id}`)
    },
    async getConversationForPrd(prdId) {
      try {
        return await apiFetch(`/prds/${prdId}/conversation`)
      } catch {
        return null
      }
    },
    listMessages(id) {
      return apiFetch(`/conversations/${id}/messages`)
    },
    async send(conversationId, content, handlers) {
      const { cookieHeaders } = rawFetchInit()
      const res = await fetch(`${apiBase()}/conversations/${conversationId}/messages`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...cookieHeaders },
        body: JSON.stringify({ content }),
      })

      if (!res.ok || !res.body) {
        const body = await res.json().catch(() => ({}))
        throw toApiError(res.status, body)
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const { events, rest } = parseSseChunk(buffer)
        buffer = rest
        for (const { event, data } of events) {
          const parsed = JSON.parse(data)
          if (event === 'delta') handlers.onDelta?.(parsed.text)
          else if (event === 'prd_updated') handlers.onPrdUpdated?.(parsed.section)
          else if (event === 'stage_changed') handlers.onStageChanged?.(parsed.stage)
          else if (event === 'fact_recorded') handlers.onFactRecorded?.(parsed)
          else if (event === 'error') throw toApiError(undefined, parsed)
        }
      }
      handlers.onDone?.()
    },
  }
}
