from __future__ import annotations

from abc import ABC, abstractmethod

from .models import (
    ConfigSchema,
    CreateInstanceResponse,
    GetURLResponse,
    HealthCheckResponse,
    InitResponse,
    JSONList,
    Manifest,
    OperationResult,
    RawJSONResponse,
    ReloadConfigResponse,
    ValidateConfigResponse,
)


class CoreService(ABC):
    @abstractmethod
    def get_manifest(self) -> Manifest:
        raise NotImplementedError

    @abstractmethod
    def get_config_schema(self) -> ConfigSchema:
        raise NotImplementedError

    @abstractmethod
    def validate_config(self, config_json: str) -> ValidateConfigResponse:
        raise NotImplementedError

    @abstractmethod
    def init(self, instance_id: str, config_json: str) -> InitResponse:
        raise NotImplementedError

    @abstractmethod
    def reload_config(self, config_json: str) -> ReloadConfigResponse:
        raise NotImplementedError

    @abstractmethod
    def health(self, instance_id: str) -> HealthCheckResponse:
        raise NotImplementedError


class AutomationService(ABC):
    """Automation plugin service (contract aligned with plugin/v1/automation.proto).

    Wire-shape convention:
    - ``JSONList`` methods return a list of dicts. Each dict uses the proto
      message field names in snake_case, as documented per method.
    - ``payload: dict`` methods accept the proto request field names in
      snake_case (documented per method).
    - Operation methods return ``OperationResult``.
    """

    @abstractmethod
    def list_areas(self) -> JSONList:
        """Return areas. Each item: {"id": int, "name": str, "state": int}."""
        raise NotImplementedError

    @abstractmethod
    def list_lines(self) -> JSONList:
        """Return lines. Each item: {"id": int, "name": str, "area_id": int, "state": int}."""
        raise NotImplementedError

    @abstractmethod
    def list_packages(self, line_id: int) -> JSONList:
        """Return packages for line_id. Each item has keys: id, name, cpu, memory_gb,
        disk_gb, bandwidth_mbps, port_num, monthly_price, capacity_remaining."""
        raise NotImplementedError

    @abstractmethod
    def list_images(self, line_id: int) -> JSONList:
        """Return images for line_id. Each item: {"id": int, "name": str, "type": str}."""
        raise NotImplementedError

    @abstractmethod
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
        """Create or update a package on line_id (proto UpsertPackageRequest)."""
        raise NotImplementedError

    @abstractmethod
    def delete_package(self, line_id: int, package_name: str) -> OperationResult:
        """Delete package_name from line_id (proto DeletePackageRequest)."""
        raise NotImplementedError

    @abstractmethod
    def create_instance(self, payload: dict) -> CreateInstanceResponse:
        """Create an instance. payload keys (proto CreateInstanceRequest):
        line_id, package_id, image_id, name, os, password, vnc_password,
        expire_at_unix, port_num, cpu, memory_gb, disk_gb, bandwidth_mbps."""
        raise NotImplementedError

    @abstractmethod
    def get_instance(self, instance_id: int) -> dict:
        """Return the instance dict (proto AutomationInstance): id, name, state, cpu,
        memory_gb, disk_gb, bandwidth_mbps, remote_ip, panel_password, vnc_password,
        os_password, expire_at_unix."""
        raise NotImplementedError

    @abstractmethod
    def list_instances_simple(self, search_tag: str) -> JSONList:
        """Return simple instances. Each item: {"id": int, "name": str, "ip": str}."""
        raise NotImplementedError

    @abstractmethod
    def start(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def shutdown(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def reboot(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def rebuild(self, payload: dict) -> OperationResult:
        """Rebuild an instance. payload keys (proto RebuildRequest):
        instance_id, image_id, password."""
        raise NotImplementedError

    @abstractmethod
    def reset_password(self, instance_id: int, password: str) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def elastic_update(self, payload: dict) -> OperationResult:
        """Elastic update. payload keys (proto ElasticUpdateRequest): instance_id,
        cpu, memory_gb, disk_gb, bandwidth_mbps, port_num (all optional)."""
        raise NotImplementedError

    @abstractmethod
    def lock(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def unlock(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def renew(self, instance_id: int, next_due_at_unix: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def destroy(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def get_panel_url(self, instance_name: str, panel_password: str) -> GetURLResponse:
        raise NotImplementedError

    @abstractmethod
    def get_vnc_url(self, instance_id: int) -> GetURLResponse:
        raise NotImplementedError

    @abstractmethod
    def get_monitor(self, instance_id: int) -> RawJSONResponse:
        raise NotImplementedError

    @abstractmethod
    def list_port_mappings(self, instance_id: int) -> JSONList:
        """Return port mappings. Each item: {"id": int, "name": str, "sport": str, "dport": int}."""
        raise NotImplementedError

    @abstractmethod
    def add_port_mapping(self, payload: dict) -> OperationResult:
        """Add a port mapping. payload keys (proto AddPortMappingRequest):
        instance_id, name, sport, dport."""
        raise NotImplementedError

    @abstractmethod
    def delete_port_mapping(self, instance_id: int, mapping_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def find_port_candidates(self, instance_id: int, keywords: str) -> list[int]:
        raise NotImplementedError

    @abstractmethod
    def list_backups(self, instance_id: int) -> JSONList:
        """Return backups. Each item: {"id": int, "name": str, "created_at_unix": int, "state": int}."""
        raise NotImplementedError

    @abstractmethod
    def create_backup(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def delete_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def restore_backup(self, instance_id: int, backup_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def list_snapshots(self, instance_id: int) -> JSONList:
        """Return snapshots. Each item: {"id": int, "name": str, "created_at_unix": int, "state": int}."""
        raise NotImplementedError

    @abstractmethod
    def create_snapshot(self, instance_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def delete_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def restore_snapshot(self, instance_id: int, snapshot_id: int) -> OperationResult:
        raise NotImplementedError

    @abstractmethod
    def list_firewall_rules(self, instance_id: int) -> JSONList:
        """Return firewall rules. Each item: {"id": int, "direction": str, "protocol": str,
        "method": str, "port": str, "ip": str, "priority": int}."""
        raise NotImplementedError

    @abstractmethod
    def add_firewall_rule(self, payload: dict) -> OperationResult:
        """Add a firewall rule. payload keys (proto AddFirewallRuleRequest):
        instance_id, direction, protocol, method, port, ip, priority."""
        raise NotImplementedError

    @abstractmethod
    def delete_firewall_rule(self, instance_id: int, rule_id: int) -> OperationResult:
        raise NotImplementedError
