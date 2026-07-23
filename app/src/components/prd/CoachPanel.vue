<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XMarkIcon,
  ChatBubbleLeftRightIcon,
  SparklesIcon,
} from '@heroicons/vue/24/outline'
import { useServices } from '@/composables/useServices'
import { useCoachStream } from '@/composables/useCoachStream'
import type { CoachStage } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import MarkdownBody from '@/components/common/MarkdownBody.vue'

const props = defineProps<{ prdId: string; open: boolean }>()
const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'prd-updated'): void
}>()
const { t, tm, rt } = useI18n()

const STAGES: CoachStage[] = ['clarify', 'draft', 'critique', 'finalize']

const conversationId = ref<string | null>(null)
const currentStage = ref<CoachStage>('clarify')
const loadingConv = ref(true)

const stream = useCoachStream({
  conversationId: computed(() => conversationId.value ?? ''),
  onPrdUpdated: () => emit('prd-updated'),
  onStageChanged: (stage) => {
    currentStage.value = stage as CoachStage
  },
})

async function load() {
  loadingConv.value = true
  const conv = await useServices().coach.getConversationForPrd(props.prdId)
  conversationId.value = conv?.id ?? null
  if (conv?.stage) currentStage.value = conv.stage
  loadingConv.value = false
  if (conversationId.value) await stream.loadHistory()
}
onMounted(load)

const stageIndex = computed(() => Math.max(0, STAGES.indexOf(currentStage.value)))

// Stage-keyed suggested prompts, rendered as clickable cards in the thread.
const suggestions = computed<string[]>(() => {
  const raw = tm(`coach.suggest.${currentStage.value}`) as unknown
  if (!Array.isArray(raw)) return []
  return raw.map((item) => rt(item as never))
})

function useSuggestion(text: string) {
  if (stream.isThinking.value || stream.isRateLimited.value || !conversationId.value) return
  stream.draft.value = text
  stream.send()
}

const close = () => emit('update:open', false)
const openPanel = () => emit('update:open', true)

const composerDisabled = computed(
  () => !conversationId.value || stream.isThinking.value || stream.isRateLimited.value,
)

// Keep the thread pinned to the latest message / streaming token.
const threadRef = ref<HTMLElement | null>(null)
watch(
  () => [stream.messages.value.length, stream.streamingText.value.length],
  async () => {
    await nextTick()
    const el = threadRef.value
    if (el) el.scrollTop = el.scrollHeight
  },
)
</script>

