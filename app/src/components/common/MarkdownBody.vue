<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ content: string }>()

// renderMarkdown sanitizes with DOMPurify before this reaches v-html.
const html = computed(() => renderMarkdown(props.content))
</script>

<template>
  <!-- Content is DOMPurify-sanitized in renderMarkdown, so v-html is safe here. -->
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="md-body" v-html="html" />
</template>

<style scoped>
/* Chat-bubble prose. Color is inherited from the bubble (ink on light, light on
 * the dark user bubble) so emphasis stays legible in both themes; only
 * structure, spacing and the seal accent are set here. */
.md-body {
  color: inherit;
  font: inherit;
  line-height: inherit;
  overflow-wrap: break-word;
}

.md-body :deep(> :first-child) {
  margin-top: 0;
}
.md-body :deep(> :last-child) {
  margin-bottom: 0;
}

.md-body :deep(p) {
  margin: 0.5em 0;
}

.md-body :deep(strong),
.md-body :deep(b) {
  font-weight: 600;
}
.md-body :deep(em),
.md-body :deep(i) {
  font-style: italic;
}

.md-body :deep(a) {
  color: inherit;
  text-decoration: underline;
  text-decoration-color: rgb(var(--color-seal) / 0.7);
  text-underline-offset: 2px;
}
.md-body :deep(a:hover) {
  text-decoration-color: currentColor;
}

.md-body :deep(ul),
.md-body :deep(ol) {
  margin: 0.5em 0;
  padding-left: 1.25em;
}
.md-body :deep(ul) {
  list-style: disc;
}
.md-body :deep(ol) {
  list-style: decimal;
}
.md-body :deep(li) {
  margin: 0.25em 0;
}
.md-body :deep(li > ul),
.md-body :deep(li > ol) {
  margin: 0.25em 0;
}

/* Inline code reads as a subtle chip — no monospace costume font. */
.md-body :deep(code) {
  border-radius: var(--radius-control);
  background-color: rgb(var(--color-fg-primary) / 0.08);
  padding: 0.1em 0.35em;
  font-size: 0.9em;
}
.md-body :deep(pre) {
  margin: 0.5em 0;
  overflow-x: auto;
  border-radius: var(--radius-control);
  background-color: rgb(var(--color-fg-primary) / 0.06);
  padding: 0.6em 0.8em;
}
.md-body :deep(pre code) {
  background: none;
  padding: 0;
}

.md-body :deep(blockquote) {
  margin: 0.5em 0;
  border-left: 2px solid rgb(var(--color-stroke-strong) / 1);
  padding-left: 0.75em;
  opacity: 0.85;
}

.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3),
.md-body :deep(h4) {
  margin: 0.6em 0 0.3em;
  font-weight: 600;
}
.md-body :deep(h1) {
  font-size: 1.15em;
}
.md-body :deep(h2) {
  font-size: 1.1em;
}
.md-body :deep(h3),
.md-body :deep(h4) {
  font-size: 1.02em;
}

.md-body :deep(hr) {
  margin: 0.75em 0;
  border: none;
  border-top: 1px solid rgb(var(--color-stroke-subtle) / 1);
}

.md-body :deep(table) {
  margin: 0.5em 0;
  border-collapse: collapse;
  font-size: 0.92em;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid rgb(var(--color-stroke-subtle) / 1);
  padding: 0.3em 0.6em;
  text-align: left;
}
</style>
