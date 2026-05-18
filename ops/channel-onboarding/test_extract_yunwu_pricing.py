#!/usr/bin/env python3
import json
import tempfile
import unittest
from pathlib import Path

import extract_yunwu_pricing as extractor


class YunwuPricingExtractionTest(unittest.TestCase):
    def test_current_scope_excludes_doubao_and_unapproved_groups(self):
        raw = json.loads(
            Path("ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-new.raw.json").read_text(
                encoding="utf-8"
            )
        )

        rows = extractor.coverage_rows(raw)
        families = {row["family"] for row in rows}
        upstream_groups = {row["upstream_group"] for row in rows}
        providers = {row["provider"] for row in rows}

        self.assertNotIn("doubao-video", families)
        self.assertNotIn("Doubao (豆包)", providers)
        self.assertLessEqual(upstream_groups, {"default", "特价kling", "特价vidu"})

    def test_generated_artifacts_record_decisions_without_active_doubao_rows(self):
        raw = json.loads(
            Path("ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-new.raw.json").read_text(
                encoding="utf-8"
            )
        )
        rows = extractor.coverage_rows(raw)

        with tempfile.TemporaryDirectory() as tmp:
            out_dir = Path(tmp)
            extractor.write_outputs(rows, out_dir, "2026-05-18T00:00:00+08:00", "http://127.0.0.1:3001")
            coverage = json.loads((out_dir / "yunwu-coverage.effective.json").read_text(encoding="utf-8"))
            manifest = (out_dir / "yunwu-billing-safe.test.manifest.yaml").read_text(encoding="utf-8")
            manifest_data = extractor.yaml.safe_load(manifest)
            doc = (out_dir / "yunwu-upstream-confirmation.zh-CN.md").read_text(encoding="utf-8")

        self.assertTrue(coverage)
        self.assertFalse(any(row["family"] == "doubao-video" for row in coverage))
        self.assertNotIn("video-upstream-doubao", manifest)
        self.assertTrue(manifest_data["channels"])
        self.assertTrue(all(channel["status"] == 1 for channel in manifest_data["channels"]))
        self.assertIn("Doubao 本轮不处理", doc)
        self.assertIn("只映射 `default`、`特价kling`、`特价vidu`", doc)


if __name__ == "__main__":
    unittest.main()
