"""Print scripts/env_catalog.py's verdict on one scratch repo root, as JSON.

Driven by parity_test.go:  python3 -I -X utf8 pyreader.py <scripts-dir> <root>

-I drops the working directory and PYTHON* variables from sys.path, so the
reader's directory is added by hand. <root> holds the .env.example and .env
under test; the real repo's .env is never reachable from here.
"""

import json
import sys
from pathlib import Path

sys.path.insert(0, sys.argv[1])
import env_catalog as cat  # noqa: E402

root = Path(sys.argv[2])
decls = cat.parse_skeleton(root / ".env.example")
print(json.dumps({
    # One entry per key line in file order: what Go's ParseSkeleton mirrors.
    "skeleton": [{"name": n, "required": r, "example": e, "comment": c}
                 for n, r, e, c in decls],
    # build(): merged by name, joined with the live .env, sorted by attention.
    "vars": [{"name": v.name, "required": v.required, "value": v.value,
              "status": v.status, "secret": v.secret} for v in cat.build(root)],
}))
