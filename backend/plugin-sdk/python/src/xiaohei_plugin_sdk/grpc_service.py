from __future__ import annotations

"""Bridges the Python ``CoreService`` / ``AutomationService`` interfaces to the
gRPC services generated from ``backend/plugin/v1/{core,automation}.proto``.

Each proto RPC method converts:
  proto request  -> Python interface call
  Python result  -> proto response

Plugin exceptions are mapped to ``grpc.StatusCode.INTERNAL`` (with the raw
message suppressed on the wire to avoid leaking internal paths); business
failures are expressed via ``OperationResult{ok=false, error_code,
error_message}`` which the plugin author controls.
"""

import grpc

# Generated stubs (from backend/plugin/v1/*.proto, see generate_protos.py).
from plugin.v1 import automation_pb2 as a_pb2
from plugin.v1 import automation_pb2_grpc as a_grpc
from plugin.v1 import core_pb2 as c_pb2
from plugin.v1 import core_pb2_grpc as c_grpc
from plugin.v1 import manifest_pb2 as m_pb2
from plugin.v1 import types_pb2 as t_pb2

from .models import (
    FEATURE_ENUM_VALUES,
    HEALTH_STATUS_OK,
    HEALTH_STATUS_DEGRADED,
    HEALTH_STATUS_ERROR,
    HEALTH_STATUS_UNSPECIFIED,
)


def _map_error(context, err: Exception) -> None:
    """Abort the RPC with INTERNAL and a safe message.

    The raw exception text is never sent verbatim (it may embed paths/secrets);
    a stable, non-revealing message is returned instead.
    """
    msg = f"plugin internal error: {type(err).__name__}"
    context.abort(grpc.StatusCode.INTERNAL, msg)


def _guard(method):
    """Decorate an RPC handler so plugin exceptions abort the RPC."""

    def wrapper(self, request, context):
        try:
            return method(self, request, context)
        except grpc.RpcError:
            raise
        except Exception as err:  # noqa: BLE001 - boundary guard
            _map_error(context, err)

    return wrapper


def _health_to_proto(status: str) -> int:
    return {
        HEALTH_STATUS_UNSPECIFIED: t_pb2.HEALTH_STATUS_UNSPECIFIED,
        HEALTH_STATUS_OK: t_pb2.HEALTH_STATUS_OK,
        HEALTH_STATUS_DEGRADED: t_pb2.HEALTH_STATUS_DEGRADED,
        HEALTH_STATUS_ERROR: t_pb2.HEALTH_STATUS_ERROR,
    }.get(status, t_pb2.HEALTH_STATUS_OK)


def _feature_to_enum(name: str) -> int:
    return FEATURE_ENUM_VALUES.get(name, m_pb2.AUTOMATION_FEATURE_UNSPECIFIED)


def _manifest_to_proto(manifest) -> m_pb2.Manifest:
    msg = m_pb2.Manifest(
        plugin_id=manifest.plugin_id,
        name=manifest.name,
        version=manifest.version,
        description=manifest.description,
    )
    if getattr(manifest, "sms", None) is not None:
        msg.sms.send = manifest.sms.send
    if getattr(manifest, "payment", None) is not None:
        msg.payment.methods.extend(manifest.payment.methods)
    if getattr(manifest, "kyc", None) is not None:
        msg.kyc.start = manifest.kyc.start
        msg.kyc.query_result = manifest.kyc.query_result
    if getattr(manifest, "automation", None) is not None:
        auto = manifest.automation
        msg.automation.catalog_readonly = bool(getattr(auto, "catalog_readonly", False))
        msg.automation.features.extend(_feature_to_enum(f) for f in (auto.features or []))
        for k, v in (auto.not_supported_reasons or {}).items():
            msg.automation.not_supported_reasons[int(k)] = str(v)
    return msg


def _op_result_to_proto(op) -> a_pb2.OperationResult:
    return a_pb2.OperationResult(
        ok=bool(op.ok),
        error_code=str(getattr(op, "error_code", "") or ""),
        error_message=str(getattr(op, "error_message", "") or ""),
    )


def _items_to_proto(items, proto_item_cls) -> list:
    out = []
    for it in items or []:
        msg = proto_item_cls()
        if isinstance(it, dict):
            for key, value in it.items():
                if hasattr(msg, key) and value is not None:
                    try:
                        setattr(msg, key, value)
                    except (TypeError, ValueError):
                        pass
        else:
            for key in it.__dataclass_fields__ if hasattr(it, "__dataclass_fields__") else ():
                value = getattr(it, key, None)
                if hasattr(msg, key) and value is not None:
                    try:
                        setattr(msg, key, value)
                    except (TypeError, ValueError):
                        pass
        out.append(msg)
    return out


