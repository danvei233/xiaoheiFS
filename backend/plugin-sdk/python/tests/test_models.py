from __future__ import annotations

import json
import unittest
from dataclasses import asdict

from xiaohei_plugin_sdk.models import (
    AUTOMATION_FEATURE_BACKUP,
    AUTOMATION_FEATURE_CATALOG_SYNC,
    AUTOMATION_FEATURE_FIREWALL,
    AUTOMATION_FEATURE_LIFECYCLE,
    AUTOMATION_FEATURE_PORT_MAPPING,
    AUTOMATION_FEATURE_SNAPSHOT,
    AUTOMATION_FEATURE_UNSPECIFIED,
    FEATURE_BACKUP,
    FEATURE_CATALOG_SYNC,
    FEATURE_ENUM_VALUES,
    FEATURE_FIREWALL,
    FEATURE_LIFECYCLE,
    FEATURE_PORT_MAPPING,
    FEATURE_SNAPSHOT,
    HEALTH_STATUS_DEGRADED,
    HEALTH_STATUS_ERROR,
    HEALTH_STATUS_OK,
    HEALTH_STATUS_UNSPECIFIED,
    AutomationCapability,
    ConfigSchema,
    CreateInstanceResponse,
    EmptyResponse,
    GetURLResponse,
    HealthCheckResponse,
    InitResponse,
    KycCapability,
    Manifest,
    OperationResult,
    PaymentCapability,
    RawJSONResponse,
    ReloadConfigResponse,
    SmsCapability,
    ValidateConfigResponse,
)


class FeatureConstantsTest(unittest.TestCase):
    """Finding r2-m067 F2: feature vocabulary must match the host/proto contract."""

    def test_feature_names_are_canonical_host_strings(self) -> None:
        # Host automationFeatureFromString (backend/internal/adapter/plugins/core/runtime.go)
        # accepts exactly these lowercase names.
        self.assertEqual(FEATURE_CATALOG_SYNC, "catalog_sync")
        self.assertEqual(FEATURE_LIFECYCLE, "lifecycle")
        self.assertEqual(FEATURE_PORT_MAPPING, "port_mapping")
        self.assertEqual(FEATURE_BACKUP, "backup")
        self.assertEqual(FEATURE_SNAPSHOT, "snapshot")
        self.assertEqual(FEATURE_FIREWALL, "firewall")

    def test_enum_values_match_proto(self) -> None:
        # plugin/v1/manifest.proto AutomationFeature numeric values.
        self.assertEqual(AUTOMATION_FEATURE_UNSPECIFIED, 0)
        self.assertEqual(AUTOMATION_FEATURE_CATALOG_SYNC, 1)
        self.assertEqual(AUTOMATION_FEATURE_LIFECYCLE, 2)
        self.assertEqual(AUTOMATION_FEATURE_PORT_MAPPING, 3)
        self.assertEqual(AUTOMATION_FEATURE_BACKUP, 4)
        self.assertEqual(AUTOMATION_FEATURE_SNAPSHOT, 5)
        self.assertEqual(AUTOMATION_FEATURE_FIREWALL, 6)

    def test_feature_enum_values_mapping(self) -> None:
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_CATALOG_SYNC], 1)
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_LIFECYCLE], 2)
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_PORT_MAPPING], 3)
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_BACKUP], 4)
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_SNAPSHOT], 5)
        self.assertEqual(FEATURE_ENUM_VALUES[FEATURE_FIREWALL], 6)
        # Every canonical feature string must resolve to a nonzero enum value.
        for value in FEATURE_ENUM_VALUES.values():
            self.assertGreaterEqual(value, 1)


class AutomationCapabilityTest(unittest.TestCase):
    """Finding r2-m067 F2: AutomationCapability is missing proto field catalog_readonly."""

    def test_catalog_readonly_default_false(self) -> None:
        cap = AutomationCapability()
        self.assertFalse(cap.catalog_readonly)

    def test_catalog_readonly_settable(self) -> None:
        cap = AutomationCapability(features=[FEATURE_CATALOG_SYNC], catalog_readonly=True)
        self.assertTrue(cap.catalog_readonly)


