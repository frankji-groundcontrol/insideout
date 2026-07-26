<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { useTimeAgo } from '@/composables/useTimeAgo'
import type { RoadmapTreeNode, RoadmapStatus } from '@/types'
import { PlusIcon, PencilSquareIcon, TrashIcon, CheckIcon, LockClosedIcon, SparklesIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

// One card on the roadmap canvas. Fixed 244×96 (see tidyTree CARD_W/H) so the
// tidy layout's leaf slots stay valid — edit / add-child render as popovers that
// overflow the card rather than growing it. Drag-to-reparent is handled by the
// parent canvas on the card's wrapper, so this component owns only CRUD.
const props = withDefaults(
  defineProps<{ node: RoadmapTreeNode; projectId: string; openUpward?: boolean; reviewing?: boolean }>(),
  { openUpward: false, reviewing: false },
)
const emit = defineEmits<{ (e: 'refresh'): void; (e: 'error', message: string): void }>()
const { t } = useI18n()
const ago = useTimeAgo()

// B2 freshness: the exact "updated X ago" (the card's granular signal). A branch
// untouched for >7d reads as "quiet" and its timestamp dims — the card's own echo
// of the edge hot/cold split, at a finer granularity (collab Decisions).
// ponytail: Date.now() in a computed isn't time-reactive; the tree re-renders on
// every focus-refresh, which is fresh enough for a relative label.
const isQuiet = computed(() => {
  if (!props.node.updatedAt) return false
  return Math.floor((Date.now() - new Date(props.node.updatedAt).getTime()) / 86_400_000) > 7
})

// B3 attribution (D10): the visible mark is the LAST editor (updated_by), with
// the tooltip naming both creator and editor. Pre-migration rows have neither
// and render a neutral "?" — never a fabricated identity. The roundel stays
// neutral, never vermilion (One Seal Rule reserves the chop for in-progress).
const editorInitial = computed(() => {
  const name = (props.node.editorName ?? props.node.creatorName ?? '').trim()
  return name.charAt(0).toUpperCase() || '?'
})
const attributionTip = computed(() =>
  t('roadmap.attribution', {
    creator: props.node.creatorName ?? t('roadmap.unknownAuthor'),
    editor: props.node.editorName ?? t('roadmap.unknownAuthor'),
  }),
)

const cardEl = ref<HTMLElement | null>(null)
const adding = ref(false)
const newChildTitle = ref('')
const editing = ref(false)
const editTitle = ref('')
const editDescription = ref('')
const busy = ref(false)
const expanding = ref(false)

// Status "seal" — the vermilion chop is reserved for in-progress (One Seal Rule).
// Whole-literal classes: Tailwind's static scan can't see dynamic interpolation.
const sealClasses: Record<RoadmapStatus, string> = {
  locked: 'border border-stroke-subtle bg-status-neutral-bg text-status-neutral-fg',
  pending: 'border border-stroke-strong bg-transparent text-fg-muted',
  in_progress: 'bg-seal text-carve',
  done: 'bg-status-success-bg text-status-success-fg',
}
const nextStatus: Record<RoadmapStatus, RoadmapStatus> = {
  locked: 'pending',
  pending: 'in_progress',
  in_progress: 'done',
  done: 'pending',
}

// ponytail: every mutation surfaces failure the same way — emit it to the canvas
// banner instead of swallowing it, so a failed call never leaves stale UI silent.
function reportError(e: unknown) {
  emit('error', (e as Error).message)
}

async function cycleStatus() {
  busy.value = true
  try {
    // Sparse body: only the status changes — title/description stay untouched (D9).
    await useServices().roadmap.update(props.node.id, { status: nextStatus[props.node.status] })
    emit('refresh')
  } catch (e) {
    reportError(e)
  } finally {
    busy.value = false
  }
}

async function addChild() {
  const title = newChildTitle.value.trim()
  if (!title) return
  busy.value = true
  try {
    await useServices().roadmap.create(props.projectId, { parentId: props.node.id, title })
    newChildTitle.value = ''
    adding.value = false
    emit('refresh')
  } catch (e) {
    reportError(e) // keep the form open with the draft so the user can retry
  } finally {
    busy.value = false
  }
}

function startEdit() {
  editTitle.value = props.node.title
  editDescription.value = props.node.description
  editing.value = true
}

async function saveEdit() {
  const title = editTitle.value.trim()
  if (!title) return
  busy.value = true
  try {
    // Sparse body: content edit only — status stays untouched (D9).
    await useServices().roadmap.update(props.node.id, { title, description: editDescription.value })
    editing.value = false
    emit('refresh')
    // The popover held focus; give it back to the card once it unmounts so the
    // keyboard user doesn't land on <body> (M6).
    await nextTick()
    cardEl.value?.focus({ preventScroll: true })
  } catch (e) {
    reportError(e) // keep the popover open with the draft
  } finally {
    busy.value = false
  }
}

function cancelEdit() {
  editing.value = false
  nextTick(() => cardEl.value?.focus({ preventScroll: true }))
}

async function removeNode() {
  if (!window.confirm(t('roadmap.deleteConfirm'))) return
  busy.value = true
  try {
    await useServices().roadmap.remove(props.node.id)
    // This card unmounts on refresh — move focus into the canvas first so the
    // keyboard user doesn't land on <body> (M6).
    ;(cardEl.value?.closest('[role="tree"]') as HTMLElement | null)?.focus({ preventScroll: true })
    emit('refresh')
  } catch (e) {
    reportError(e)
  } finally {
    busy.value = false
  }
}

async function expandWithAI() {
  expanding.value = true
  try {
    await useServices().roadmap.expand(props.node.id)
    emit('refresh')
  } catch {
    // Convenience, not a critical path — the tree simply stays as-is.
  } finally {
    expanding.value = false
  }
}
</script>

<template>
  <div
    ref="cardEl"
    tabindex="-1"
    class="group relative flex h-full w-full items-start gap-2.5 rounded-card border border-stroke-subtle bg-surface-raised p-3 text-left outline-none transition-colors hover:border-stroke-strong focus:border-stroke-strong"
  >
    <button
      type="button"
      :title="t(`roadmap.status.${node.status}`)"
      :aria-label="t(`roadmap.status.${node.status}`)"
      :disabled="busy || reviewing"
      class="mt-0.5 flex h-6 w-6 flex-none items-center justify-center rounded-md transition-transform hover:scale-110 disabled:cursor-default disabled:hover:scale-100"
      :class="sealClasses[node.status]"
      @click="cycleStatus"
    >
      <CheckIcon v-if="node.status === 'done'" class="h-4 w-4" />
      <LockClosedIcon v-else-if="node.status === 'locked'" class="h-3.5 w-3.5" />
      <span v-else-if="node.status === 'in_progress'" class="h-2 w-2 rounded-full bg-current" />
    </button>

    <!-- ponytail: overflow-hidden keeps a 2-line title + desc + freshness from
         spilling past the fixed 96px card border on the rare tallest card. -->
    <div class="min-w-0 flex-1 overflow-hidden">
      <p
        class="line-clamp-2 text-sm font-medium leading-snug text-fg-primary"
        :class="{ 'line-through opacity-60': node.status === 'done' }"
      >
        {{ node.title }}
      </p>
      <p v-if="node.description" class="mt-0.5 truncate text-xs text-fg-muted">{{ node.description }}</p>
      <p class="mt-1 flex items-center gap-1 text-[10px] leading-none" :class="isQuiet ? 'text-fg-muted/60' : 'text-fg-muted'">
        <span
          role="img"
          :aria-label="attributionTip"
          :title="attributionTip"
          class="flex h-3.5 w-3.5 flex-none items-center justify-center rounded-full bg-surface-sunken font-semibold text-fg-secondary"
        >{{ editorInitial }}</span>
        <span class="truncate">{{ t('roadmap.updatedAgo', { when: ago(node.updatedAt) }) }}</span>
      </p>
    </div>

    <!-- Hover actions (top-right). Kept off the drag path by the canvas's
         interactive-element guard. Reveal on hover (mouse), on focus-within
         (keyboard), and always on touch devices where there is no hover.
         B1 review mode: removed from the DOM (v-if), not just hidden — so no
         mutation control is reachable by Tab+Enter while read-only. -->
    <div v-if="!reviewing" class="absolute right-1.5 top-1.5 flex flex-none items-center gap-0.5 rounded-md bg-surface-raised/90 opacity-0 backdrop-blur-sm transition-opacity group-hover:opacity-100 group-focus-within:opacity-100 [@media(hover:none)]:opacity-100">
      <button
        type="button"
        class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-seal disabled:opacity-50"
        :title="t('roadmap.expandAI')"
        :aria-label="t('roadmap.expandAI')"
        :disabled="expanding"
        @click="expandWithAI"
      >
        <ArrowPathIcon v-if="expanding" class="h-4 w-4 animate-spin" />
        <SparklesIcon v-else class="h-4 w-4" />
      </button>
      <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.addChild')" :aria-label="t('roadmap.addChild')" @click="adding = !adding; editing = false">
        <PlusIcon class="h-4 w-4" />
      </button>
      <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.edit')" :aria-label="t('roadmap.edit')" @click="startEdit(); adding = false">
        <PencilSquareIcon class="h-4 w-4" />
      </button>
      <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-danger" :title="t('roadmap.delete')" :aria-label="t('roadmap.delete')" @click="removeNode">
        <TrashIcon class="h-4 w-4" />
      </button>
    </div>

    <!-- Edit popover: overflows the card so the 96px slot never shifts.
         Bottom-half cards open upward so the popover isn't clipped off the
         overflow-hidden viewport (M4). -->
    <form
      v-if="editing && !reviewing"
      class="absolute left-0 z-20 flex w-[260px] flex-col gap-2 rounded-card border border-stroke-subtle bg-surface-raised p-3 shadow-popover"
      :class="openUpward ? 'bottom-full mb-2' : 'top-full mt-2'"
      @submit.prevent="saveEdit"
      @pointerdown.stop
    >
      <input
        v-model="editTitle"
        type="text"
        autofocus
        class="w-full rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
      />
      <input
        v-model="editDescription"
        type="text"
        :placeholder="t('roadmap.descPlaceholder')"
        class="w-full rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-secondary focus:border-stroke-focus focus:outline-none"
      />
      <div class="flex gap-2">
        <button type="submit" :disabled="busy" class="rounded-control bg-btn px-2.5 py-1 text-xs font-medium text-btn-fg">{{ t('roadmap.save') }}</button>
        <button type="button" class="rounded-control px-2.5 py-1 text-xs text-fg-muted hover:text-fg-primary" @click="cancelEdit">{{ t('roadmap.cancel') }}</button>
      </div>
    </form>

    <!-- Add-child popover. -->
    <form
      v-if="adding && !reviewing"
      class="absolute left-0 z-20 flex w-[260px] gap-2 rounded-card border border-stroke-subtle bg-surface-raised p-3 shadow-popover"
      :class="openUpward ? 'bottom-full mb-2' : 'top-full mt-2'"
      @submit.prevent="addChild"
      @pointerdown.stop
    >
      <input
        v-model="newChildTitle"
        type="text"
        :placeholder="t('roadmap.childPlaceholder')"
        autofocus
        class="flex-1 rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
      />
      <button type="submit" :disabled="busy" class="rounded-control bg-btn px-2.5 py-1 text-xs font-medium text-btn-fg">{{ t('roadmap.add') }}</button>
    </form>
  </div>
</template>
