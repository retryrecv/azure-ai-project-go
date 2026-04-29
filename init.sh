#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
  echo "go not found on PATH" >&2
  exit 1
fi

echo "go version: $(go version)"

# Ensure the module is initialized (created lazily by the first task).
if [ ! -f go.mod ]; then
  echo "go.mod not present yet — first feature task will create it. Skipping build/test."
  exit 0
fi

go mod tidy

# Tolerate the bootstrap state where no .go files exist yet.
if ! find . -name '*.go' -not -path './.*' -print -quit | grep -q .; then
  echo "no .go files yet — skipping build/vet/test"
  echo "smoke test ok"
  exit 0
fi

go build ./...
go vet ./...
go test ./...
echo "smoke test ok"
