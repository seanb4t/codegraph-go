"""Unit suite for scripts/third_party_notices.py.

Every case passes literal report rows and license files in a temporary
directory. Nothing here runs go-licenses, go or Vite; what those print is not
ours to test, only what this script makes of it.

`task check:ci-scripts` runs this file and asserts that unittest's executed count
equals the number of `def test_` methods defined here (rule 84d1gfpywd).
"""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

import third_party_notices as tpn

WEB = "Name: elkjs\nVersion: 0.9.3\nLicense: EPL-2.0\n\nEclipse text\n"


class NoticesTest(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = Path(tmp.name)

    def _license(self, name, text):
        path = self.dir / name
        path.write_text(text, encoding="utf-8")
        return str(path)

    def _row(self, name, version, license_name, path):
        return f"{name}\t{version}\t{license_name}\t{path}\n"

    def test_libraries_with_one_text_share_one_block(self):
        apache = self._license("a", "Apache text\n")
        mit = self._license("m", "MIT text\n")
        rows = tpn.parse_rows(
            [
                self._row("example.com/b", "v2.0.0", "Apache-2.0", apache),
                self._row("example.com/a", "v1.0.0", "Apache-2.0", apache),
                self._row("example.com/c", "v3.0.0", "MIT", mit),
            ]
        )
        out = tpn.render(rows, WEB)
        self.assertEqual(out.count("Apache text"), 1)
        self.assertIn("example.com/a v1.0.0 (Apache-2.0)\nexample.com/b v2.0.0 (Apache-2.0)\n\nApache text", out)
        self.assertIn("example.com/c v3.0.0 (MIT)\n\nMIT text", out)
        self.assertIn("Name: elkjs", out)

    def test_main_module_is_left_out(self):
        mit = self._license("m", "MIT text\n")
        rows = tpn.parse_rows(
            [
                self._row(tpn.MAIN_MODULE, "Unknown", "MIT", mit),
                self._row(tpn.MAIN_MODULE + "/internal/x", "Unknown", "MIT", mit),
                self._row("example.com/c", "v3.0.0", "MIT", mit),
            ]
        )
        self.assertEqual(list(rows), [("example.com/c", "v3.0.0")])

    def test_same_library_from_two_platforms_is_listed_once(self):
        mit = self._license("m", "MIT text\n")
        row = self._row("example.com/c", "v3.0.0", "MIT", mit)
        out = tpn.render(tpn.parse_rows([row, row]), WEB)
        self.assertEqual(out.count("example.com/c v3.0.0"), 1)

    def test_unknown_license_outside_the_table_fails_and_names_the_library(self):
        rows = tpn.parse_rows([self._row("example.com/mystery", "v1.0.0", "Unknown", "Unknown")])
        with self.assertRaises(tpn.NoticesError) as ctx:
            tpn.render(rows, WEB)
        self.assertIn("example.com/mystery", str(ctx.exception))

    def test_short_form_module_uses_its_license_file(self):
        module = next(iter(tpn.SHORT_FORM))
        (self.dir / "LICENSE").write_text("Licensed under the Apache License\n", encoding="utf-8")
        rows = tpn.parse_rows([self._row(module + "/pkg", "v0.1.0", "Unknown", "Unknown")])
        out = tpn.render(rows, WEB, find_module_dir=lambda m: self.dir)
        self.assertIn(f"{module}/pkg v0.1.0 (Apache-2.0)\n\nLicensed under the Apache License", out)

    def test_empty_report_fails(self):
        with self.assertRaises(tpn.NoticesError):
            tpn.render({}, WEB)

    def test_web_notices_without_a_package_fail(self):
        mit = self._license("m", "MIT text\n")
        rows = tpn.parse_rows([self._row("example.com/c", "v3.0.0", "MIT", mit)])
        with self.assertRaises(tpn.NoticesError):
            tpn.render(rows, "\n")

    def test_row_with_wrong_field_count_fails(self):
        with self.assertRaises(tpn.NoticesError):
            tpn.parse_rows(["example.com/c\tv3.0.0\tMIT\n"])


if __name__ == "__main__":
    unittest.main()