class CoreServicer(c_grpc.CoreServiceServicer):
    def __init__(self, impl):
        self._impl = impl

    @_guard
    def GetManifest(self, request, context):
        return _manifest_to_proto(self._impl.get_manifest())

    @_guard
    def GetConfigSchema(self, request, context):
        schema = self._impl.get_config_schema()
        return t_pb2.ConfigSchema(json_schema=schema.json_schema, ui_schema=schema.ui_schema)

    @_guard
    def ValidateConfig(self, request, context):
        resp = self._impl.validate_config(request.config_json)
        return c_pb2.ValidateConfigResponse(ok=bool(resp.ok), error=str(resp.error))

    @_guard
    def Init(self, request, context):
        resp = self._impl.init(request.instance_id, request.config_json)
        return c_pb2.InitResponse(ok=bool(resp.ok), error=str(resp.error))

    @_guard
    def ReloadConfig(self, request, context):
        resp = self._impl.reload_config(request.config_json)
        return c_pb2.ReloadConfigResponse(ok=bool(resp.ok), error=str(resp.error))

    @_guard
    def Health(self, request, context):
        resp = self._impl.health(request.instance_id)
        return t_pb2.HealthCheckResponse(
            status=_health_to_proto(resp.status),
            message=str(resp.message),
            unix_millis=int(resp.unix_millis),
        )


