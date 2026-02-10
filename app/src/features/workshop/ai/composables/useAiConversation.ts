import { computed, nextTick, ref, watch, type Ref } from 'vue'
import { services } from '@/services/registry'
import { useEditorStore } from '@/stores/editor'
import type { AiMessage } from '@/types'

interface UseAiConversationOptions {
  nodeId: Ref<string>
}

export function useAiConversation(options: UseAiConversationOptions) {
  const editorStore = useEditorStore()

  const draft = ref('')
  const isThinking = ref(false)
  const listEl = ref<HTMLElement | null>(null)
  const requestToken = ref(0)
  const messages = ref<AiMessage[]>([])
  const adoptedIds = ref<Set<string>>(new Set())

  const canSend = computed(() => draft.value.trim().length > 0 && !isThinking.value)

  function buildUserMessage(content: string): AiMessage {
    return {
      id: `local_user_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    }
  }

  function formatTime(iso: string): string {
    const date = new Date(iso)
    if (Number.isNaN(date.getTime())) {
      return '--:--'
    }
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  async function scrollToBottom() {
    await nextTick()
    if (!listEl.value) {
      return
    }
    listEl.value.scrollTop = listEl.value.scrollHeight
  }

  function adoptMessage(message: AiMessage) {
    if (adoptedIds.value.has(message.id)) {
      return
    }

    editorStore.enqueueInsert(message.content)
    const next = new Set(adoptedIds.value)
    next.add(message.id)
    adoptedIds.value = next
  }

  async function sendMessage() {
    const content = draft.value.trim()
    if (!content || isThinking.value) {
      return
    }

    const userMessage = buildUserMessage(content)
    messages.value.push(userMessage)
    draft.value = ''
    isThinking.value = true
    await scrollToBottom()

    const token = requestToken.value + 1
    requestToken.value = token
    const nodeAtRequest = options.nodeId.value

    try {
      const reply = await services.ai.reply(nodeAtRequest, content)
      if (token !== requestToken.value || nodeAtRequest !== options.nodeId.value) {
        return
      }
      messages.value.push(reply)
    } catch (error) {
      console.error('AI reply failed', error)
    } finally {
      if (token === requestToken.value) {
        isThinking.value = false
      }
      await scrollToBottom()
    }
  }

  watch(
    () => options.nodeId.value,
    async () => {
      requestToken.value += 1
      draft.value = ''
      isThinking.value = false
      messages.value = []
      adoptedIds.value = new Set()
      await scrollToBottom()
    },
    { immediate: true },
  )

  watch(
    () => [messages.value.length, isThinking.value],
    async () => {
      await scrollToBottom()
    },
  )

  return {
    draft,
    isThinking,
    listEl,
    messages,
    adoptedIds,
    canSend,
    formatTime,
    adoptMessage,
    sendMessage,
  }
}
