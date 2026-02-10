<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const year = computed(() => new Date().getFullYear())
</script>

<template>
  <footer class="mt-auto border-t border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
    <div class="w-full py-6 px-4 sm:px-6 lg:px-8">
      <p class="text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('footer.copyright', { year }) }}
      </p>
    </div>
  </footer>
</template>
