#!/usr/bin/env python3
"""
P0-24 build-context contract test: the Docker build context must never carry
deployment secrets (app.config.yaml, SQLite databases, private keys, .env*).

Runs without a Docker daemon. It reads `.dockerignore`, then simulates Docker's
context-exclusion rules for a representative set of secret files placed at
various depths, and asserts each one is excluded. It also asserts the patterns
do NOT over-exclude files the image actually needs to build.

Docker `.dockerignore` semantics used here:
  - A pattern without `/` matches the basename at any depth (`*.db` -> any
    `foo.db` anywhere).
  - A pattern with `/` matches the (relative) path; `**` spans directories.
  - `fnmatch` in Python treats `*` as spanning `/`, which matches Docker's
    per-component `*` for the simple globs used in this file.

Exit code is non-zero on any contract violation.
"""
import fnmatch
import pathlib
import sys

REPO = pathlib.Path(__file__).resolve().parent.parent
DOCKERIGNORE = REPO / ".dockerignore"

# Secret files that must be excluded from any build context (name -> depth).
SECRET_FILES = [
    "app.config.yaml",
    "config/app.config.yaml",
    "nested/deep/config.yaml",
    "data/xiaohei.db",
    "data/xiaohei.db-wal",
    "data/xiaohei.db-shm",
    "backups/orders.sqlite",
    "nested/deep/state.sqlite3",
    "nested/deep/state.sqlite-wal",
    "secrets/deploy.key",
    "certs/server.pem",
    "certs/server.key",
    "android/upload-keystore.jks",
    "android/upload-key.p12",
    "secrets/oauth-client.pfx",
    "certs/selfsigned.crt",
    "config/operator.secret",
    "config/agent.token",
    ".env",
    ".env.production",
    ".env.local",
]

# Files the image build genuinely needs; these must NOT be excluded.
REQUIRED_FILES = [
    "docker/Dockerfile",
    "docker/Dockerfile.alpine",
    "docker/entrypoint.sh",
    "docker/sync-plugins.sh",
    "backend/go.mod",
    "backend/go.sum",
    "backend/plugins/payment/demo/manifest.json",
    "backend/plugins/payment/demo/bin/linux_amd64/plugin",
    "frontend/package.json",
    ".env.example",  # explicitly re-included via !.env.example
]

failures = []


def check(cond, msg):
    if cond:
        print(f"  ok   {msg}")
    else:
        print(f"  FAIL {msg}")
        failures.append(msg)


def load_patterns():
    patterns = []
    for raw in DOCKERIGNORE.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        # Trailing `/` excludes a directory; not used here but normalize anyway.
        patterns.append(line.rstrip("/"))
    return patterns


def _match_single(pattern, relpath):
    """True if one .dockerignore pattern matches the (relative) path."""
    if pattern.startswith("**/"):
        # **/x also matches x at the root.
        return fnmatch.fnmatch(relpath, pattern) or fnmatch.fnmatch(relpath, pattern[3:])
    if "/" not in pattern:
        # No slash -> matches the basename at any depth.
        return fnmatch.fnmatch(relpath, pattern) or fnmatch.fnmatch(relpath.split("/")[-1], pattern)
    return fnmatch.fnmatch(relpath, pattern)


def is_excluded(patterns, relpath):
    """Docker semantics: iterate in order; a later `!pattern` re-includes."""
    relpath = relpath.lstrip("/")  # keep a leading dot (e.g. .env)
    excluded = False
    for p in patterns:
        if p.startswith("!"):
            if _match_single(p[1:], relpath):
                excluded = False
        elif _match_single(p, relpath):
            excluded = True
    return excluded


def main():
    print("== .dockerignore build-context contract ==")
    patterns = load_patterns()
    check(len(patterns) > 10, ".dockerignore defines a substantive rule set")

    for needle in (
        "app.config.yaml",
        "*.db",
        "*.sqlite",
        "*.sqlite3",
        "*.key",
        "*.pem",
        "*.jks",
        "*.p12",
        "*.pfx",
        "*.keystore",
        "*.crt",
        ".env",
        ".env.*",
    ):
        check(any(needle in p for p in patterns), f"rule present for {needle}")

    print("[exclusion] every secret file at every depth is excluded")
    for secret in SECRET_FILES:
        check(
            is_excluded(patterns, secret),
            f"{secret} is excluded from the build context",
        )

    print("[inclusion] image-required files are not over-excluded")
    for req in REQUIRED_FILES:
        check(
            not is_excluded(patterns, req),
            f"{req} remains available to the build",
        )

    print()
    if failures:
        print(f"{len(failures)} contract violation(s)")
        sys.exit(1)
    print("all .dockerignore build-context contracts satisfied")


if __name__ == "__main__":
    main()
