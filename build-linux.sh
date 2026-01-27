#!/usr/bin/env bash
set -euo pipefail

# Wrapper for linux build artifacts: outputs to ./build/linux/
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/build/linux/build-linux.sh" "$@"