class AutomationServicer(a_grpc.AutomationServiceServicer):
    def __init__(self, impl):
        self._impl = impl

    # ---- catalog sync ----
    @_guard
    def ListAreas(self, request, context):
        return a_pb2.ListAreasResponse(items=_items_to_proto(self._impl.list_areas(), a_pb2.AutomationArea))

    @_guard
    def ListLines(self, request, context):
        return a_pb2.ListLinesResponse(items=_items_to_proto(self._impl.list_lines(), a_pb2.AutomationLine))

    @_guard
    def ListPackages(self, request, context):
        return a_pb2.ListPackagesResponse(items=_items_to_proto(self._impl.list_packages(request.line_id), a_pb2.AutomationPackage))

    @_guard
    def ListImages(self, request, context):
        return a_pb2.ListImagesResponse(items=_items_to_proto(self._impl.list_images(request.line_id), a_pb2.AutomationImage))

    # ---- catalog write-back ----
    @_guard
    def UpsertPackage(self, request, context):
        op = self._impl.upsert_package(
            request.line_id,
            request.package_name,
            cpu=request.cpu,
            memory_gb=request.memory_gb,
            disk_gb=request.disk_gb,
            bandwidth_mbps=request.bandwidth_mbps,
            traffic_gb=request.traffic_gb,
            gpu_mem_gb=request.gpu_mem_gb,
        )
        return _op_result_to_proto(op)

    @_guard
    def DeletePackage(self, request, context):
        return _op_result_to_proto(self._impl.delete_package(request.line_id, request.package_name))

    # ---- instance lifecycle ----
    @_guard
    def CreateInstance(self, request, context):
        resp = self._impl.create_instance(
            {
                "line_id": request.line_id,
                "package_id": request.package_id,
                "image_id": request.image_id,
                "name": request.name,
                "os": request.os,
                "password": request.password,
                "vnc_password": request.vnc_password,
                "expire_at_unix": request.expire_at_unix,
                "port_num": request.port_num,
                "cpu": request.cpu,
                "memory_gb": request.memory_gb,
                "disk_gb": request.disk_gb,
                "bandwidth_mbps": request.bandwidth_mbps,
            }
        )
        return a_pb2.CreateInstanceResponse(instance_id=int(resp.instance_id))

    @_guard
    def GetInstance(self, request, context):
        data = self._impl.get_instance(request.instance_id) or {}
        return a_pb2.GetInstanceResponse(instance=_dict_to_instance_proto(data))

    @_guard
    def ListInstancesSimple(self, request, context):
        return a_pb2.ListInstancesSimpleResponse(
            items=_items_to_proto(self._impl.list_instances_simple(request.search_tag), a_pb2.AutomationInstanceSimple)
        )

    @_guard
    def Start(self, request, context):
        return _op_result_to_proto(self._impl.start(request.instance_id))

    @_guard
    def Shutdown(self, request, context):
        return _op_result_to_proto(self._impl.shutdown(request.instance_id))

    @_guard
    def Reboot(self, request, context):
        return _op_result_to_proto(self._impl.reboot(request.instance_id))

    @_guard
    def Rebuild(self, request, context):
        return _op_result_to_proto(
            self._impl.rebuild(
                {
                    "instance_id": request.instance_id,
                    "image_id": request.image_id,
                    "password": request.password,
                }
            )
        )

    @_guard
    def ResetPassword(self, request, context):
        return _op_result_to_proto(self._impl.reset_password(request.instance_id, request.password))

    @_guard
    def ElasticUpdate(self, request, context):
        payload = {"instance_id": request.instance_id}
        for key in ("cpu", "memory_gb", "disk_gb", "bandwidth_mbps", "port_num"):
            if request.HasField(key):
                payload[key] = getattr(request, key)
        return _op_result_to_proto(self._impl.elastic_update(payload))

    @_guard
    def Lock(self, request, context):
        return _op_result_to_proto(self._impl.lock(request.instance_id))

    @_guard
    def Unlock(self, request, context):
        return _op_result_to_proto(self._impl.unlock(request.instance_id))

    @_guard
    def Renew(self, request, context):
        return _op_result_to_proto(self._impl.renew(request.instance_id, request.next_due_at_unix))

    @_guard
    def Destroy(self, request, context):
        return _op_result_to_proto(self._impl.destroy(request.instance_id))

    @_guard
    def GetPanelURL(self, request, context):
        resp = self._impl.get_panel_url(request.instance_name, request.panel_password)
        return a_pb2.GetPanelURLResponse(url=resp.url)

    @_guard
    def GetVNCURL(self, request, context):
        resp = self._impl.get_vnc_url(request.instance_id)
        return a_pb2.GetVNCURLResponse(url=resp.url)

    @_guard
    def GetMonitor(self, request, context):
        resp = self._impl.get_monitor(request.instance_id)
        return a_pb2.GetMonitorResponse(raw_json=resp.raw_json)

    # ---- port mapping ----
    @_guard
    def ListPortMappings(self, request, context):
        return a_pb2.ListPortMappingsResponse(
            items=_items_to_proto(self._impl.list_port_mappings(request.instance_id), a_pb2.AutomationPortMapping)
        )

    @_guard
    def AddPortMapping(self, request, context):
        return _op_result_to_proto(
            self._impl.add_port_mapping(
                {
                    "instance_id": request.instance_id,
                    "name": request.name,
                    "sport": request.sport,
                    "dport": request.dport,
                }
            )
        )

    @_guard
    def DeletePortMapping(self, request, context):
        return _op_result_to_proto(self._impl.delete_port_mapping(request.instance_id, request.mapping_id))

    @_guard
    def FindPortCandidates(self, request, context):
        ports = self._impl.find_port_candidates(request.instance_id, request.keywords) or []
        return a_pb2.FindPortCandidatesResponse(ports=[int(p) for p in ports])

    # ---- backup ----
    @_guard
    def ListBackups(self, request, context):
        return a_pb2.ListBackupsResponse(items=_items_to_proto(self._impl.list_backups(request.instance_id), a_pb2.AutomationBackup))

    @_guard
    def CreateBackup(self, request, context):
        return _op_result_to_proto(self._impl.create_backup(request.instance_id))

    @_guard
    def DeleteBackup(self, request, context):
        return _op_result_to_proto(self._impl.delete_backup(request.instance_id, request.backup_id))

    @_guard
    def RestoreBackup(self, request, context):
        return _op_result_to_proto(self._impl.restore_backup(request.instance_id, request.backup_id))

    # ---- snapshot ----
    @_guard
    def ListSnapshots(self, request, context):
        return a_pb2.ListSnapshotsResponse(items=_items_to_proto(self._impl.list_snapshots(request.instance_id), a_pb2.AutomationSnapshot))

    @_guard
    def CreateSnapshot(self, request, context):
        return _op_result_to_proto(self._impl.create_snapshot(request.instance_id))

    @_guard
    def DeleteSnapshot(self, request, context):
        return _op_result_to_proto(self._impl.delete_snapshot(request.instance_id, request.snapshot_id))

    @_guard
    def RestoreSnapshot(self, request, context):
        return _op_result_to_proto(self._impl.restore_snapshot(request.instance_id, request.snapshot_id))

    # ---- firewall ----
    @_guard
    def ListFirewallRules(self, request, context):
        return a_pb2.ListFirewallRulesResponse(
            items=_items_to_proto(self._impl.list_firewall_rules(request.instance_id), a_pb2.AutomationFirewallRule)
        )

    @_guard
    def AddFirewallRule(self, request, context):
        return _op_result_to_proto(
            self._impl.add_firewall_rule(
                {
                    "instance_id": request.instance_id,
                    "direction": request.direction,
                    "protocol": request.protocol,
                    "method": request.method,
                    "port": request.port,
                    "ip": request.ip,
                    "priority": request.priority,
                }
            )
        )

    @_guard
    def DeleteFirewallRule(self, request, context):
        return _op_result_to_proto(self._impl.delete_firewall_rule(request.instance_id, request.rule_id))


def _dict_to_instance_proto(data: dict) -> a_pb2.AutomationInstance:
    msg = a_pb2.AutomationInstance()
    for key in (
        "id",
        "name",
        "state",
        "cpu",
        "memory_gb",
        "disk_gb",
        "bandwidth_mbps",
        "remote_ip",
        "panel_password",
        "vnc_password",
        "os_password",
        "expire_at_unix",
    ):
        if key in data and data[key] is not None:
            try:
                setattr(msg, key, data[key])
            except (TypeError, ValueError):
                pass
    return msg


def register_all(server, servicers: dict) -> None:
    """Register the built-in Core + Automation servicers on a grpc.Server.

    ``servicers`` may contain a subset of {"core", "automation"}; extra keys
    are ignored (kept for future service families).
    """
    core = servicers.get("core")
    if core is not None:
        c_grpc.add_CoreServiceServicer_to_server(core, server)
    auto = servicers.get("automation")
    if auto is not None:
        a_grpc.add_AutomationServiceServicer_to_server(auto, server)