class OperationResultTest(unittest.TestCase):
    """Finding r2-m067 F4: operation responses must use proto OperationResult shape."""

    def test_has_proto_fields(self) -> None:
        # automation.proto OperationResult{ok, error_code, error_message};
        # ok mirrors the proto3 zero value (False).
        result = OperationResult()
        self.assertFalse(result.ok)
        self.assertEqual(result.error_code, "")
        self.assertEqual(result.error_message, "")

    def test_expresses_structured_failure(self) -> None:
        result = OperationResult(ok=False, error_code="E_BUSY", error_message="backend busy")
        self.assertFalse(result.ok)
        self.assertEqual(result.error_code, "E_BUSY")
        self.assertEqual(result.error_message, "backend busy")

    def test_expresses_success(self) -> None:
        self.assertTrue(OperationResult(ok=True).ok)


class HealthStatusConstantsTest(unittest.TestCase):
    """Finding r2-m067 F8: health status should use typed constants, not free strings."""

    def test_constants_match_proto_enum_names(self) -> None:
        # plugin/v1/types.proto HealthStatus enum names.
        self.assertEqual(HEALTH_STATUS_UNSPECIFIED, "HEALTH_STATUS_UNSPECIFIED")
        self.assertEqual(HEALTH_STATUS_OK, "HEALTH_STATUS_OK")
        self.assertEqual(HEALTH_STATUS_DEGRADED, "HEALTH_STATUS_DEGRADED")
        self.assertEqual(HEALTH_STATUS_ERROR, "HEALTH_STATUS_ERROR")

    def test_health_response_default_status_uses_constant(self) -> None:
        self.assertEqual(HealthCheckResponse().status, HEALTH_STATUS_OK)


class ManifestCapabilityTest(unittest.TestCase):
    """Finding r2-m067 F9: Manifest must expose optional sms/payment/kyc capabilities."""

    def test_optional_capabilities_default_none(self) -> None:
        m = Manifest(plugin_id="p", name="P", version="1", description="")
        self.assertIsNone(m.sms)
        self.assertIsNone(m.payment)
        self.assertIsNone(m.kyc)

    def test_capability_models_match_proto_fields(self) -> None:
        # manifest.proto: SmsCapability{send}, PaymentCapability{methods}, KycCapability{start, query_result}.
        self.assertFalse(SmsCapability().send)
        self.assertEqual(PaymentCapability().methods, [])
        self.assertFalse(KycCapability().start)
        self.assertFalse(KycCapability().query_result)

    def test_manifest_accepts_all_capabilities(self) -> None:
        m = Manifest(
            plugin_id="p",
            name="P",
            version="1",
            description="",
            sms=SmsCapability(send=True),
            payment=PaymentCapability(methods=["alipay"]),
            kyc=KycCapability(start=True, query_result=True),
        )
        self.assertTrue(m.sms.send)
        self.assertEqual(m.payment.methods, ["alipay"])
        self.assertTrue(m.kyc.start)
        self.assertTrue(m.kyc.query_result)


class ModelRoundTripTest(unittest.TestCase):
    """Finding r2-m067 F11: every response model must survive asdict -> json round-trip."""

    def test_all_response_models_json_roundtrip(self) -> None:
        models = [
            ConfigSchema(),
            ValidateConfigResponse(),
            InitResponse(),
            ReloadConfigResponse(),
            HealthCheckResponse(),
            EmptyResponse(),
            OperationResult(ok=True),
            AutomationCapability(),
            CreateInstanceResponse(),
            GetURLResponse(),
            RawJSONResponse(),
        ]
        for model in models:
            with self.subTest(model=type(model).__name__):
                serialized = json.dumps(asdict(model))
                self.assertIsInstance(serialized, str)
                json.loads(serialized)


if __name__ == "__main__":
    unittest.main()
