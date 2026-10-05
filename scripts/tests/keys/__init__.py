"""Package hook so unittest discovery honors key-name filenames.

The per-key test files are named exactly like their keys
(test_jwt-secret.py, test_gh-app-id.py, …) — hyphens included, per the
store convention. Hyphens are not valid Python identifiers, so plain
discovery cannot import those files. This hook is therefore the single
collector: it loads every test_*.py in this package (root and module
folders) by file path, identifier-named or not, and returns the
combined suite.
"""

import glob
import importlib.util
import os
import sys


def load_tests(loader, tests, pattern):
    here = os.path.dirname(__file__)
    for path in sorted(
        glob.glob(os.path.join(here, "test_*.py"))
        + glob.glob(os.path.join(here, "*", "test_*.py"))
    ):
        stem = os.path.splitext(os.path.basename(path))[0]
        if stem == "__init__" or stem.startswith("_"):
            continue
        module_dir = os.path.basename(os.path.dirname(path))
        modname = f"keys_tests_{module_dir}_{stem.replace('-', '_')}"
        if modname not in sys.modules:
            spec = importlib.util.spec_from_file_location(modname, path)
            mod = importlib.util.module_from_spec(spec)
            sys.modules[modname] = mod
            spec.loader.exec_module(mod)
        tests.addTests(loader.loadTestsFromModule(sys.modules[modname]))
    return tests
