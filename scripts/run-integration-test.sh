#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

docker build -f test/docker/Dockerfile -t papabear-test .
docker run --rm papabear-test
