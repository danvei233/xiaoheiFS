from __future__ import annotations

import inspect
import unittest

from xiaohei_plugin_sdk.interfaces import AutomationService, CoreService

# Every automation operation RPC returns OperationResult in automation.proto.
OPERATION_METHODS = [
    "start",
    "shutdown",
    "reboot",
    "rebuild",
    "reset_password",
    "elastic_update",
    "lock",
    "unlock",
    "renew",
    "destroy",
    "add_port_mapping",
    "delete_port_mapping",
    "create_backup",
    "delete_backup",
    "restore_backup",
    "create_snapshot",
    "delete_snapshot",
    "restore_snapshot",
    "add_firewall_rule",
    "delete_firewall_rule",
    "upsert_package",
    "delete_package",
]


class AutomationServiceContractTest(unittest.TestCase):
    """Finding r2-m067 F4/F7: operation methods must return OperationResult, not legacy Empty."""

    def test_declares_catalog_writeback_rpcs(self) -> None:
        # Finding r2-m067 F7: automation.proto declares UpsertPackage/DeletePackage.
        self.assertTrue(hasattr(AutomationService, "upsert_package"))
        self.assertTrue(hasattr(AutomationService, "delete_package"))

    def test_operation_methods_return_operation_result(self) -> None:
        for name in OPERATION_METHODS:
            with self.subTest(method=name):
                fn = getattr(AutomationService, name)
                # interfaces.py uses `from __future__ import annotations`, so the
                # annotation is the string "OperationResult".
                self.assertEqual(
                    inspect.signature(fn).return_annotation,
                    "OperationResult",
                    f"{name} must be annotated -> OperationResult",
                )

    def test_services_are_abstract(self) -> None:
        self.assertTrue(inspect.isabstract(CoreService))
        self.assertTrue(inspect.isabstract(AutomationService))


if __name__ == "__main__":
    unittest.main()
