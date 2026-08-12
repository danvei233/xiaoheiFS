from __future__ import annotations

import unittest

import xiaohei_plugin_sdk as sdk

# Public API that must be importable from the package root.
# Finding r2-m067 F5: __init__.py only exported a subset; half the models
# (InitResponse, AutomationCapability, ...) raised ImportError.
PUBLIC_API = [
    # interfaces
    "AutomationService",
    "CoreService",
    # models
    "ConfigSchema",
    "ValidateConfigResponse",
    "InitResponse",
    "ReloadConfigResponse",
    "HealthCheckResponse",
    "EmptyResponse",
    "OperationResult",
    "AutomationCapability",
    "Manifest",
    "SmsCapability",
    "PaymentCapability",
    "KycCapability",
    "CreateInstanceResponse",
    "GetURLResponse",
    "RawJSONResponse",
    "JSONMap",
    "JSONList",
    # feature name constants
    "FEATURE_CATALOG_SYNC",
    "FEATURE_LIFECYCLE",
    "FEATURE_PORT_MAPPING",
    "FEATURE_BACKUP",
    "FEATURE_SNAPSHOT",
    "FEATURE_FIREWALL",
    "FEATURE_ENUM_VALUES",
    # proto AutomationFeature enum numeric values
    "AUTOMATION_FEATURE_UNSPECIFIED",
    "AUTOMATION_FEATURE_CATALOG_SYNC",
    "AUTOMATION_FEATURE_LIFECYCLE",
    "AUTOMATION_FEATURE_PORT_MAPPING",
    "AUTOMATION_FEATURE_BACKUP",
    "AUTOMATION_FEATURE_SNAPSHOT",
    "AUTOMATION_FEATURE_FIREWALL",
    # health status constants
    "HEALTH_STATUS_UNSPECIFIED",
    "HEALTH_STATUS_OK",
    "HEALTH_STATUS_DEGRADED",
    "HEALTH_STATUS_ERROR",
    # noop
    "NoopAutomationPlugin",
    # server
    "run_local_stub",
]


class PackageExportsTest(unittest.TestCase):
    def test_public_api_importable_from_package_root(self) -> None:
        for name in PUBLIC_API:
            with self.subTest(symbol=name):
                self.assertTrue(
                    hasattr(sdk, name), f"xiaohei_plugin_sdk.{name} is not exported"
                )

    def test_models_importable_via_from_import(self) -> None:
        # These all exist in models.py but were absent from __init__.__all__.
        from xiaohei_plugin_sdk import (
            AutomationCapability,
            CreateInstanceResponse,
            GetURLResponse,
            InitResponse,
            JSONList,
            JSONMap,
            RawJSONResponse,
            ReloadConfigResponse,
            ValidateConfigResponse,
        )

        self.assertIsNotNone(InitResponse)
        self.assertIsNotNone(AutomationCapability)
        self.assertIsNotNone(JSONList)
        self.assertIsNotNone(JSONMap)
        self.assertIsNotNone(RawJSONResponse)
        self.assertIsNotNone(ValidateConfigResponse)
        self.assertIsNotNone(ReloadConfigResponse)
        self.assertIsNotNone(CreateInstanceResponse)
        self.assertIsNotNone(GetURLResponse)

    def test_feature_constants_importable_via_from_import(self) -> None:
        # Finding r2-m067 F2: plugin authors need the feature constants at the
        # package root, not just from the internal models module.
        from xiaohei_plugin_sdk import (
            AUTOMATION_FEATURE_CATALOG_SYNC,
            FEATURE_CATALOG_SYNC,
            FEATURE_ENUM_VALUES,
        )

        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_CATALOG_SYNC], AUTOMATION_FEATURE_CATALOG_SYNC)


if __name__ == "__main__":
    unittest.main()
