"""Regression tests for secret-safe key validation diagnostics."""

import io
import os
import sys
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import _helpers  # noqa: E402


class TestSecretSafeDiagnostics(unittest.TestCase):
    def test_single_line_normalization_accepts_one_terminal_lf_only(self):
        self.assertTrue(_helpers.is_single_line(b"synthetic-secret"))
        self.assertTrue(_helpers.is_single_line(b"synthetic-secret\n"))
        self.assertFalse(_helpers.is_single_line(b"synthetic-secret\nsecond-line"))
        self.assertFalse(_helpers.is_single_line(b"synthetic-secret\n\n"))

    def test_optional_blank_is_distinct_from_malformed_nonblank(self):
        self.assertTrue(_helpers.is_optional_blank(b""))
        self.assertTrue(_helpers.is_optional_blank(b"\n"))
        self.assertFalse(_helpers.is_optional_blank(b"synthetic-secret"))
        self.assertFalse(_helpers.is_optional_blank(b"synthetic-secret\n\n"))

    def test_invalid_single_line_does_not_render_secret(self):
        secret = b"synthetic-secret\nsecond-line"

        class InvalidKey(unittest.TestCase):
            def runTest(self):
                _helpers.assert_single_line(self, secret, "key must be one line")

        output = io.StringIO()
        result = unittest.TextTestRunner(stream=output, verbosity=2).run(InvalidKey())

        self.assertFalse(result.wasSuccessful())
        rendered = output.getvalue()
        self.assertTrue(secret.decode() not in rendered,
                        "nested unittest output leaked synthetic secret")
        self.assertTrue(repr(secret) not in rendered,
                        "nested unittest output leaked synthetic secret repr")


if __name__ == "__main__":
    unittest.main()
