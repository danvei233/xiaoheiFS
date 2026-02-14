from __future__ import annotations

from xiaohei_plugin_sdk.server import run_local_stub

from .plugin import MinimalAutomationPlugin


def main() -> None:
    run_local_stub(MinimalAutomationPlugin())


if __name__ == "__main__":
    main()

