<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { RoadmapTreeNode, RoadmapStatus } from '@/types'
import BaseBadge from '@/components/common/BaseBadge.vue'
import { PlusIcon, PencilSquareIcon, TrashIcon, CheckIcon, LockClosedIcon, SparklesIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

const props = withDefaults(defineProps<{ node: RoadmapTreeNode; projectId: string; depth?: number }>(), { depth: 0 })
const emit = defineEmits<{ (e: 'refresh'): void }>()
const { t } = useI18n()

const adding = ref(false)
const newChildTitle = ref('')
const editing = ref(false)
const editTitle = ref('')
const editDescription = ref('')
const busy = ref(false)
const expanding = ref(false)

const statusTone: Record<RoadmapStatus, 'neutral' | 'warn' | 'info' | 'success'> = {
  locked: 'neutral',
  pending: 'warn',
  in_progress: 'info',
  done: 'success',
}
// Status "seal" (the vermilion chop is reserved for in-progress). Whole-literal
// classes — Tailwind's static scan can't see dynamic `bg-${x}` interpolation.
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

async function cycleStatus() {
  busy.value = true
  try {
    await useServices().roadmap.update(props.node.id, {
      title: props.node.title,
      description: props.node.description,
      status: nextStatus[props.node.status],
    })
    emit('refresh')
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
    await useServices().roadmap.update(props.node.id, {
      title,
      description: editDescription.value,
      status: props.node.status,
    })
    editing.value = false
    emit('refresh')
  } finally {
    busy.value = false
  }
}

async function removeNode() {
  if (!window.confirm(t('roadmap.deleteConfirm'))) return
  busy.value = true
  try {
    await useServices().roadmap.remove(props.node.id)
    emit('refresh')
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
    // Surface nothing extra — the tree simply stays as-is; the button is a
    // convenience, not a critical path.
  } finally {
    expanding.value = false
  }
}
</script>

<template>
  <div class="rm-node" :class="{ 'rm-child': depth > 0 }">
    <div class="group flex items-start gap-3 py-1.5">
      <button
        type="button"
        :title="t(`roadmap.status.${node.status}`)"
        :disabled="busy"
        class="mt-0.5 flex h-6 w-6 flex-none items-center justify-center rounded-md transition-transform hover:scale-110"
        :class="sealClasses[node.status]"
        @click="cycleStatus"
      >
        <CheckIcon v-if="node.status === 'done'" class="h-4 w-4" />
        <LockClosedIcon v-else-if="node.status === 'locked'" class="h-3.5 w-3.5" />
        <span v-else-if="node.status === 'in_progress'" class="h-2 w-2 rounded-full bg-current" />
      </button>

      <div class="min-w-0 flex-1">
        <div v-if="!editing" class="flex items-center gap-2">
          <p class="truncate font-medium text-fg-primary" :class="{ 'line-through opacity-60': node.status === 'done' }">
            {{ node.title }}
          </p>
          <BaseBadge :tone="statusTone[node.status]" class="flex-none">{{ t(`roadmap.status.${node.status}`) }}</BaseBadge>
          <div class="ml-auto flex flex-none items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
            <button
              type="button"
              class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-seal disabled:opacity-50"
              :title="t('roadmap.expandAI')"
              :disabled="expanding"
              @click="expandWithAI"
            >
              <ArrowPathIcon v-if="expanding" class="h-4 w-4 animate-spin" />
              <SparklesIcon v-else class="h-4 w-4" />
            </button>
            <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.addChild')" @click="adding = !adding">
              <PlusIcon class="h-4 w-4" />
            </button>
            <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.edit')" @click="startEdit">
              <PencilSquareIcon class="h-4 w-4" />
            </button>
            <button type="button" class="rounded p-1 text-fg-muted hover:bg-surface-sunken hover:text-fg-danger" :title="t('roadmap.delete')" @click="removeNode">
              <TrashIcon class="h-4 w-4" />
            </button>
          </div>
        </div>

        <form v-else class="flex flex-col gap-2" @submit.prevent="saveEdit">
          <input
            v-model="editTitle"
            type="text"
            class="w-full rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
          />
          <input
            v-model="editDescription"
            type="text"
            :placeholder="t('roadmap.descPlaceholder')"
            class="w-full rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-secondary focus:border-stroke-focus focus:outline-none"
          />
          <div class="flex gap-2">
            <button type="submit" class="rounded-control bg-btn px-2.5 py-1 text-xs font-medium text-btn-fg">{{ t('roadmap.save') }}</button>
            <button type="button" class="rounded-control px-2.5 py-1 text-xs text-fg-muted hover:text-fg-primary" @click="editing = false">{{ t('roadmap.cancel') }}</button>
          </div>
        </form>

        <p v-if="!editing && node.description" class="mt-0.5 text-sm text-fg-muted">{{ node.description }}</p>

        <form v-if="adding" class="mt-2 flex gap-2" @submit.prevent="addChild">
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
    </div>

    <div v-if="node.children.length" class="rm-children">
      <RoadmapNodeItem
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :project-id="projectId"
        :depth="depth + 1"
        @refresh="emit('refresh')"
      />
    </div>
  </div>
</template>

<style scoped>
/* Branched-tree connectors in the Ink & Seal language: a hairline vertical
   stroke down each parent's child column, with a horizontal stub into every
   child — siblings read as parallel branches off the same trunk. */
.rm-children {
  position: relative;
  margin-left: 0.75rem;
  padding-left: 1.5rem;
}
.rm-children::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 1.35rem;
  width: 2px;
  border-radius: 2px;
  background: rgb(var(--color-stroke-subtle));
}
.rm-child {
  position: relative;
}
.rm-child::before {
  content: '';
  position: absolute;
  left: -1.5rem;
  top: 1.35rem;
  width: 1.5rem;
  height: 2px;
  background: rgb(var(--color-stroke-subtle));
}
</style>
