<script setup lang="ts">
import NavBar from '@/components/layout/NavBar.vue'
import AppFooter from '@/components/layout/AppFooter.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-gray-50 text-gray-900 font-sans dark:bg-gray-900 dark:text-gray-100">
    <NavBar />

    <main class="flex-grow">
      <slot />
    </main>

    <AppFooter />
  </div>
</template>
