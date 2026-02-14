from __future__ import annotations

import argparse
import tarfile
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser(description="Build minimal python automation plugin package")
    parser.add_argument("--out", default="dist/python_automation_minimal.tgz", help="output tgz file")
    args = parser.parse_args()

    root = Path(__file__).resolve().parent
    out = (root / args.out).resolve()
    out.parent.mkdir(parents=True, exist_ok=True)

    include = ["manifest.json", "schemas", "bin"]
    with tarfile.open(out, "w:gz") as tf:
        for item in include:
            p = root / item
            tf.add(p, arcname=item)

    print(f"package created: {out}")


if __name__ == "__main__":
    main()

