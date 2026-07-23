<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { useUserStore } from '@/stores/user'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import { LinkIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{ projectId: string; repoUrl?: string; ownerId?: string | null }>()
const emit = defineEmits<{ (e: 'synced'): void }>()
const { t } = useI18n()
const userStore = useUserStore()

// The backend allows owner OR workspace admin; the UI gates on ownership (the
// common case) — a non-owner admin can still sync via the API.
const canManage = computed(() => !!props.ownerId && props.ownerId === userStore.user?.id)

const editing = ref(false)
const repoInput = ref('')
const syncing = ref(false)
const saving = ref(false)
const message = ref('')
const error = ref('')

function startEdit() {
  repoInput.value = props.repoUrl ?? ''
  editing.value = true
  message.value = ''
  error.value = ''
}

async function saveRepo() {
  saving.value = true
  error.value = ''
  try {
    await useServices().project.setRepo(props.projectId, repoInput.value.trim())
    editing.value = false
    emit('synced')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    saving.value = false
  }
}

async function sync() {
  syncing.value = true
  message.value = ''
  error.value = ''
  try {
    const res = await useServices().project.syncGithub(props.projectId)
    message.value = t('github.synced', { count: res.added })
    emit('synced')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    syncing.value = false
  }
}
</script>

<template>
  <BaseCard class="mb-8">
    <div class="flex items-center justify-between gap-3">
      <h3 class="flex items-center text-sm font-semibold text-fg-primary">
        <LinkIcon class="mr-2 h-4 w-4 text-seal" />
        {{ t('github.title') }}
      </h3>
      <div v-if="canManage && repoUrl && !editing" class="flex items-center gap-2">
        <BaseButton size="sm" variant="outline" :loading="syncing" @click="sync">
          <ArrowPathIcon class="-ml-1 mr-1.5 h-4 w-4" />
          {{ t('github.sync') }}
        </BaseButton>
        <button type="button" class="text-xs text-fg-muted hover:text-fg-primary" @click="startEdit">{{ t('github.change') }}</button>
      </div>
    </div>

    <div v-if="repoUrl && !editing" class="mt-2">
      <a :href="repoUrl" target="_blank" rel="noopener" class="text-sm text-fg-brand hover:underline">{{ repoUrl }}</a>
      <p v-if="message" class="mt-1 text-xs text-status-success-fg">{{ message }}</p>
    </div>

    <form v-else-if="canManage && editing" class="mt-3 flex gap-2" @submit.prevent="saveRepo">
      <input
        v-model="repoInput"
        type="text"
        :placeholder="t('github.placeholder')"
        class="flex-1 rounded-control border border-stroke-subtle bg-surface-base px-3 py-2 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
      />
      <BaseButton type="submit" size="sm" :loading="saving">{{ t('github.save') }}</BaseButton>
      <button type="button" class="text-xs text-fg-muted hover:text-fg-primary" @click="editing = false">{{ t('github.cancel') }}</button>
    </form>

    <div v-else-if="canManage" class="mt-2">
      <p class="mb-2 text-sm text-fg-muted">{{ t('github.empty') }}</p>
      <BaseButton size="sm" variant="outline" @click="startEdit">{{ t('github.link') }}</BaseButton>
    </div>
    <p v-else class="mt-2 text-sm text-fg-muted">{{ t('github.noRepo') }}</p>

    <p v-if="error" class="mt-2 text-xs text-fg-danger">{{ error }}</p>
  </BaseCard>
</template>
