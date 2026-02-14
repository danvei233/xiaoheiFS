from __future__ import annotations

from abc import ABC, abstractmethod

from .models import (
    ConfigSchema,
    CreateInstanceResponse,
    EmptyResponse,
    GetURLResponse,
    HealthCheckResponse,
    InitResponse,
    JSONList,
    Manifest,
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
    @abstractmethod
    def list_areas(self) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def list_lines(self) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def list_packages(self, line_id: int) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def list_images(self, line_id: int) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def create_instance(self, payload: dict) -> CreateInstanceResponse:
        raise NotImplementedError

    @abstractmethod
    def get_instance(self, instance_id: int) -> dict:
        raise NotImplementedError

    @abstractmethod
    def list_instances_simple(self, search_tag: str) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def start(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def shutdown(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def reboot(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def rebuild(self, payload: dict) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def reset_password(self, instance_id: int, password: str) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def elastic_update(self, payload: dict) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def lock(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def unlock(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def renew(self, instance_id: int, next_due_at_unix: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def destroy(self, instance_id: int) -> EmptyResponse:
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
        raise NotImplementedError

    @abstractmethod
    def add_port_mapping(self, payload: dict) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def delete_port_mapping(self, instance_id: int, mapping_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def find_port_candidates(self, instance_id: int, keywords: str) -> list[int]:
        raise NotImplementedError

    @abstractmethod
    def list_backups(self, instance_id: int) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def create_backup(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def delete_backup(self, instance_id: int, backup_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def restore_backup(self, instance_id: int, backup_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def list_snapshots(self, instance_id: int) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def create_snapshot(self, instance_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def delete_snapshot(self, instance_id: int, snapshot_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def restore_snapshot(self, instance_id: int, snapshot_id: int) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def list_firewall_rules(self, instance_id: int) -> JSONList:
        raise NotImplementedError

    @abstractmethod
    def add_firewall_rule(self, payload: dict) -> EmptyResponse:
        raise NotImplementedError

    @abstractmethod
    def delete_firewall_rule(self, instance_id: int, rule_id: int) -> EmptyResponse:
        raise NotImplementedError

