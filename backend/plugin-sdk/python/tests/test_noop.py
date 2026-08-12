from __future__ import annotations

import unittest

from xiaohei_plugin_sdk.models import FEATURE_ENUM_VALUES, HEALTH_STATUS_OK, OperationResult
from xiaohei_plugin_sdk.noop import NoopAutomationPlugin


class NoopManifestTest(unittest.TestCase):
    """Finding r2-m067 F2: the noop manifest must declare canonical feature names."""

    def setUp(self) -> None:
        self.plugin = NoopAutomationPlugin()

    def test_instantiable_implements_full_surface(self) -> None:
        # Instantiating raises TypeError if any abstract method is unimplemented.
        self.assertIsInstance(self.plugin, NoopAutomationPlugin)

    def test_manifest_features_are_canonical_names(self) -> None:
        features = self.plugin.get_manifest().automation.features
        self.assertTrue(features)
        for name in features:
            with self.subTest(feature=name):
                self.assertIn(
                    name,
                    FEATURE_ENUM_VALUES,
                    f"feature {name!r} must resolve to a proto AutomationFeature value",
                )

    def test_manifest_catalog_readonly_default_false(self) -> None:
        self.assertFalse(self.plugin.get_manifest().automation.catalog_readonly)

    def test_health_uses_ok_status_constant(self) -> None:
        # Finding r2-m067 F8: noop health must use the canonical status constant.
        h = self.plugin.health("instance-1")
        self.assertEqual(h.status, HEALTH_STATUS_OK)
        self.assertGreater(h.unix_millis, 0)


class NoopOperationResultTest(unittest.TestCase):
    """Finding r2-m067 F4: noop operations must report success via OperationResult."""

    def setUp(self) -> None:
        self.plugin = NoopAutomationPlugin()

    def test_operations_return_success_operation_result(self) -> None:
        calls = [
            lambda: self.plugin.start(1),
            lambda: self.plugin.shutdown(1),
            lambda: self.plugin.reboot(1),
            lambda: self.plugin.rebuild({"instance_id": 1}),
            lambda: self.plugin.reset_password(1, "secret"),
            lambda: self.plugin.elastic_update({"instance_id": 1}),
            lambda: self.plugin.lock(1),
            lambda: self.plugin.unlock(1),
            lambda: self.plugin.renew(1, 0),
            lambda: self.plugin.destroy(1),
            lambda: self.plugin.add_port_mapping({"instance_id": 1}),
            lambda: self.plugin.delete_port_mapping(1, 1),
            lambda: self.plugin.create_backup(1),
            lambda: self.plugin.delete_backup(1, 1),
            lambda: self.plugin.restore_backup(1, 1),
            lambda: self.plugin.create_snapshot(1),
            lambda: self.plugin.delete_snapshot(1, 1),
            lambda: self.plugin.restore_snapshot(1, 1),
            lambda: self.plugin.add_firewall_rule({"instance_id": 1}),
            lambda: self.plugin.delete_firewall_rule(1, 1),
            lambda: self.plugin.upsert_package(1, "p1"),
            lambda: self.plugin.delete_package(1, "p1"),
        ]
        for call in calls:
            with self.subTest(method=getattr(call, "__name__", repr(call))):
                result = call()
                self.assertIsInstance(result, OperationResult)
                self.assertTrue(result.ok, "noop operations must report success")


if __name__ == "__main__":
    unittest.main()
