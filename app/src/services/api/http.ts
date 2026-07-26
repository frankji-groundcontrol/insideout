import { AiRateLimitError, AiServiceUnavailableError, RoadmapReplaceConflictError } from '@/types'

/**
 * Base URL for API calls. On the server we hit the Go backend directly
 * (no benefit in looping back through our own Nitro proxy); in the
 * browser we always go through the same-origin `/api/v1` proxy so the
 * httpOnly auth cookies are first-party. See
 * docs/plans/2026-07-20-go-rewrite/04-frontend.md §1.
 */
export function apiBase(): string {
  if (import.meta.server) {
    const cfg = useRuntimeConfig()
    return (cfg.apiInternalBase as string) || 'http://127.0.0.1:8080/api/v1'
  }
  return '/api/v1'
}

function forwardedCookieHeaders(): Record<string, string> {
  if (!import.meta.server) return {}
  return useRequestHeaders(['cookie']) as Record<string, string>
}

interface ApiErrorShape {
  error?: string
  code?: string
  retry_after_seconds?: number
  current_count?: number
  max_requests?: number
  circuit_state?: string
  liveCount?: number
}

export function toApiError(status: number | undefined, body: ApiErrorShape): Error {
  if (status === 409 && body.code === 'replace_conflict') {
    return new RoadmapReplaceConflictError(body.error || 'Roadmap replace not confirmed', body.liveCount ?? 0)
  }
  if (status === 429 && body.code === 'APP_THROTTLE') {
    return new AiRateLimitError(
      body.error || 'Rate limit exceeded',
      body.retry_after_seconds ?? 60,
      body.current_count,
      body.max_requests,
    )
  }
  // CIRCUIT_OPEN arrives pre-stream with a real 503; ANTHROPIC_RATE_LIMIT only
  // ever surfaces as a mid-stream SSE `error` event whose HTTP status is already
  // gone (undefined), so match it on code alone — otherwise the provider backoff
  // countdown never fires and Send stays enabled into an upstream rate limit.
  if ((status === 503 && body.code === 'CIRCUIT_OPEN') || body.code === 'ANTHROPIC_RATE_LIMIT') {
    return new AiServiceUnavailableError(
      body.error || 'AI service temporarily unavailable',
      body.retry_after_seconds ?? 30,
      body.circuit_state,
    )
  }
  return new Error(body.error || 'Request failed')
}

/** Thin wrapper over ofetch that forwards cookies during SSR and maps the API's error contract. */
export async function apiFetch<T>(path: string, opts: Record<string, unknown> = {}): Promise<T> {
  try {
    return await $fetch<T>(apiBase() + path, {
      credentials: 'include',
      ...opts,
      headers: { ...forwardedCookieHeaders(), ...(opts.headers as Record<string, string> | undefined) },
    })
  } catch (err: unknown) {
    const fetchErr = err as { response?: { status?: number }; data?: ApiErrorShape }
    throw toApiError(fetchErr.response?.status, fetchErr.data || {})
  }
}

/** Cookie headers + base URL for the one raw-fetch (SSE) call in coachService.ts. */
export function rawFetchInit(): { baseURL: string; cookieHeaders: Record<string, string> } {
  return { baseURL: apiBase(), cookieHeaders: forwardedCookieHeaders() }
}
