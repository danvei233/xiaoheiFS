from __future__ import annotations

"""``python -m xiaohei_plugin_sdk`` — run a Python plugin over the go-plugin
transport (handshake + gRPC). With no ``--plugin-impl`` it serves the bundled
NoopAutomationPlugin; with ``--dev`` it runs without a host magic cookie.
"""

import sys

from .server import _main

if __name__ == "__main__":
    sys.exit(_main())
