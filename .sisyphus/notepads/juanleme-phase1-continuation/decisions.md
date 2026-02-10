# Architectural Decisions

## 2026-02-10 Task: Planning Phase
- Service Layer: Interface contracts + mock adapters + service registry (VITE_API_MODE=mock|supabase)
- Tiptap Editor: JSON as primary storage format, 800ms debounce autosave to localStorage
- AI-Editor Communication: editorStore.enqueueInsert(text) command pattern — no direct coupling
- Export: Markdown file download + Print CSS + window.print() (no html2canvas/jspdf)
- Mobile Layout: Tab-based switching below md breakpoint with KeepAlive for Tiptap
- Testing: Vitest + @vue/test-utils + happy-dom, TDD RED-GREEN-REFACTOR
- Dark mode: Tailwind `darkMode: 'class'` strategy
- i18n: vue-i18n@9 with EN/CN locales, default CN
- Draft key format: `draft:${userId}:${workshopId}:${nodeId}:v1`
- One-way Tiptap→Pinia sync only (Tiptap is runtime source-of-truth)
