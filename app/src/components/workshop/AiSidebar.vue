<script setup lang="ts">
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
const { draft, isThinking, listEl, messages, adoptedIds, canSend, formatTime, adoptMessage, sendMessage } =
  useAiConversation({ nodeId })

// 通过显式引用避免 noUnusedLocals 对模板 ref 的误判。
void listEl
</script>

<template>
  <aside class="ai-sidebar flex h-full min-h-0 flex-col">
    <header class="border-b border-white/60 px-4 py-4 dark:border-gray-700/70 sm:px-5">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('ai.title') }}</h3>
    </header>

    <section ref="listEl" class="flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-5">
      <div
        v-if="messages.length === 0 && !isThinking"
        class="rounded-2xl border border-dashed border-gray-300/90 bg-white/70 px-4 py-6 text-center text-sm text-gray-500 shadow-sm dark:border-gray-600/80 dark:bg-gray-900/50 dark:text-gray-400"
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
            class="mt-1 text-xs text-gray-500 dark:text-gray-400"
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

    <footer class="border-t border-white/65 p-3 dark:border-gray-700/70 sm:p-4">
      <form class="flex items-center gap-2" @submit.prevent="sendMessage">
        <input
          v-model="draft"
          type="text"
          class="input-field"
          :placeholder="t('ai.placeholder')"
          @keydown.enter.prevent="sendMessage"
        />
        <button type="submit" class="send-btn" :disabled="!canSend">
          <PaperAirplaneIcon class="h-4 w-4" />
          <span class="hidden sm:inline">{{ t('ai.send') }}</span>
        </button>
      </form>
    </footer>
  </aside>
</template>

<style scoped>
.ai-sidebar {
  background:
    radial-gradient(circle at 8% 8%, rgb(199 210 254 / 30%), transparent 42%),
    radial-gradient(circle at 95% 88%, rgb(191 219 254 / 26%), transparent 48%),
    linear-gradient(180deg, rgb(248 250 252), rgb(241 245 249));
}

:global(.dark) .ai-sidebar {
  background:
    radial-gradient(circle at 10% 10%, rgb(30 58 138 / 26%), transparent 45%),
    radial-gradient(circle at 92% 92%, rgb(15 23 42 / 36%), transparent 52%),
    linear-gradient(180deg, rgb(15 23 42), rgb(3 7 18));
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
  color: rgb(255 255 255);
  background: linear-gradient(135deg, rgb(79 70 229), rgb(37 99 235));
  box-shadow: 0 8px 18px rgb(37 99 235 / 28%);
}

.ai-avatar {
  color: rgb(30 64 175);
  background: rgb(224 231 255);
}

:global(.dark) .ai-avatar {
  color: rgb(191 219 254);
  background: rgb(30 41 59);
}

.bubble {
  border-radius: 1rem;
  padding: 0.7rem 0.82rem;
  font-size: 0.875rem;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
}

.user-bubble {
  border-top-right-radius: 0.35rem;
  color: rgb(255 255 255);
  background: linear-gradient(140deg, rgb(79 70 229), rgb(29 78 216));
  box-shadow: 0 10px 24px rgb(37 99 235 / 28%);
}

.assistant-bubble {
  border: 1px solid rgb(229 231 235);
  border-top-left-radius: 0.35rem;
  color: rgb(17 24 39);
  background: rgb(255 255 255 / 92%);
}

:global(.dark) .assistant-bubble {
  border-color: rgb(71 85 105 / 60%);
  color: rgb(226 232 240);
  background: rgb(15 23 42 / 72%);
}

.adopt-btn {
  border: 1px solid rgb(99 102 241 / 45%);
  border-radius: 9999px;
  color: rgb(67 56 202);
  background: rgb(238 242 255 / 75%);
  padding: 0.18rem 0.72rem;
  font-size: 0.74rem;
  line-height: 1.5;
  transition: all 160ms ease;
}

.adopt-btn:hover:not(:disabled) {
  border-color: rgb(79 70 229);
  background: rgb(224 231 255);
}

.adopt-btn:disabled {
  cursor: not-allowed;
  color: rgb(107 114 128);
  border-color: rgb(209 213 219);
  background: rgb(243 244 246);
}

:global(.dark) .adopt-btn {
  color: rgb(199 210 254);
  border-color: rgb(129 140 248 / 45%);
  background: rgb(30 41 59 / 70%);
}

:global(.dark) .adopt-btn:hover:not(:disabled) {
  border-color: rgb(129 140 248 / 80%);
  background: rgb(51 65 85 / 90%);
}

:global(.dark) .adopt-btn:disabled {
  color: rgb(148 163 184);
  border-color: rgb(71 85 105);
  background: rgb(30 41 59);
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
  background: rgb(100 116 139);
  animation: dotPulse 1s infinite ease-in-out;
}

:global(.dark) .typing-dots i {
  background: rgb(148 163 184);
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
  border: 1px solid rgb(209 213 219);
  border-radius: 0.8rem;
  color: rgb(17 24 39);
  background: rgb(255 255 255 / 92%);
  padding: 0.62rem 0.8rem;
  font-size: 0.875rem;
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 85%);
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}

.input-field::placeholder {
  color: rgb(107 114 128);
}

.input-field:focus {
  outline: none;
  border-color: rgb(99 102 241);
  box-shadow: 0 0 0 3px rgb(99 102 241 / 20%);
}

:global(.dark) .input-field {
  border-color: rgb(71 85 105 / 70%);
  color: rgb(241 245 249);
  background: rgb(15 23 42 / 88%);
}

:global(.dark) .input-field::placeholder {
  color: rgb(148 163 184);
}

.send-btn {
  display: inline-flex;
  height: 2.4rem;
  flex-shrink: 0;
  align-items: center;
  gap: 0.35rem;
  border: 0;
  border-radius: 0.8rem;
  color: rgb(255 255 255);
  background: linear-gradient(140deg, rgb(79 70 229), rgb(37 99 235));
  padding: 0 0.85rem;
  font-size: 0.82rem;
  font-weight: 600;
  transition: transform 160ms ease, filter 160ms ease, opacity 160ms ease;
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  filter: brightness(1.06);
}

.send-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
  transform: none;
}
</style>