<template>
  <!-- Mobile backdrop (drawer mode only) -->
  <Transition
    enter-active-class="transition duration-200 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition duration-150 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="open"
      class="fixed inset-0 z-overlay bg-surface-overlay lg:hidden"
      aria-hidden="true"
      @click="close"
    />
  </Transition>

  <!-- Persistent reopen affordance when collapsed -->
  <button
    v-if="!open"
    type="button"
    :aria-label="t('coach.openCoach')"
    class="fixed right-0 top-1/2 z-overlay flex -translate-y-1/2 flex-col items-center gap-2 rounded-l-card border border-r-0 border-stroke-subtle bg-surface-raised px-2 py-3 text-fg-muted shadow-popover transition-colors hover:text-seal"
    @click="openPanel"
  >
    <ChatBubbleLeftRightIcon class="h-5 w-5" />
    <span class="text-xs font-medium [writing-mode:vertical-rl]">{{ t('coach.title') }}</span>
  </button>

  <!-- Panel: docked sidebar on lg+, overlay drawer below -->
  <aside
    :aria-label="t('coach.title')"
    class="fixed inset-y-0 right-0 z-overlay flex w-[min(440px,94vw)] flex-col border-l border-stroke-subtle bg-surface-raised shadow-modal transition-transform duration-300 ease-out lg:sticky lg:top-16 lg:z-auto lg:h-[calc(100vh-4rem)] lg:w-[440px] lg:shrink-0 lg:self-start lg:shadow-none lg:transition-none"
    :class="open ? 'translate-x-0' : 'translate-x-full lg:hidden'"
  >
    <!-- Header: title + detach + stage stepper -->
    <div class="border-b border-stroke-subtle px-4 pb-3 pt-4">
      <div class="flex items-center justify-between">
        <h3 class="font-serif text-base font-semibold text-fg-primary">{{ t('coach.title') }}</h3>
        <button
          type="button"
          :aria-label="t('coach.closeCoach')"
          class="rounded-control p-1 text-fg-muted transition-colors hover:bg-surface-sunken hover:text-fg-primary"
          @click="close"
        >
          <XMarkIcon class="h-5 w-5" />
        </button>
      </div>

      <ol v-if="conversationId" class="mt-3 flex items-center">
        <template v-for="(stage, i) in STAGES" :key="stage">
          <li class="flex min-w-0 items-center">
            <span
              class="h-2.5 w-2.5 shrink-0 rounded-full"
              :class="
                i < stageIndex
                  ? 'bg-fg-brand'
                  : i === stageIndex
                    ? 'bg-seal ring-2 ring-seal/30'
                    : 'border border-stroke-strong bg-surface-raised'
              "
            />
            <span
              class="ml-1.5 truncate text-[11px]"
              :class="i === stageIndex ? 'font-semibold text-fg-primary' : 'text-fg-muted'"
            >
              {{ t(`coach.stage.${stage}`) }}
            </span>
          </li>
          <li
            v-if="i < STAGES.length - 1"
            class="mx-2 h-px flex-1"
            :class="i < stageIndex ? 'bg-fg-brand' : 'bg-stroke-subtle'"
            aria-hidden="true"
          />
        </template>
      </ol>
    </div>

    <!-- Fact ledger (collapsible) -->
    <details v-if="stream.facts.value.length" class="border-b border-stroke-subtle px-4 py-2">
      <summary class="cursor-pointer text-xs font-semibold text-fg-secondary">
        {{ t('coach.facts.title') }} ({{ stream.facts.value.length }})
      </summary>
      <ul class="mt-2 space-y-1">
        <li
          v-for="f in stream.facts.value"
          :key="f.id"
          class="rounded-control bg-surface-sunken px-2 py-1 text-xs text-fg-primary"
        >
          <span class="font-medium">
            [{{ f.status === 'assumed' ? t('coach.facts.assumption') : f.kind }}]
          </span>
          {{ f.text }}
        </li>
      </ul>
    </details>

    <!-- Conversation thread -->
    <div ref="threadRef" class="flex-1 space-y-3 overflow-y-auto px-4 py-4">
      <p v-if="!conversationId && !loadingConv" class="text-sm text-fg-muted">
        {{ t('coach.noConversation') }}
      </p>
      <template v-else>
        <div
          v-for="m in stream.messages.value"
          :key="m.id"
          class="max-w-[85%] rounded-control px-3 py-2 text-sm leading-relaxed"
          :class="m.role === 'user' ? 'ml-auto bg-btn text-btn-fg' : 'bg-surface-sunken text-fg-primary'"
        >
          <MarkdownBody :content="m.content" />
        </div>

        <div
          v-if="stream.streamingText.value"
          class="max-w-[85%] rounded-control bg-surface-sunken px-3 py-2 text-sm leading-relaxed text-fg-primary"
        >
          <MarkdownBody :content="stream.streamingText.value" />
          <span class="coach-caret" aria-hidden="true" />
        </div>
        <p v-if="stream.isThinking.value && !stream.streamingText.value" class="text-xs text-fg-muted">
          {{ t('coach.thinking') }}
        </p>

        <!-- Suggested-prompt cards, inline in the conversation -->
        <div v-if="conversationId && !stream.isThinking.value && suggestions.length" class="pt-1">
          <p class="mb-2 text-xs font-medium text-fg-muted">{{ t('coach.suggestTitle') }}</p>
          <div class="space-y-2">
            <button
              v-for="s in suggestions"
              :key="s"
              type="button"
              class="flex w-full items-start gap-2 rounded-card border border-stroke-subtle bg-surface-raised px-3 py-2 text-left text-sm text-fg-secondary transition-colors hover:border-stroke-strong hover:bg-surface-sunken hover:text-fg-primary"
              @click="useSuggestion(s)"
            >
              <SparklesIcon class="mt-0.5 h-4 w-4 shrink-0 text-seal" />
              <span>{{ s }}</span>
            </button>
          </div>
        </div>
      </template>
    </div>

    <!-- Composer -->
    <div class="border-t border-stroke-subtle p-3">
      <p v-if="stream.isRateLimited.value" class="mb-2 text-xs text-status-warn-fg">
        {{ t('coach.rateLimited', { seconds: stream.retryCountdown.value }) }}
      </p>
      <form class="flex gap-2" @submit.prevent="stream.send()">
        <input
          v-model="stream.draft.value"
          :placeholder="t('coach.placeholder')"
          :disabled="composerDisabled"
          class="min-w-0 flex-1 rounded-control border border-stroke-subtle bg-surface-sunken px-3 py-2 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none focus:ring-1 focus:ring-stroke-focus"
        />
        <BaseButton type="submit" size="sm" :disabled="composerDisabled || !stream.draft.value.trim()">
          {{ t('coach.send') }}
        </BaseButton>
      </form>
    </div>
  </aside>
</template>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .coach-caret {
    display: inline-block;
    width: 1px;
    height: 1em;
    margin-left: 2px;
    vertical-align: text-bottom;
    background: currentColor;
    animation: coach-caret-blink 1s steps(1) infinite;
  }
  @keyframes coach-caret-blink {
    0%,
    50% {
      opacity: 1;
    }
    50.01%,
    100% {
      opacity: 0;
    }
  }
}
</style>
