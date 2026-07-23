<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MoonIcon, SunIcon } from '@heroicons/vue/24/outline'

const THEME_STORAGE_KEY = 'juanleme-theme'
const isDark = ref(false)
const { t } = useI18n()

const applyTheme = (theme: 'dark' | 'light') => {
  isDark.value = theme === 'dark'
  // document 仅在客户端可用，SSR 下不可触碰。
  if (import.meta.client) {
    document.documentElement.classList.toggle('dark', isDark.value)
  }
}

const toggleTheme = () => {
  const nextTheme = isDark.value ? 'light' : 'dark'
  applyTheme(nextTheme)
  if (import.meta.client) {
    localStorage.setItem(THEME_STORAGE_KEY, nextTheme)
  }
}

onMounted(() => {
  // onMounted 仅在客户端运行，这里读取 localStorage 进行首屏主题同步。
  const savedTheme = localStorage.getItem(THEME_STORAGE_KEY)
  const theme = savedTheme === 'dark' ? 'dark' : 'light'
  applyTheme(theme)
})

const label = computed(() => (isDark.value ? t('theme.toggleLight') : t('theme.toggleDark')))
</script>

<template>
  <button
    type="button"
    data-testid="theme-toggle"
    class="inline-flex h-9 w-9 items-center justify-center rounded-control border border-stroke-subtle bg-surface-raised text-fg-secondary transition-colors hover:bg-surface-sunken hover:text-fg-primary"
    :aria-label="label"
    :title="label"
    @click="toggleTheme"
  >
    <MoonIcon v-if="!isDark" class="h-5 w-5" />
    <SunIcon v-else class="h-5 w-5" />
  </button>
</template>
