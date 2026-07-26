<script setup lang="ts">
import NavBar from '@/components/layout/NavBar.vue'
</script>

<template>
  <!-- Full-viewport canvas layout: NavBar chrome, no footer, and a fixed-height
       main so the roadmap canvas can fill exactly the remaining viewport
       (100vh minus the h-16 navbar) without page scroll. -->
  <div class="flex h-screen flex-col overflow-hidden bg-surface-base text-fg-primary font-sans">
    <NavBar />
    <main class="min-h-0 flex-1">
      <slot />
    </main>
  </div>
</template>
