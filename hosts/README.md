# Hosts

Per-host metadata and the credential store for machines that run this
repository. One folder per host; inside a host, all credentials live
under a single `keys/` folder: subfolders by functional module, YAML
manifests inside them (split or integrated as the module warrants), and
each module's git-ignored `.secrets/` of `<name>.key` value files
(mode 0600). Manifests note what keys exist and how each is structured —
never one folder per key.

| Path | Host |
| --- | --- |
| [franks-macbook-pro.local/](franks-macbook-pro.local/README.md) | The macOS development workstation |

Do not put passwords, tokens, or raw key files anywhere tracked. Values
live only under each module's `.secrets/` (a dotfolder — `ls -a` to see
it), which `.gitignore` excludes. Manage them with
`python3 scripts/keys.py audit|fill` (never prints values); see
[docs/SETENV.md](../docs/SETENV.md) for how the live `.env` consumer
relates.
