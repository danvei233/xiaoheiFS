# Python Automation Plugin SDK (Scaffold)

This SDK is a scaffold for writing `automation` plugins in Python.

Current scope:

1. Defines `CoreService` and `AutomationService` full method interfaces.
2. Provides manifest/config/health helper models.
3. Provides a no-op server scaffold for local development.

Important:

1. This scaffold does not implement Hashicorp `go-plugin` transport bridge.
2. It is intended for interface alignment and fast plugin prototyping.

## Layout

1. `xiaohei_plugin_sdk/models.py`: typed data models.
2. `xiaohei_plugin_sdk/interfaces.py`: abstract interfaces.
3. `xiaohei_plugin_sdk/noop.py`: no-op implementation.
4. `xiaohei_plugin_sdk/server.py`: local stub runner.

## Install (editable)

```bash
cd backend/plugin-sdk/python
pip install -e .
```

## Quick Start

```python
from xiaohei_plugin_sdk.noop import NoopAutomationPlugin
from xiaohei_plugin_sdk.server import run_local_stub

plugin = NoopAutomationPlugin()
run_local_stub(plugin)
```

