<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import NavBar from '@/components/layout/NavBar.vue'
import AppFooter from '@/components/layout/AppFooter.vue'

const route = useRoute()
const isLayoutEmpty = computed(() => route.meta.layout === 'empty')
</script>

<template>
  <div class="min-h-screen flex flex-col bg-gray-50 text-gray-900 font-sans">
    <NavBar v-if="!isLayoutEmpty" />

    <main class="flex-grow">
      <RouterView />
    </main>

    <AppFooter v-if="!isLayoutEmpty" />
  </div>
</template>
