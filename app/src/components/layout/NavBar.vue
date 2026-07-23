<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDownIcon } from '@heroicons/vue/24/outline'
import ThemeToggle from '../common/ThemeToggle.vue'
import LangToggle from '../common/LangToggle.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { t } = useI18n()
const route = useRoute()

const brandTo = computed(() => (userStore.isAuthenticated ? '/dashboard' : '/'))
const initial = computed(() => (userStore.user?.username?.charAt(0) ?? 'U').toUpperCase())

const menuOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)
const closeMenu = () => {
  menuOpen.value = false
}

function onDocClick(e: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) closeMenu()
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') closeMenu()
}

watch(menuOpen, (open) => {
  if (!import.meta.client) return
  if (open) {
    document.addEventListener('click', onDocClick)
    document.addEventListener('keydown', onKeydown)
  } else {
    document.removeEventListener('click', onDocClick)
    document.removeEventListener('keydown', onKeydown)
  }
})
onBeforeUnmount(() => {
  if (!import.meta.client) return
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKeydown)
})

const handleLogout = async () => {
  closeMenu()
  await userStore.logout()
  await navigateTo('/login')
}
</script>

<template>
  <nav class="sticky top-0 z-sticky border-b border-stroke-subtle bg-surface-raised">
    <div class="w-full px-4 sm:px-6 lg:px-8">
      <div class="flex h-16 items-center justify-between">
        <!-- 品牌区 + 主导航 -->
        <div class="flex items-center gap-8">
          <NuxtLink :to="brandTo" class="flex flex-shrink-0 items-center">
            <span class="font-serif text-xl font-bold text-seal">{{ t('nav.brand') }}</span>
          </NuxtLink>
          <NuxtLink
            v-if="userStore.isAuthenticated"
            to="/dashboard"
            class="hidden text-sm font-medium transition-colors sm:block"
            :class="route.path === '/dashboard' ? 'text-fg-primary' : 'text-fg-muted hover:text-fg-primary'"
          >
            {{ t('nav.dashboard') }}
          </NuxtLink>
        </div>

        <!-- 右侧操作 -->
        <div class="flex items-center gap-2">
          <LangToggle />
          <ThemeToggle />

          <template v-if="!userStore.isAuthenticated">
            <NuxtLink to="/login" class="px-2 text-sm text-fg-muted transition-colors hover:text-fg-primary">
              {{ t('nav.login') }}
            </NuxtLink>
            <NuxtLink
              to="/register"
              class="inline-flex items-center justify-center rounded-control bg-btn px-3 py-1.5 text-sm font-medium text-btn-fg transition-colors hover:opacity-90"
            >
              {{ t('nav.startJourney') }}
            </NuxtLink>
          </template>

          <!-- 用户菜单 -->
          <div v-else ref="menuRef" class="relative">
            <button
              type="button"
              class="flex items-center gap-2 rounded-control px-2 py-1 text-fg-secondary transition-colors hover:bg-surface-sunken hover:text-fg-primary"
              :aria-expanded="menuOpen"
              aria-haspopup="menu"
              @click="menuOpen = !menuOpen"
            >
              <img
                v-if="userStore.user?.avatarUrl"
                :src="userStore.user.avatarUrl"
                :alt="userStore.user?.username ?? ''"
                class="h-7 w-7 rounded-full object-cover"
              />
              <span
                v-else
                class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-brand-subtle text-xs font-semibold text-fg-brand"
              >
                {{ initial }}
              </span>
              <span class="hidden text-sm font-medium sm:block">{{ userStore.user?.username ?? '' }}</span>
              <ChevronDownIcon class="h-4 w-4 text-fg-muted transition-transform" :class="{ 'rotate-180': menuOpen }" />
            </button>

            <Transition
              enter-active-class="transition duration-150 ease-out"
              enter-from-class="scale-95 opacity-0"
              enter-to-class="scale-100 opacity-100"
              leave-active-class="transition duration-100 ease-in"
              leave-from-class="scale-100 opacity-100"
              leave-to-class="scale-95 opacity-0"
            >
              <div
                v-if="menuOpen"
                role="menu"
                class="absolute right-0 mt-2 w-44 origin-top-right rounded-card border border-stroke-subtle bg-surface-raised py-1 shadow-popover"
              >
                <NuxtLink
                  to="/profile"
                  role="menuitem"
                  class="block px-4 py-2 text-sm text-fg-secondary transition-colors hover:bg-surface-sunken hover:text-fg-primary"
                  @click="closeMenu"
                >
                  {{ t('nav.profile') }}
                </NuxtLink>
                <button
                  type="button"
                  role="menuitem"
                  class="block w-full px-4 py-2 text-left text-sm text-fg-danger transition-colors hover:bg-surface-sunken"
                  @click="handleLogout"
                >
                  {{ t('nav.logout') }}
                </button>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>
