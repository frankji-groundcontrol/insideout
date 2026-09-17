# 2026-09-17 — Vibrant Ink & Seal for younger audiences

Status: **In flight** — OpenDesign exploration; Flutter not restyled yet.

## Why

The user asked to design the frontend in OpenDesign (AMR,
`deepseek-v4.1-flash`), then said:

> Make it vibrant for younger audiences

The live Flutter client is the quiet 国风留白 / Ink & Seal world.
The user then lifted that lock: "No need to stick to ink and seal."
This task explores a new, vibrant visual language for younger workshop
users. Product loop stays the same.

## Design

- Invent a distinctive, high-energy look. Do not reuse celadon / sumi /
  vermilion-seal as the system.
- Bilingual zh-CN + en-US still first-class.
- First artifacts live in the OpenDesign project
  `InsideOut frontend — Ink & Seal` (name is historical). Flutter
  restyle is a later call.

## Out of scope

- Shipping a Flutter theme change in this slice.
- Shipping a replacement for `DESIGN.md` until a direction is accepted.

## Checklist

- [x] OpenDesign project created (AMR + `deepseek-v4.1-flash`)
- [x] Direction brief updated for vibrant / younger audiences
- [x] First screens generated and previewed (VOLT rejected as not vibrant enough)
- [x] Five+ image-backed visual worlds for the user to pick
- [x] User voted 3 — Club Editorial
- [x] Six Club Editorial variants generated (3A–3F, Seedream 5.0 Pro)
- [x] User rejected 3A–3F as unrelated to app functionality
- [x] Club Editorial applied to real product screens (3G–3L)
- [x] User rejected 3G–3L as too old-fashioned
- [x] Youth Club Editorial product screens (3M–3R)
- [ ] User picks a variant or requests another pass
- [ ] Flutter restyle only after acceptance

## Club Editorial variants

3A–3F were fashion-print moodboards (cover, spread, lookbook). Rejected:
they do not show the product loop.

3G–3L are app UI in cream / cobalt / blood orange. They show the
product loop, but they flattened toward generic SaaS chrome — the
magazine energy of vote 3 is weaker here.

Moodboards only — Seedream garbles UI copy. Do not treat on-image
strings as product copy.

| ID | Surface | What it shows |
| --- | --- | --- |
| 3G | Ideas | Inbox capture + idea list, Convert to PRD |
| 3H | PRD + Coach | Working version sections + Coach challenge |
| 3I | Commit | Human Commit dialog, version + decision note |
| 3J | Roadmap | Now / Next / Later bands, deadlines, pressure |
| 3K | Evidence | Git commits on a roadmap node |
| 3L | Dashboard | Projects with version, deadline, evidence |
