#!/usr/bin/env python3
"""Regenerate the committed gRPC/protobuf stubs from backend/plugin/v1/*.proto.

The generated modules under ``src/plugin/v1/*_pb2*.py`` are checked in so the
SDK works without a build step; this script reproduces them after a proto
change. Requires ``grpcio-tools`` (the ``dev`` extra).

Usage:
    PYTHONPATH=src python3 generate_protos.py
"""

from __future__ import annotations

import pathlib
import subprocess
import sys

REPO = pathlib.Path(__file__).resolve().parent.parent.parent.parent  # backend/plugin-sdk/python -> repo
PROTO_DIR = REPO / "backend" / "plugin" / "v1"
OUT_DIR = pathlib.Path(__file__).resolve().parent / "src"

PROTOS = [
    "types.proto",
    "manifest.proto",
    "core.proto",
    "automation.proto",
    "payment.proto",
    "sms.proto",
    "kyc.proto",
]


def main() -> int:
    try:
        import grpc_tools  # noqa: F401
    except ImportError as exc:  # pragma: no cover
        print("grpcio-tools is required: pip install -e '.[dev]'", file=sys.stderr)
        return 2

    proto_args = [str(PROTO_DIR / p) for p in PROTOS]
    cmd = [
        sys.executable,
        "-m",
        "grpc_tools.protoc",
        "-I",
        str(REPO / "backend"),
        "--python_out",
        str(OUT_DIR),
        "--grpc_python_out",
        str(OUT_DIR),
        *proto_args,
    ]
    subprocess.run(cmd, check=True)
    print(f"generated {len(PROTOS)} proto modules into {OUT_DIR / 'plugin' / 'v1'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
