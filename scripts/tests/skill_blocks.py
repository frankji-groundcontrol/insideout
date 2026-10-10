#!/usr/bin/env python3
"""Print one module's Ubuntu command block from the install skill, verbatim.

Usage: python3 -I scripts/tests/skill_blocks.py <module-number>

The Ubuntu CI job (.github/workflows/ubuntu-install.yml) runs the skill's own
commands instead of a hand-kept copy, so the doc cannot drift from what CI
proves. Exits 2 when the module or its Ubuntu block is missing.
"""
import pathlib
import re
import sys

DOC = pathlib.Path(__file__).resolve().parents[2] / "skills/insideout-setup/references/modules.md"


def ubuntu_block(doc, module):
    section = re.search(rf"^## {re.escape(module)}\. .*?(?=^## \d+\. |\Z)", doc, re.S | re.M)
    if not section:
        return None
    # Prose may sit between the **Ubuntu** marker and the fence; take the first fence after it.
    block = re.search(r"\*\*Ubuntu\*\*.*?```bash\n(.*?)```", section.group(0), re.S)
    return block.group(1) if block else None


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    code = ubuntu_block(DOC.read_text(encoding="utf-8"), sys.argv[1])
    if code is None:
        print(f"no Ubuntu block for module {sys.argv[1]} in {DOC}", file=sys.stderr)
        sys.exit(2)
    sys.stdout.write(code)
