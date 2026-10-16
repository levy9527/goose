#!/usr/bin/env bash
# Bad path: the NON-automated way (script/human runs schema first, Go code last).
# Expected to FAIL with "no such column: title" — the failure goose prevents.
set -uo pipefail
cd "$(dirname "$0")/.."   # repo root

echo "================================================================"
echo " Non-automated path: schema changes applied FIRST, Go code LAST."
echo " Expected to FAIL — this is the failure Goose prevents."
echo "================================================================"
set +e
go run ./demo/bad
code=$?
set -e
echo
if [ $code -ne 0 ]; then
  echo "Confirmed: out-of-order execution failed exactly as a naive approach would."
else
  echo "WARNING: insert unexpectedly succeeded — this demo is broken."
fi
exit $code
