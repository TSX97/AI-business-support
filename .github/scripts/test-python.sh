#!/usr/bin/env bash
set -euo pipefail

echo "Checking Python syntax for all backend services."
python -m compileall -q backend
echo "Python syntax checks passed."
