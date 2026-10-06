# 2026-10-06 — Install skill's Ubuntu path has never run on real Ubuntu

## Problem

[`skills/insideout-setup/`](../../skills/insideout-setup/SKILL.md) was
written for macOS and Ubuntu, but only the macOS side has any live
evidence: the scripts it drives were verified on this macOS host
(sandboxed `install.sh` runs, the 2026-10-05 installer plan) and
`env.sh check` ran green here. Every Ubuntu command in
[references/modules.md](../../skills/insideout-setup/references/modules.md)
is check-then-choose by design (e.g. `apt policy golang-go` before
apt-Go, `docker compose version` after install), but **no step has been
executed on a real Ubuntu machine**.

## Specific risk points

- Ubuntu's `docker-compose-v2` package name (24.04) vs Docker's
  official apt repository — the skill gives both, but neither is proven.
- `docker.io` + group relogin flow (`newgrp docker`) untested here.
- Flutter pinned-archive extraction and PATH placement on Ubuntu.
- The skill's platform gate reads `/etc/os-release` — that branch is
  unexecuted.

## Resolution path

Next Ubuntu onboarding (real machine, container, or VM) runs the skill
end-to-end as a fresh-eyes agent would; fix what the machine teaches,
and record the run as a changelog linked from the install-skill plan.
Until then the skill is macOS-proven and Ubuntu-best-effort — say so if
you route a user to it.

## Not this issue

The module content itself was fact-checked at write time (Flutter
3.44.0 archive URLs and go.dev/dl fetched live; apt-version claims
written conditionally rather than asserted).
