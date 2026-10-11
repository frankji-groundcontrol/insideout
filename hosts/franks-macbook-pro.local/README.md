# franks-macbook-pro.local

The macOS workstation that develops and deploys this repository (local dev
against the shared deployment database; deploys via the Railway CLI — see
[usage/deployment.md](../../docs/usage/deployment.md)).

- Metadata: [host.yml](host.yml)
- Keys: [keys/](keys/README.md) — module subfolders (app, auth, compose,
  database, github, llm, mail) with YAML manifests + `.secrets/` values,
  git-ignored
