<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { LANG_STORAGE_KEY } from '@/i18n'

const { locale, t } = useI18n()

const toggleLanguage = () => {
  const nextLocale = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  locale.value = nextLocale
  // localStorage 仅在客户端可用，SSR 下不可触碰。
  if (import.meta.client) {
    localStorage.setItem(LANG_STORAGE_KEY, nextLocale)
  }
}

const currentCode = computed(() => (locale.value === 'zh-CN' ? 'CN' : 'EN'))
</script>

<template>
  <button
    type="button"
    data-testid="lang-toggle"
    class="inline-flex h-9 min-w-12 items-center justify-center rounded-control border border-stroke-subtle bg-surface-raised px-2 text-xs font-semibold text-fg-secondary transition-colors hover:bg-surface-sunken hover:text-fg-primary"
    :aria-label="t('lang.switchTo')"
    :title="t('lang.switchTo')"
    @click="toggleLanguage"
  >
    {{ currentCode }}
  </button>
</template>
