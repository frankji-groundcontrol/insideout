<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { useUserStore } from '@/stores/user'
import type { Workspace, WorkspaceMember } from '@/types'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseCard from '@/components/common/BaseCard.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseEmptyState from '@/components/common/BaseEmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { FolderIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'

const route = useRoute()
const workspaceId = route.params.id as string
const { t } = useI18n()
const userStore = useUserStore()

const loading = ref(true)
const notFound = ref(false)
const workspace = ref<Workspace | null>(null)
const members = ref<WorkspaceMember[]>([])

const isAdmin = computed(() => workspace.value?.myRole === 'admin')

const breadcrumb = computed(() => [
  { label: t('nav.dashboard'), to: '/dashboard' },
  { label: workspace.value?.title ?? '…', to: `/workspace/${workspaceId}` },
  { label: t('workspace.settings') },
])

// General form
const titleDraft = ref('')
const descDraft = ref('')
const savingGeneral = ref(false)
const generalMsg = ref('')

// Members
const memberError = ref('')
const roleChanging = ref<string | null>(null)
const removeTarget = ref<WorkspaceMember | null>(null)
const removing = ref(false)

// Danger
const confirmDelete = ref(false)
const deleting = ref(false)

async function load() {
  loading.value = true
  notFound.value = false
  try {
    const ws = useServices().workspace
    const [wsData, memberData] = await Promise.all([ws.get(workspaceId), ws.listMembers(workspaceId)])
    workspace.value = wsData
    members.value = memberData
    titleDraft.value = wsData.title
    descDraft.value = wsData.description
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function saveGeneral() {
  if (!workspace.value || !titleDraft.value.trim()) return
  savingGeneral.value = true
  generalMsg.value = ''
  try {
    const updated = await useServices().workspace.update(workspaceId, {
      title: titleDraft.value.trim(),
      description: descDraft.value,
    })
    workspace.value = updated
    generalMsg.value = t('workspace.saved')
  } finally {
    savingGeneral.value = false
  }
}

async function setRole(member: WorkspaceMember, role: 'admin' | 'member') {
  roleChanging.value = member.userId
  memberError.value = ''
  try {
    await useServices().workspace.updateMemberRole(workspaceId, member.userId, role)
    members.value = members.value.map((m) => (m.userId === member.userId ? { ...m, role } : m))
  } catch {
    memberError.value = t('workspace.removeMember')
  } finally {
    roleChanging.value = null
  }
}

async function confirmRemove() {
  if (!removeTarget.value) return
  removing.value = true
  memberError.value = ''
  try {
    await useServices().workspace.removeMember(workspaceId, removeTarget.value.userId)
    members.value = members.value.filter((m) => m.userId !== removeTarget.value?.userId)
    removeTarget.value = null
  } catch {
    memberError.value = t('workspace.removeMember')
  } finally {
    removing.value = false
  }
}

async function handleDelete() {
  deleting.value = true
  try {
    await useServices().workspace.remove(workspaceId)
    await navigateTo('/dashboard')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="w-full px-4 py-8 sm:px-6 lg:px-8">
    <div v-if="loading" class="space-y-6">
      <div class="h-9 w-64 animate-pulse rounded-card bg-surface-sunken" />
      <div class="h-40 animate-pulse rounded-card bg-surface-sunken" />
    </div>

    <BaseEmptyState v-else-if="notFound" :title="t('workspace.notFound')">
      <template #icon><FolderIcon class="h-6 w-6" /></template>
      <BaseButton to="/dashboard">{{ t('nav.dashboard') }}</BaseButton>
    </BaseEmptyState>

    <template v-else-if="workspace">
      <PageHeader :trail="breadcrumb" :title="t('workspace.settingsTitle')" />

      <div class="max-w-2xl space-y-6">
        <!-- General -->
        <BaseCard>
          <h2 class="mb-4 font-serif text-lg font-semibold text-fg-primary">{{ t('workspace.general') }}</h2>
          <form class="space-y-4" @submit.prevent="saveGeneral">
            <BaseInput v-model="titleDraft" :label="t('workspace.createNamePlaceholder')" :disabled="!isAdmin" required />
            <BaseInput v-model="descDraft" :label="t('workspace.createDescPlaceholder')" :disabled="!isAdmin" />
            <div class="rounded-control border border-stroke-subtle bg-surface-sunken px-3 py-2">
              <p class="text-xs text-fg-muted">{{ t('workspace.inviteHint') }}</p>
              <p class="mt-1 font-medium tracking-widest text-fg-primary">{{ workspace.code }}</p>
            </div>
            <div v-if="isAdmin" class="flex items-center gap-3">
              <BaseButton type="submit" :loading="savingGeneral">{{ t('workspace.saveChanges') }}</BaseButton>
              <span v-if="generalMsg" class="text-sm text-status-success-fg">{{ generalMsg }}</span>
            </div>
          </form>
        </BaseCard>

        <!-- Members -->
        <BaseCard>
          <h2 class="mb-4 font-serif text-lg font-semibold text-fg-primary">{{ t('workspace.membersTitle') }}</h2>
          <p v-if="memberError" class="mb-3 text-sm text-fg-danger">{{ memberError }}</p>
          <ul class="divide-y divide-stroke-subtle">
            <li v-for="m in members" :key="m.userId" class="flex items-center justify-between gap-3 py-3">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-fg-primary">
                  {{ m.username }}
                  <span v-if="m.userId === userStore.user?.id" class="text-fg-muted">· {{ t('workspace.you') }}</span>
                </p>
                <p class="truncate text-xs text-fg-muted">{{ m.email }}</p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <template v-if="isAdmin && m.userId !== userStore.user?.id">
                  <select
                    :value="m.role"
                    :disabled="roleChanging === m.userId"
                    class="rounded-control border border-stroke-subtle bg-surface-sunken px-2 py-1 text-sm text-fg-primary"
                    :aria-label="t('workspace.setRole')"
                    @change="setRole(m, ($event.target as HTMLSelectElement).value as 'admin' | 'member')"
                  >
                    <option value="admin">{{ t('workspace.role.admin') }}</option>
                    <option value="member">{{ t('workspace.role.member') }}</option>
                  </select>
                  <BaseButton size="sm" variant="outline" @click="removeTarget = m">
                    {{ t('workspace.removeMember') }}
                  </BaseButton>
                </template>
                <BaseBadge v-else :tone="m.role === 'admin' ? 'warn' : 'neutral'">
                  {{ t(`workspace.role.${m.role}`) }}
                </BaseBadge>
              </div>
            </li>
          </ul>
        </BaseCard>

        <!-- Danger zone -->
        <BaseCard v-if="isAdmin" class="border-fg-danger/40">
          <div class="flex items-start gap-3">
            <ExclamationTriangleIcon class="mt-0.5 h-5 w-5 shrink-0 text-fg-danger" />
            <div class="min-w-0 flex-1">
              <h2 class="font-serif text-lg font-semibold text-fg-primary">{{ t('workspace.dangerZone') }}</h2>
              <p class="mt-1 text-sm text-fg-muted">{{ t('workspace.deleteWorkspaceHint') }}</p>
              <BaseButton variant="danger" size="sm" class="mt-4" @click="confirmDelete = true">
                {{ t('workspace.deleteWorkspace') }}
              </BaseButton>
            </div>
          </div>
        </BaseCard>
      </div>
    </template>

    <!-- Remove member confirm -->
    <BaseModal :open="removeTarget !== null" :title="t('workspace.removeMember')" size="sm" @close="removeTarget = null">
      <p class="text-sm text-fg-secondary">
        {{ t('workspace.removeMemberConfirm', { name: removeTarget?.username ?? '' }) }}
      </p>
      <template #footer>
        <BaseButton variant="outline" @click="removeTarget = null">{{ t('common.cancel') }}</BaseButton>
        <BaseButton variant="danger" :loading="removing" @click="confirmRemove">{{ t('workspace.removeMember') }}</BaseButton>
      </template>
    </BaseModal>

    <!-- Delete workspace confirm -->
    <BaseModal :open="confirmDelete" :title="t('workspace.deleteWorkspace')" size="sm" @close="confirmDelete = false">
      <p class="text-sm text-fg-secondary">{{ t('workspace.deleteWorkspaceConfirm') }}</p>
      <template #footer>
        <BaseButton variant="outline" @click="confirmDelete = false">{{ t('common.cancel') }}</BaseButton>
        <BaseButton variant="danger" :loading="deleting" @click="handleDelete">{{ t('workspace.deleteWorkspace') }}</BaseButton>
      </template>
    </BaseModal>
  </div>
</template>
