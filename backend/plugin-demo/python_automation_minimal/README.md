# Python Automation Minimal Plugin

This is a minimal no-op automation plugin package based on `xiaohei-plugin-sdk`.

Scope:

1. Full `core + automation` interface shape is present.
2. Business logic is intentionally empty.
3. Manifest/schema/package layout is ready for customization.

## Files

1. `src/python_automation_minimal/plugin.py`: empty implementation class.
2. `manifest.json`: plugin identity and capabilities.
3. `schemas/config.schema.json`: config schema.
4. `schemas/config.ui.json`: UI schema.
5. `bin/*`: placeholder launcher scripts.

## Run Local Stub

```bash
cd backend/plugin-demo/python_automation_minimal
pip install -e .
python -m python_automation_minimal.main
```

## Package Example

```bash
cd backend/plugin-demo/python_automation_minimal
tar -czf python_automation_minimal.tgz manifest.json schemas bin
```

Note:

1. The generated package is a scaffold package.
2. To be fully host-runnable, a go-plugin transport bridge is still required.

