#!/usr/bin/env python3
"""A real, runnable example plugin that starts and serves over go-plugin gRPC.

Run it (no Go host, dev mode):

    cd backend/plugin-sdk/python
    PYTHONPATH=src python3 -m xiaohei_plugin_sdk --plugin-impl examples.echo_plugin:build_plugin --dev

It binds a Unix socket, prints the go-plugin handshake line to stdout, and
serves ``plugin.v1.CoreService`` + ``plugin.v1.AutomationService`` until
SIGTERM/SIGINT. This is NOT a print-and-exit stub.
"""

from __future__ import annotations

import time

from xiaohei_plugin_sdk.models import (
    FEATURE_CATALOG_SYNC,
    FEATURE_LIFECYCLE,
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
from xiaohei_plugin_sdk.interfaces import AutomationService, CoreService


class EchoPlugin(CoreService, AutomationService):
    """Echo plugin: echoes catalog/lifecycle calls back for local verification.

    Password-bearing fields are never logged and are echoed back only through
    the normal OperationResult response shape (never in exceptions).
    """

    def __init__(self) -> None:
        self._instances: dict[int, dict] = {}

    # ---- core ----
    def get_manifest(self) -> Manifest:
        return Manifest(
            plugin_id="python_echo",
            name="Python Echo Plugin",
            version="0.2.0",
            description="Real go-plugin gRPC transport example (Python)",
            automation=AutomationCapability(features=[FEATURE_CATALOG_SYNC, FEATURE_LIFECYCLE]),
        )

    def get_config_schema(self) -> ConfigSchema:
        return ConfigSchema(
            json_schema='{"type":"object","properties":{"base_url":{"type":"string"}}}',
            ui_schema="{}",
        )

    def validate_config(self, config_json: str) -> ValidateConfigResponse:
        return ValidateConfigResponse(ok=True, error="")

    def init(self, instance_id: str, config_json: str) -> InitResponse:
        self._instances.setdefault(int(instance_id or 0), {})
        return InitResponse(ok=True, error="")

    def reload_config(self, config_json: str) -> ReloadConfigResponse:
        return ReloadConfigResponse(ok=True, error="")

    def health(self, instance_id: str) -> HealthCheckResponse:
        return HealthCheckResponse(status=HEALTH_STATUS_OK, message="ok", unix_millis=int(time.time() * 1000))

    # ---- automation ----
    def list_areas(self) -> list[dict]:
        return [{"id": 1, "name": "area-1", "state": 1}]

    def list_lines(self) -> list[dict]:
        return [{"id": 10, "name": "line-10", "area_id": 1, "state": 1}]

    def list_packages(self, line_id: int) -> list[dict]:
        return [{"id": 100, "name": "pkg-100", "cpu": 2, "memory_gb": 4, "disk_gb": 40, "bandwidth_mbps": 10, "port_num": 1, "monthly_price": 99}]

    def list_images(self, line_id: int) -> list[dict]:
        return [{"id": 7, "name": "debian-12", "type": "linux"}]

    def upsert_package(self, line_id: int, package_name: str, cpu: int = 0, memory_gb: int = 0, disk_gb: int = 0, bandwidth_mbps: int = 0, traffic_gb: int = 0, gpu_mem_gb: int = 0) -> OperationResult:
        return OperationResult(ok=True)

    def delete_package(self, line_id: int, package_name: str) -> OperationResult:
        return OperationResult(ok=True)

    def create_instance(self, payload: dict) -> CreateInstanceResponse:
        iid = int(payload.get("instance_id", 0)) or (len(self._instances) + 1)
        self._instances[iid] = dict(payload)
        return CreateInstanceResponse(instance_id=iid)

    def get_instance(self, instance_id: int) -> dict:
        return self._instances.get(instance_id, {})

    def list_instances_simple(self, search_tag: str) -> list[dict]:
        return [{"id": i, "name": d.get("name", ""), "ip": "203.0.113.10"} for i, d in self._instances.items()]

    def start(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def shutdown(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def reboot(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def rebuild(self, payload: dict) -> OperationResult:
        return OperationResult(ok=True)

    def reset_password(self, instance_id: int, password: str) -> OperationResult:
        return OperationResult(ok=True)

    def elastic_update(self, payload: dict) -> OperationResult:
        return OperationResult(ok=True)

    def lock(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def unlock(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def renew(self, instance_id: int, next_due_at_unix: int) -> OperationResult:
        return OperationResult(ok=True)

    def destroy(self, instance_id: int) -> OperationResult:
        self._instances.pop(instance_id, None)
        return OperationResult(ok=True)

    def get_panel_url(self, instance_name: str, panel_password: str) -> GetURLResponse:
        return GetURLResponse(url="https://panel.example.com/control/ecs/login")

    def get_vnc_url(self, instance_id: int) -> GetURLResponse:
        return GetURLResponse(url="https://panel.example.com/control/ecs/vnc")

    def get_monitor(self, instance_id: int) -> RawJSONResponse:
        return RawJSONResponse(raw_json='{"cpu":1.5,"memory":2048}')

    def list_port_mappings(self, instance_id: int) -> list[dict]:
        return [{"id": 1, "name": "ssh", "sport": "20001", "dport": 22}]

    def add_port_mapping(self, payload: dict) -> OperationResult:
        return OperationResult(ok=True)

    def delete_port_mapping(self, instance_id: int, mapping_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def find_port_candidates(self, instance_id: int, keywords: str) -> list[int]:
        return [20001]

    def list_backups(self, instance_id: int) -> list[dict]:
        return []

    def create_backup(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def delete_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def restore_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def list_snapshots(self, instance_id: int) -> list[dict]:
        return []

    def create_snapshot(self, instance_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def delete_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def restore_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        return OperationResult(ok=True)

    def list_firewall_rules(self, instance_id: int) -> list[dict]:
        return [{"id": 1, "direction": "in", "protocol": "tcp", "method": "allow", "port": "22", "ip": "0.0.0.0/0", "priority": 1}]

    def add_firewall_rule(self, payload: dict) -> OperationResult:
        return OperationResult(ok=True)

    def delete_firewall_rule(self, instance_id: int, rule_id: int) -> OperationResult:
        return OperationResult(ok=True)


def build_plugin() -> EchoPlugin:
    """Factory referenced by ``--plugin-impl examples.echo_plugin:build_plugin``."""
    return EchoPlugin()


if __name__ == "__main__":
    from xiaohei_plugin_sdk.server import serve

    serve(build_plugin(), allow_dev=True)
