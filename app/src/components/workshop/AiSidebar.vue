<script setup lang="ts">
/* biome-ignore-all assist/source/organizeImports: Vue SFC 按语义分组导入 */
/* biome-ignore-all lint/correctness/noUnusedImports: 模板中会使用导入内容 */
/* biome-ignore-all lint/correctness/noUnusedVariables: 模板中会使用变量 */
import { toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { PaperAirplaneIcon } from '@heroicons/vue/24/outline'
import { useAiConversation } from '@/features/workshop/ai/composables/useAiConversation'

interface Props {
  nodeId: string
}

const props = defineProps<Props>()
const { t } = useI18n()
const nodeId = toRef(props, 'nodeId')
const {
  draft,
  isThinking,
  listEl,
  messages,
  retryCountdown,
  isRateLimited,
  adoptedIds,
  hasNode,
  canSend,
  formatTime,
  adoptMessage,
  sendMessage,
  startNewConversation,
} =
  useAiConversation({ nodeId })

// 通过显式引用避免 noUnusedLocals 对模板 ref 的误判。
void listEl
</script>

<template>
  <aside class="ai-sidebar flex h-full min-h-0 flex-col">
    <header class="border-b border-border-default px-4 py-4 sm:px-5">
      <div class="flex items-center justify-between gap-3">
        <h3 class="text-lg font-bold text-content-primary">{{ t('ai.title') }}</h3>
        <button
          type="button"
          class="new-chat-btn"
          :disabled="!hasNode || isThinking"
          @click="startNewConversation"
        >
          {{ t('ai.newConversation') }}
        </button>
      </div>
      <div v-if="isRateLimited" class="rate-limit-banner mt-3">
        {{ t('ai.rateLimited', { seconds: retryCountdown }) }}
      </div>
    </header>

    <section ref="listEl" class="flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-5">
      <div
        v-if="messages.length === 0 && !isThinking"
        class="rounded-none border border-dashed border-border-default bg-surface-elevated px-4 py-6 text-center text-sm text-content-secondary shadow-brutal-sm"
      >
        {{ t('ai.emptyState') }}
      </div>

      <div
        v-for="message in messages"
        :key="message.id"
        class="flex items-start gap-3"
        :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
      >
        <div
          v-if="message.role === 'assistant'"
          class="avatar ai-avatar"
        >
          AI
        </div>

        <div class="max-w-[85%] sm:max-w-[78%]">
          <div
            class="bubble"
            :class="message.role === 'user' ? 'user-bubble' : 'assistant-bubble'"
          >
            {{ message.content }}
          </div>
          <p
            class="mt-1 text-xs text-content-secondary"
            :class="message.role === 'user' ? 'text-right' : 'text-left'"
          >
            {{ formatTime(message.timestamp) }}
          </p>

          <button
            v-if="message.role === 'assistant'"
            type="button"
            class="adopt-btn mt-2"
            :disabled="adoptedIds.has(message.id)"
            @click="adoptMessage(message)"
          >
            {{ adoptedIds.has(message.id) ? t('ai.adopted') : t('ai.adopt') }}
          </button>
        </div>

        <div
          v-if="message.role === 'user'"
          class="avatar user-avatar"
        >
          U
        </div>
      </div>

      <div v-if="isThinking" class="flex items-start gap-3">
        <div class="avatar ai-avatar">AI</div>
        <div class="assistant-bubble bubble typing min-w-[132px]">
          <span>{{ t('ai.thinking') }}</span>
          <span class="typing-dots" aria-hidden="true">
            <i></i>
            <i></i>
            <i></i>
          </span>
        </div>
      </div>
    </section>

    <footer class="border-t border-border-default p-3 sm:p-4">
      <form class="flex items-center gap-2" @submit.prevent="sendMessage">
        <input
          v-model="draft"
          type="text"
          class="input-field"
          :placeholder="hasNode ? t('ai.placeholder') : t('ai.selectTaskFirst')"
          :disabled="!hasNode || isThinking || isRateLimited"
          @keydown.enter.prevent="sendMessage"
        />
        <button type="submit" class="send-btn" :disabled="!canSend">
          <PaperAirplaneIcon class="h-4 w-4" />
          <span class="hidden sm:inline">{{ t('ai.send') }}</span>
        </button>
      </form>
      <p v-if="!hasNode" class="mt-2 text-xs text-content-secondary">
        {{ t('ai.selectTaskFirst') }}
      </p>
    </footer>
  </aside>
</template>

<style scoped>
.ai-sidebar {
  background: var(--jlm-surface-base);
}

.avatar {
  display: inline-flex;
  height: 1.9rem;
  width: 1.9rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.user-avatar {
  color: var(--jlm-content-inverse);
  background: var(--jlm-interactive-default);
  box-shadow: 3px 3px 0 0 var(--jlm-shadow-color);
}

.ai-avatar {
  color: var(--jlm-accent-primary);
  background: var(--jlm-surface-muted);
}

.bubble {
  border-radius: 0;
  padding: 0.7rem 0.82rem;
  font-size: 0.875rem;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
}

.user-bubble {
  color: var(--jlm-content-inverse);
  background: var(--jlm-interactive-default);
  box-shadow: 3px 3px 0 0 var(--jlm-shadow-color);
}

.assistant-bubble {
  border: 1px solid var(--jlm-border-default);
  color: var(--jlm-content-primary);
  background: var(--jlm-surface-elevated);
}

.adopt-btn {
  border: 1px solid var(--jlm-interactive-default);
  border-radius: 0;
  color: var(--jlm-interactive-default);
  background: var(--jlm-surface-elevated);
  padding: 0.18rem 0.72rem;
  font-size: 0.74rem;
  line-height: 1.5;
  transition: all 160ms ease;
}

.adopt-btn:hover:not(:disabled) {
  border-color: var(--jlm-interactive-hover);
  color: var(--jlm-interactive-hover);
  background: var(--jlm-surface-muted);
}

.adopt-btn:disabled {
  cursor: not-allowed;
  color: var(--jlm-content-secondary);
  border-color: var(--jlm-border-default);
  background: var(--jlm-surface-muted);
}

.new-chat-btn {
  border: 1px solid var(--jlm-border-default);
  border-radius: 0;
  color: var(--jlm-content-primary);
  background: var(--jlm-surface-elevated);
  padding: 0.28rem 0.62rem;
  font-size: 0.72rem;
  line-height: 1.4;
  transition: all 160ms ease;
}

.new-chat-btn:hover:not(:disabled) {
  border-color: var(--jlm-interactive-default);
  color: var(--jlm-interactive-default);
}

.new-chat-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.rate-limit-banner {
  border: 1px solid color-mix(in srgb, var(--jlm-warning, #d97706) 45%, var(--jlm-border-default));
  background: color-mix(in srgb, var(--jlm-warning, #d97706) 14%, var(--jlm-surface-elevated));
  color: var(--jlm-content-primary);
  padding: 0.45rem 0.55rem;
  font-size: 0.74rem;
  line-height: 1.4;
}

.typing {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.typing-dots {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
}

.typing-dots i {
  height: 0.28rem;
  width: 0.28rem;
  border-radius: 9999px;
  background: var(--jlm-content-secondary);
  animation: dotPulse 1s infinite ease-in-out;
}

.typing-dots i:nth-child(2) {
  animation-delay: 0.15s;
}

.typing-dots i:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes dotPulse {
  0%,
  80%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-2px);
  }
}

.input-field {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--jlm-border-default);
  border-radius: 0;
  color: var(--jlm-content-primary);
  background: var(--jlm-surface-muted);
  padding: 0.62rem 0.8rem;
  font-size: 0.875rem;
  box-shadow: inset 3px 3px 0 0 color-mix(in srgb, var(--jlm-shadow-color) 5%, transparent);
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}

.input-field::placeholder {
  color: var(--jlm-content-secondary);
}

.input-field:focus {
  outline: none;
  border-color: var(--jlm-interactive-default);
  box-shadow: inset 3px 3px 0 0 color-mix(in srgb, var(--jlm-shadow-color) 5%, transparent);
}

.send-btn {
  display: inline-flex;
  height: 2.4rem;
  flex-shrink: 0;
  align-items: center;
  gap: 0.35rem;
  border: 0;
  border-radius: 0;
  color: var(--jlm-content-inverse);
  background: var(--jlm-interactive-default);
  box-shadow: 3px 3px 0 0 var(--jlm-shadow-color);
  padding: 0 0.85rem;
  font-size: 0.82rem;
  font-weight: 600;
  transition: transform 160ms ease, filter 160ms ease, opacity 160ms ease;
}

.send-btn:hover:not(:disabled) {
  transform: translate(-1px, -1px);
  background: var(--jlm-interactive-hover);
}

.send-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
  transform: none;
}
</style>
