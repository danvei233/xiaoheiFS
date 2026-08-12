from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

# Canonical automation feature names. These are the exact strings the Go host
# accepts in automationFeatureFromString (backend/internal/adapter/plugins/core/runtime.go)
# and are what plugin authors should put in AutomationCapability.features.
FEATURE_CATALOG_SYNC = "catalog_sync"
FEATURE_LIFECYCLE = "lifecycle"
FEATURE_PORT_MAPPING = "port_mapping"
FEATURE_BACKUP = "backup"
FEATURE_SNAPSHOT = "snapshot"
FEATURE_FIREWALL = "firewall"

# Feature name -> AutomationFeature enum numeric value (plugin/v1/manifest.proto).
FEATURE_ENUM_VALUES = {
    FEATURE_CATALOG_SYNC: 1,
    FEATURE_LIFECYCLE: 2,
    FEATURE_PORT_MAPPING: 3,
    FEATURE_BACKUP: 4,
    FEATURE_SNAPSHOT: 5,
    FEATURE_FIREWALL: 6,
}

# AutomationFeature enum numeric values mirroring plugin/v1/manifest.proto.
AUTOMATION_FEATURE_UNSPECIFIED = 0
AUTOMATION_FEATURE_CATALOG_SYNC = 1
AUTOMATION_FEATURE_LIFECYCLE = 2
AUTOMATION_FEATURE_PORT_MAPPING = 3
AUTOMATION_FEATURE_BACKUP = 4
AUTOMATION_FEATURE_SNAPSHOT = 5
AUTOMATION_FEATURE_FIREWALL = 6

# HealthStatus enum names mirroring plugin/v1/types.proto.
HEALTH_STATUS_UNSPECIFIED = "HEALTH_STATUS_UNSPECIFIED"
HEALTH_STATUS_OK = "HEALTH_STATUS_OK"
HEALTH_STATUS_DEGRADED = "HEALTH_STATUS_DEGRADED"
HEALTH_STATUS_ERROR = "HEALTH_STATUS_ERROR"


@dataclass(slots=True)
class ConfigSchema:
    json_schema: str = "{}"
    ui_schema: str = "{}"


@dataclass(slots=True)
class ValidateConfigResponse:
    ok: bool = True
    error: str = ""


@dataclass(slots=True)
class InitResponse:
    ok: bool = True
    error: str = ""


@dataclass(slots=True)
class ReloadConfigResponse:
    ok: bool = True
    error: str = ""


@dataclass(slots=True)
class HealthCheckResponse:
    status: str = HEALTH_STATUS_OK
    message: str = "ok"
    unix_millis: int = 0


@dataclass(slots=True)
class EmptyResponse:
    """Legacy operation shape matching plugin.v1.Empty (types.proto).

    Retained for reference/back-compat only; automation operation RPCs in
    automation.proto return OperationResult, not Empty.
    """

    status: str = "success"
    msg: str = "ok"
    other: str = ""


@dataclass(slots=True)
class OperationResult:
    """Operation response matching automation.proto OperationResult{ok, error_code, error_message}.

    ``ok`` mirrors the proto3 zero value (False); callers must set ``ok=True``
    to report success, and set error_code/error_message to express a
    structured business failure.
    """

    ok: bool = False
    error_code: str = ""
    error_message: str = ""


@dataclass(slots=True)
class AutomationCapability:
    features: list[str] = field(default_factory=list)
    not_supported_reasons: dict[int, str] = field(default_factory=dict)
    # Catalog data is plugin-managed and should be treated as read-only in admin UI.
    catalog_readonly: bool = False


@dataclass(slots=True)
class SmsCapability:
    send: bool = False


@dataclass(slots=True)
class PaymentCapability:
    methods: list[str] = field(default_factory=list)


@dataclass(slots=True)
class KycCapability:
    start: bool = False
    query_result: bool = False


@dataclass(slots=True)
class Manifest:
    plugin_id: str
    name: str
    version: str
    description: str
    automation: AutomationCapability | None = None
    sms: SmsCapability | None = None
    payment: PaymentCapability | None = None
    kyc: KycCapability | None = None


@dataclass(slots=True)
class CreateInstanceResponse:
    instance_id: int = 0


@dataclass(slots=True)
class GetURLResponse:
    url: str = ""


@dataclass(slots=True)
class RawJSONResponse:
    raw_json: str = "{}"


JSONMap = dict[str, Any]
JSONList = list[JSONMap]

