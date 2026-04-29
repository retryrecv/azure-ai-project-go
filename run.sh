#!/usr/bin/env bash
# Usage: ./run.sh <example>
# Examples: connections | deployments | indexes | datasets
#
# Loads .env automatically, then runs ./examples/<example>.

set -euo pipefail
cd "$(dirname "$0")"

if [ $# -ne 1 ]; then
  echo "usage: $0 <connections|deployments|indexes|datasets>" >&2
  exit 2
fi

if [ ! -d "examples/$1" ]; then
  echo "no example named $1; available:" >&2
  ls examples >&2
  exit 2
fi

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

exec go run "./examples/$1"
