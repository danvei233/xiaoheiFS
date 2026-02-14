from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


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
    status: str = "HEALTH_STATUS_OK"
    message: str = "ok"
    unix_millis: int = 0


@dataclass(slots=True)
class EmptyResponse:
    status: str = "success"
    msg: str = "ok"
    other: str = ""


@dataclass(slots=True)
class AutomationCapability:
    features: list[str] = field(default_factory=list)
    not_supported_reasons: dict[int, str] = field(default_factory=dict)


@dataclass(slots=True)
class Manifest:
    plugin_id: str
    name: str
    version: str
    description: str
    automation: AutomationCapability | None = None


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

