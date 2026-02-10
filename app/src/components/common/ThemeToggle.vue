<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MoonIcon, SunIcon } from '@heroicons/vue/24/outline'

const THEME_STORAGE_KEY = 'juanleme-theme'
const isDark = ref(false)
const { t } = useI18n()

const applyTheme = (theme: 'dark' | 'light') => {
  isDark.value = theme === 'dark'
  document.documentElement.classList.toggle('dark', isDark.value)
}

const toggleTheme = () => {
  const nextTheme = isDark.value ? 'light' : 'dark'
  applyTheme(nextTheme)
  localStorage.setItem(THEME_STORAGE_KEY, nextTheme)
}

onMounted(() => {
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
    class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-900 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 dark:hover:text-gray-100"
    :aria-label="label"
    :title="label"
    @click="toggleTheme"
  >
    <MoonIcon v-if="!isDark" class="h-5 w-5" />
    <SunIcon v-else class="h-5 w-5" />
  </button>
</template>
