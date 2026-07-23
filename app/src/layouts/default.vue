<script setup lang="ts">
import NavBar from '@/components/layout/NavBar.vue'
import AppFooter from '@/components/layout/AppFooter.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col bg-surface-base text-fg-primary font-sans">
    <NavBar />

    <main class="flex-grow">
      <slot />
    </main>

    <AppFooter />
  </div>
</template>
