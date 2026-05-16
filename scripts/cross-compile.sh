#!/usr/bin/env bash
set -euo pipefail

cat >&2 <<'EOF'
fsdtrace requires CGO_ENABLED=1 because the Spring indexer uses tree-sitter.
Cross-compilation is not configured in this repository.

Build release artifacts natively on each target OS/architecture, or add an
explicit target C toolchain setup before re-enabling this target.
EOF

exit 1
