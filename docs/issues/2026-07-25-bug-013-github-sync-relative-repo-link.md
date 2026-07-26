# BUG-013 — GitHub-sync repo link is a relative URL

Date: 2026-07-25. Status: open (deferred). Severity: low (cosmetic, but breaks
the one link the card shows). Component: `app/src/components/project/GithubSync.vue`.

## Symptom

After linking a repo as `owner/repo` (the form placeholder literally invites
"owner/repo or a github.com URL") and reloading, the GitHub-sync card renders
the repo as a link whose `href` is the bare `owner/repo` string. Clicking it
navigates to `<app-origin>/owner/repo` — a 404 inside the app — instead of the
repository on github.com.

Observed live during the self-referential sync demo: the card showed the linked
repo, but its anchor `href` was the relative `owner/repo`, not
`https://github.com/<owner>/<repo>`.

## Root cause

`GithubSync.vue` line ~80 binds the stored value straight through:

```html
<a :href="repoUrl" target="_blank" rel="noopener" ...>{{ repoUrl }}</a>
```

`PUT /projects/{id}/repo` stores `repoUrl` exactly as submitted (a pure DB
UPDATE with no normalization), so an `owner/repo` input is persisted without a
scheme and rendered as a relative link.

## Fix (suggested)

Normalize for display only — leave storage as-is. Compute an href: if `repoUrl`
already starts with `http`, use it; otherwise prefix `https://github.com/`.
A small `computed` in the component keeps it on the frontend and avoids a
migration. (Normalizing in the backend instead would also work but changes what
is stored and needs the same display path anyway.)

## Notes

- The sync itself is unaffected — `POST /sync-github` reads the stored value
  and fetches commits fine; only the outbound anchor is wrong.
- Filed from the live demo that linked this repository to its own running
  instance; the repo identifier is redacted here per the privacy rule.
