import { ref, type Ref } from 'vue'
import { useServices } from '@/composables/useServices'
import { AiRateLimitError, AiServiceUnavailableError, type CoachFact, type CoachMessage } from '@/types'

interface UseCoachStreamOptions {
  conversationId: Ref<string>
  onPrdUpdated?: (section: string) => void
  onStageChanged?: (stage: string) => void
}

/**
 * Wraps ICoachService.send()'s SSE callbacks with reactive state — message
 * history, live-streaming buffer, and the rate-limit countdown — mirroring
 * the pre-rewrite useAiConversation.ts pattern. See
 * docs/plans/2026-07-20-go-rewrite/04-frontend.md §1.
 */
export function useCoachStream(options: UseCoachStreamOptions) {
  const messages = ref<CoachMessage[]>([])
  const draft = ref('')
  const isThinking = ref(false)
  const streamingText = ref('')
  const isRateLimited = ref(false)
  const retryCountdown = ref(0)
  const facts = ref<CoachFact[]>([])
  let retryTimer: ReturnType<typeof setInterval> | null = null

  function clearRetryTimer() {
    if (retryTimer) {
      clearInterval(retryTimer)
      retryTimer = null
    }
  }

  function startRetryCountdown(seconds: number) {
    clearRetryTimer()
    retryCountdown.value = seconds
    isRateLimited.value = true
    retryTimer = setInterval(() => {
      if (retryCountdown.value <= 1) {
        clearRetryTimer()
        retryCountdown.value = 0
        isRateLimited.value = false
        return
      }
      retryCountdown.value -= 1
    }, 1000)
  }

  function localId(prefix: string): string {
    return `local_${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  }

  async function loadHistory() {
    messages.value = await useServices().coach.listMessages(options.conversationId.value)
  }

  async function send() {
    const content = draft.value.trim()
    if (!content || isThinking.value || isRateLimited.value) return

    messages.value.push({ id: localId('user'), role: 'user', content, createdAt: new Date().toISOString() })
    draft.value = ''
    isThinking.value = true
    streamingText.value = ''

    try {
      await useServices().coach.send(options.conversationId.value, content, {
        onDelta: (text) => {
          streamingText.value += text
        },
        onPrdUpdated: options.onPrdUpdated,
        onStageChanged: options.onStageChanged,
        onFactRecorded: (fact) => {
          facts.value.push(fact)
        },
        onDone: () => {
          messages.value.push({
            id: localId('assistant'),
            role: 'assistant',
            content: streamingText.value,
            createdAt: new Date().toISOString(),
          })
          streamingText.value = ''
        },
      })
    } catch (error) {
      if (error instanceof AiRateLimitError || error instanceof AiServiceUnavailableError) {
        startRetryCountdown(error.retryAfterSeconds)
      }
      streamingText.value = ''
      // The backend persists whatever streamed before the failure (with
      // an [interrupted] marker) rather than losing it — resync with
      // that instead of just dropping our local optimistic buffer.
      await loadHistory().catch(() => {})
    } finally {
      isThinking.value = false
    }
  }

  return { messages, draft, isThinking, streamingText, isRateLimited, retryCountdown, facts, loadHistory, send }
}
