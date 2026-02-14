from __future__ import annotations

import json
from dataclasses import asdict, is_dataclass

from .interfaces import AutomationService, CoreService


def _to_jsonable(data):
    if is_dataclass(data):
        return asdict(data)
    return data


def run_local_stub(plugin: CoreService | AutomationService) -> None:
    """
    Local debug runner.
    This does not expose go-plugin transport; it only validates scaffold wiring.
    """
    print("[xiaohei-plugin-sdk] local stub started")
    if isinstance(plugin, CoreService):
        manifest = plugin.get_manifest()
        print("[manifest]", json.dumps(_to_jsonable(manifest), ensure_ascii=False))
    print("[xiaohei-plugin-sdk] ready")

