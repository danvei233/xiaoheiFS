from __future__ import annotations

import time

from .interfaces import AutomationService, CoreService
from .models import (
    AutomationCapability,
    ConfigSchema,
    CreateInstanceResponse,
    EmptyResponse,
    GetURLResponse,
    HealthCheckResponse,
    InitResponse,
    Manifest,
    RawJSONResponse,
    ReloadConfigResponse,
    ValidateConfigResponse,
)


class NoopAutomationPlugin(CoreService, AutomationService):
    def get_manifest(self) -> Manifest:
        return Manifest(
            plugin_id="python_noop",
            name="Python Noop Automation Plugin",
            version="0.1.0",
            description="No-op python automation plugin scaffold",
            automation=AutomationCapability(
                features=[
                    "catalog_sync",
                    "lifecycle",
                    "port_mapping",
                    "backup",
                    "snapshot",
                    "firewall",
                ]
            ),
        )

    def get_config_schema(self) -> ConfigSchema:
        return ConfigSchema(
            json_schema='{"type":"object","properties":{"base_url":{"type":"string"},"api_key":{"type":"string","format":"password"}}}',
            ui_schema='{"api_key":{"ui:widget":"password"}}',
        )

    def validate_config(self, config_json: str) -> ValidateConfigResponse:
        _ = config_json
        return ValidateConfigResponse(ok=True, error="")

    def init(self, instance_id: str, config_json: str) -> InitResponse:
        _ = instance_id
        _ = config_json
        return InitResponse(ok=True, error="")

    def reload_config(self, config_json: str) -> ReloadConfigResponse:
        _ = config_json
        return ReloadConfigResponse(ok=True, error="")

    def health(self, instance_id: str) -> HealthCheckResponse:
        _ = instance_id
        return HealthCheckResponse(status="HEALTH_STATUS_OK", message="ok", unix_millis=int(time.time() * 1000))

    def list_areas(self) -> list[dict]:
        return []

    def list_lines(self) -> list[dict]:
        return []

    def list_packages(self, line_id: int) -> list[dict]:
        _ = line_id
        return []

    def list_images(self, line_id: int) -> list[dict]:
        _ = line_id
        return []

    def create_instance(self, payload: dict) -> CreateInstanceResponse:
        _ = payload
        return CreateInstanceResponse(instance_id=0)

    def get_instance(self, instance_id: int) -> dict:
        _ = instance_id
        return {}

    def list_instances_simple(self, search_tag: str) -> list[dict]:
        _ = search_tag
        return []

    def start(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def shutdown(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def reboot(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def rebuild(self, payload: dict) -> EmptyResponse:
        _ = payload
        return EmptyResponse()

    def reset_password(self, instance_id: int, password: str) -> EmptyResponse:
        _ = instance_id
        _ = password
        return EmptyResponse()

    def elastic_update(self, payload: dict) -> EmptyResponse:
        _ = payload
        return EmptyResponse()

    def lock(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def unlock(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def renew(self, instance_id: int, next_due_at_unix: int) -> EmptyResponse:
        _ = instance_id
        _ = next_due_at_unix
        return EmptyResponse()

    def destroy(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def get_panel_url(self, instance_name: str, panel_password: str) -> GetURLResponse:
        _ = instance_name
        _ = panel_password
        return GetURLResponse(url="")

    def get_vnc_url(self, instance_id: int) -> GetURLResponse:
        _ = instance_id
        return GetURLResponse(url="")

    def get_monitor(self, instance_id: int) -> RawJSONResponse:
        _ = instance_id
        return RawJSONResponse(raw_json="{}")

    def list_port_mappings(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def add_port_mapping(self, payload: dict) -> EmptyResponse:
        _ = payload
        return EmptyResponse()

    def delete_port_mapping(self, instance_id: int, mapping_id: int) -> EmptyResponse:
        _ = instance_id
        _ = mapping_id
        return EmptyResponse()

    def find_port_candidates(self, instance_id: int, keywords: str) -> list[int]:
        _ = instance_id
        _ = keywords
        return []

    def list_backups(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def create_backup(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def delete_backup(self, instance_id: int, backup_id: int) -> EmptyResponse:
        _ = instance_id
        _ = backup_id
        return EmptyResponse()

    def restore_backup(self, instance_id: int, backup_id: int) -> EmptyResponse:
        _ = instance_id
        _ = backup_id
        return EmptyResponse()

    def list_snapshots(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def create_snapshot(self, instance_id: int) -> EmptyResponse:
        _ = instance_id
        return EmptyResponse()

    def delete_snapshot(self, instance_id: int, snapshot_id: int) -> EmptyResponse:
        _ = instance_id
        _ = snapshot_id
        return EmptyResponse()

    def restore_snapshot(self, instance_id: int, snapshot_id: int) -> EmptyResponse:
        _ = instance_id
        _ = snapshot_id
        return EmptyResponse()

    def list_firewall_rules(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def add_firewall_rule(self, payload: dict) -> EmptyResponse:
        _ = payload
        return EmptyResponse()

    def delete_firewall_rule(self, instance_id: int, rule_id: int) -> EmptyResponse:
        _ = instance_id
        _ = rule_id
        return EmptyResponse()

