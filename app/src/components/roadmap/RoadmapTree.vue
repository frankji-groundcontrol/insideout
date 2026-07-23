<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import type { RoadmapNode, RoadmapTreeNode } from '@/types'
import RoadmapNodeItem from './RoadmapNodeItem.vue'
import { MapIcon, PlusIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{ projectId: string }>()
const { t } = useI18n()

const loading = ref(true)
const nodes = ref<RoadmapNode[]>([])
const newRootTitle = ref('')
const busy = ref(false)

async function load() {
  loading.value = true
  try {
    nodes.value = await useServices().roadmap.list(props.projectId)
  } finally {
    loading.value = false
  }
}
onMounted(load)

// Assemble the flat node list into a sorted forest (roots + nested children).
const tree = computed<RoadmapTreeNode[]>(() => {
  const map = new Map<string, RoadmapTreeNode>()
  for (const n of nodes.value) map.set(n.id, { ...n, children: [] })
  const roots: RoadmapTreeNode[] = []
  for (const n of map.values()) {
    if (n.parentId && map.has(n.parentId)) {
      map.get(n.parentId)!.children.push(n)
    } else {
      roots.push(n)
    }
  }
  const sortRec = (list: RoadmapTreeNode[]) => {
    list.sort((a, b) => a.position - b.position)
    for (const x of list) sortRec(x.children)
  }
  sortRec(roots)
  return roots
})

const total = computed(() => nodes.value.length)
const doneCount = computed(() => nodes.value.filter((n) => n.status === 'done').length)
const progressPct = computed(() => (total.value === 0 ? 0 : Math.round((doneCount.value / total.value) * 100)))

async function addRoot() {
  const title = newRootTitle.value.trim()
  if (!title) return
  busy.value = true
  try {
    await useServices().roadmap.create(props.projectId, { title })
    newRootTitle.value = ''
    await load()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="flex items-center font-serif text-lg font-semibold text-fg-primary">
        <MapIcon class="mr-2 h-5 w-5 text-seal" />
        {{ t('roadmap.title') }}
      </h2>
      <div v-if="total > 0" class="flex items-center gap-2">
        <div class="h-1.5 w-28 overflow-hidden rounded-pill bg-surface-sunken">
          <div class="h-full rounded-pill bg-seal transition-all" :style="{ width: progressPct + '%' }" />
        </div>
        <span class="text-xs text-fg-muted">{{ doneCount }}/{{ total }}</span>
      </div>
    </div>

    <div v-if="loading" class="space-y-2">
      <div v-for="i in 3" :key="i" class="h-8 animate-pulse rounded bg-surface-sunken" />
    </div>

    <template v-else>
      <div v-if="tree.length" class="space-y-1">
        <RoadmapNodeItem v-for="root in tree" :key="root.id" :node="root" :project-id="projectId" @refresh="load" />
      </div>
      <p v-else class="rounded-card border border-dashed border-stroke-strong bg-surface-raised px-4 py-8 text-center text-sm text-fg-muted">
        {{ t('roadmap.empty') }}
      </p>

      <form class="mt-4 flex gap-2" @submit.prevent="addRoot">
        <input
          v-model="newRootTitle"
          type="text"
          :placeholder="t('roadmap.rootPlaceholder')"
          class="flex-1 rounded-control border border-stroke-subtle bg-surface-base px-3 py-2 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
        />
        <button
          type="submit"
          :disabled="busy"
          class="inline-flex items-center rounded-control bg-btn px-3 py-2 text-sm font-medium text-btn-fg transition-colors hover:opacity-90"
        >
          <PlusIcon class="-ml-0.5 mr-1 h-4 w-4" />
          {{ t('roadmap.addRoot') }}
        </button>
      </form>
    </template>
  </div>
</template>
