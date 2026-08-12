from __future__ import annotations

"""Plugin serving entrypoints.

``serve(plugin)`` runs the real go-plugin transport (handshake + gRPC on a Unix
socket) so the Go host can load a Python plugin. ``run_local_stub`` is kept for
backward compatibility but now **starts a real server** instead of printing and
exiting — use it for a quick local smoke test.
"""

import argparse
import sys
from typing import Optional

from .interfaces import AutomationService, CoreService
from .transport import HandshakeError, serve as _serve, validate_handshake


def serve(
    plugin,
    *,
    addr: Optional[str] = None,
    block: bool = True,
    allow_dev: bool = False,
    environ: Optional[dict] = None,
):
    """Serve [plugin] over the go-plugin transport (gRPC on a Unix socket).

    Called by the Go host (which sets the magic-cookie env vars) or by an
    example's ``main()``. Blocks until SIGTERM/SIGINT unless ``block=False``.
    """
    return _serve(plugin, addr=addr, block=block, allow_dev=allow_dev, environ=environ, plugins=None)


def run_local_stub(plugin) -> None:
    """Deprecated name kept for compatibility.

    Now actually starts the transport server (dev mode) instead of printing and
    exiting. Prefer :func:`serve`.
    """
    print("[xiaohei-plugin-sdk] starting go-plugin transport (dev, no host cookie)")
    serve(plugin, allow_dev=True)


def _main(argv: Optional[list] = None) -> int:
    parser = argparse.ArgumentParser(prog="xiaohei_plugin_sdk", description="Run a xiaohei Python plugin over go-plugin transport")
    parser.add_argument("--plugin-impl", default=None, help="python:module.path to the plugin instance factory (optional; uses NoopAutomationPlugin)")
    parser.add_argument("--dev", action="store_true", help="allow running without a host magic cookie")
    args = parser.parse_args(argv)

    plugin = _load_plugin(args.plugin_impl)
    try:
        validate_handshake(allow_dev=args.dev)
    except HandshakeError as exc:
        # Executed directly by a human (not by the Go host): go-plugin prints a
        # diagnostic and exits non-zero. We mirror that.
        print(str(exc), file=sys.stderr)
        return 2

    serve(plugin, allow_dev=args.dev)
    return 0


def _load_plugin(spec: Optional[str]):
    if not spec:
        from .noop import NoopAutomationPlugin

        return NoopAutomationPlugin()
    module_name, sep, attr = spec.partition(":")
    if not sep or not attr:
        raise SystemExit("--plugin-impl must be 'module:attribute'")
    import importlib

    module = importlib.import_module(module_name)
    factory = getattr(module, attr)
    if callable(factory) and not isinstance(factory, (CoreService, AutomationService)):
        plugin = factory()
    else:
        plugin = factory
    return plugin


if __name__ == "__main__":
    raise SystemExit(_main())
