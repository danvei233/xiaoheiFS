from __future__ import annotations

import time

from .interfaces import AutomationService, CoreService
from .models import (
    FEATURE_BACKUP,
    FEATURE_CATALOG_SYNC,
    FEATURE_FIREWALL,
    FEATURE_LIFECYCLE,
    FEATURE_PORT_MAPPING,
    FEATURE_SNAPSHOT,
    HEALTH_STATUS_OK,
    AutomationCapability,
    ConfigSchema,
    CreateInstanceResponse,
    GetURLResponse,
    HealthCheckResponse,
    InitResponse,
    Manifest,
    OperationResult,
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
                    FEATURE_CATALOG_SYNC,
                    FEATURE_LIFECYCLE,
                    FEATURE_PORT_MAPPING,
                    FEATURE_BACKUP,
                    FEATURE_SNAPSHOT,
                    FEATURE_FIREWALL,
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
        return HealthCheckResponse(status=HEALTH_STATUS_OK, message="ok", unix_millis=int(time.time() * 1000))

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

    def upsert_package(
        self,
        line_id: int,
        package_name: str,
        cpu: int = 0,
        memory_gb: int = 0,
        disk_gb: int = 0,
        bandwidth_mbps: int = 0,
        traffic_gb: int = 0,
        gpu_mem_gb: int = 0,
    ) -> OperationResult:
        _ = line_id
        _ = package_name
        _ = cpu
        _ = memory_gb
        _ = disk_gb
        _ = bandwidth_mbps
        _ = traffic_gb
        _ = gpu_mem_gb
        return OperationResult(ok=True)

    def delete_package(self, line_id: int, package_name: str) -> OperationResult:
        _ = line_id
        _ = package_name
        return OperationResult(ok=True)

    def create_instance(self, payload: dict) -> CreateInstanceResponse:
        _ = payload
        return CreateInstanceResponse(instance_id=0)

    def get_instance(self, instance_id: int) -> dict:
        _ = instance_id
        return {}

    def list_instances_simple(self, search_tag: str) -> list[dict]:
        _ = search_tag
        return []

    def start(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def shutdown(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def reboot(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def rebuild(self, payload: dict) -> OperationResult:
        _ = payload
        return OperationResult(ok=True)

    def reset_password(self, instance_id: int, password: str) -> OperationResult:
        _ = instance_id
        _ = password
        return OperationResult(ok=True)

    def elastic_update(self, payload: dict) -> OperationResult:
        _ = payload
        return OperationResult(ok=True)

    def lock(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def unlock(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def renew(self, instance_id: int, next_due_at_unix: int) -> OperationResult:
        _ = instance_id
        _ = next_due_at_unix
        return OperationResult(ok=True)

    def destroy(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

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

    def add_port_mapping(self, payload: dict) -> OperationResult:
        _ = payload
        return OperationResult(ok=True)

    def delete_port_mapping(self, instance_id: int, mapping_id: int) -> OperationResult:
        _ = instance_id
        _ = mapping_id
        return OperationResult(ok=True)

    def find_port_candidates(self, instance_id: int, keywords: str) -> list[int]:
        _ = instance_id
        _ = keywords
        return []

    def list_backups(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def create_backup(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def delete_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        _ = instance_id
        _ = backup_id
        return OperationResult(ok=True)

    def restore_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        _ = instance_id
        _ = backup_id
        return OperationResult(ok=True)

    def list_snapshots(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def create_snapshot(self, instance_id: int) -> OperationResult:
        _ = instance_id
        return OperationResult(ok=True)

    def delete_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        _ = instance_id
        _ = snapshot_id
        return OperationResult(ok=True)

    def restore_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        _ = instance_id
        _ = snapshot_id
        return OperationResult(ok=True)

    def list_firewall_rules(self, instance_id: int) -> list[dict]:
        _ = instance_id
        return []

    def add_firewall_rule(self, payload: dict) -> OperationResult:
        _ = payload
        return OperationResult(ok=True)

    def delete_firewall_rule(self, instance_id: int, rule_id: int) -> OperationResult:
        _ = instance_id
        _ = rule_id
        return OperationResult(ok=True)

