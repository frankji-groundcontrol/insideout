# Nuxt file-based routing: a `pages/x/[id].vue` without `<NuxtPage />` silently swallows `pages/x/[id]/*` children

## What was learned

In Nuxt file-based routing, when **both** `pages/projects/[id].vue` and a
folder `pages/projects/[id]/` (with `index.vue`, `roadmap.vue`, …) exist, the
`[id].vue` file is not a sibling route — it becomes the **parent layout route**
of everything under the `[id]/` folder. A parent route that renders no
`<NuxtPage />` outlet **silently drops its children**: navigating to
`/projects/:id/roadmap` renders only `[id].vue` and never mounts the child.
There is no warning, no 404 — the URL changes, the wrong component renders.

The fix is to make the `[id].vue` file a bare outlet and move its former page
content into `[id]/index.vue`:

```
pages/projects/[id].vue          -> <template><NuxtPage /></template>   (shell)
pages/projects/[id]/index.vue    -> the project-detail page (layout: default)
pages/projects/[id]/roadmap.vue  -> the full-viewport canvas (layout: canvas)
```

Per-child layouts still resolve correctly: in Vue Router's route-record merge
the **child's** `definePageMeta({ layout })` overrides the parent's, so
`index.vue` keeps the default layout while `roadmap.vue` keeps `canvas`.

## Evidence

`/projects/:id/roadmap` was rendering the embedded project page instead of the
full canvas. Two DOM tells confirmed it was a routing shadow, not a canvas or
minimap bug:

- the page `<h1>` was the project title (`Canvas Tree Demo`), not `路线图`; and
- the embedded-only "open full canvas" link (rendered only when
  `RoadmapCanvas` is `embedded`) was present — i.e. the embedded project page
  was what actually mounted on the full route.

After the passthrough restructure: `h1 === 路线图`, the embedded-only link is
gone, and the minimap (full-route-only, `v-if="!embedded"`) renders. Verified
live light + dark on both routes; the index route shows the project page with
the embedded canvas and no minimap, so there is no regression.

## Scope

Applies to any Nuxt (3 or 4) app using file-based routing where a
`foo/[param].vue` file coexists with a `foo/[param]/` folder. The same trap
exists for a `foo.vue` next to a `foo/` folder at any depth — the file is the
parent, the folder's `index.vue` is the default child.

## When to apply

- The moment you add a `pages/x/[id]/something.vue` and an existing
  `pages/x/[id].vue` stops behaving like a page.
- Prefer `[id].vue` as a `<NuxtPage />` shell **from the start** if you expect
  the `[id]/` folder to grow children.
- Leave a comment in the shell explaining it is a routing shell, not a page,
  so a future edit does not re-add page content there and re-break the child
  routes.

Related: [collab T9 changelog](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md).
