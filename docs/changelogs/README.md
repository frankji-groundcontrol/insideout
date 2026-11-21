# Changelogs

Dated records of what changed in this repository, written for maintainers.

## Convention

- **Ordinary change**: one dated file, `YYYY-MM-DD-short-slug.md`.
- **Large change** (multi-phase, multiple subsystems): a dated folder,
  `YYYY-MM-DD-short-slug/`, with an `index.md` linking focused child files
  (e.g. `summary.md`, `verification.md`, `migration-notes.md`).

Each record states what changed, how it was verified, and what an operator of
an existing deployment needs to do. Cite sources (plans, bug book entries,
code) with relative links.

## Records

Per-month index files ([folder README](index/README.md)), newest first:

- [November 2026](index/2026-11.md)
- [October 2026](index/2026-10.md)
- [September 2026](index/2026-09.md)
- [August 2026](index/2026-08.md)
- [July 2026](index/2026-07.md)

The month files are the only index: every record file and dated folder
directly in this folder is listed, with a one-line description, in the month
file for its date. A new record adds its entry there; a new month adds a
month file and a line here and in the [month index README](index/README.md).
