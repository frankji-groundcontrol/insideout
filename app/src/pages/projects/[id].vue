<!--
  Routing shell, NOT a page. Nuxt treats `projects/[id].vue` as the PARENT of
  everything under `projects/[id]/` (`index.vue` = the project detail page,
  `roadmap.vue` = the full-viewport canvas). A parent without `<NuxtPage />`
  silently swallows its children — that is exactly how the full roadmap route
  once degraded to re-rendering the embedded project page. Keep this file to a
  bare outlet; the real pages live in the `[id]/` folder, each with its own
  layout (index = default, roadmap = canvas).
-->
<template>
  <NuxtPage />
</template>
