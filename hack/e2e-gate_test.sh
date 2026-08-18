#!/usr/bin/env bash
# Regression cover for e2e-gate.sh. The two cases that matter most are the ones
# that would silently pass a PR whose suite never ran: a broken authorize and a
# broken changes job both skip e2e, and reading that skip as "nothing to do"
# would defeat the point of making this a required check.
set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT="${HERE}/e2e-gate.sh"

pass=0
fail=0
expect() {
  local label=$1 want=$2 authorize=$3 changes=$4 result=$5
  local out rc=0
  out=$(AUTHORIZE="${authorize}" CHANGES="${changes}" RESULT="${result}" \
    NEEDED=true "${SCRIPT}" 2>&1) || rc=$?
  local got=pass
  [ "${rc}" -ne 0 ] && got=fail
  if [ "${got}" = "${want}" ]; then
    echo "  ok: ${label} -> ${got}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${label} -> ${got}, want ${want} (rc=${rc}, out: ${out})"
    fail=$((fail + 1))
  fi
}

echo "cases that must let the PR merge"
expect "go PR, suite green"            pass success success success
expect "docs PR, suite skipped"       pass success success skipped
expect "fork PR, authorize skipped"   pass skipped skipped skipped

echo "cases that must block the PR"
expect "suite failed"                 fail success success failure
expect "suite cancelled"              fail success success cancelled
expect "authorize failed"             fail failure skipped skipped
expect "authorize cancelled"          fail cancelled skipped skipped
expect "changes failed"               fail success failure skipped
expect "changes cancelled"            fail success cancelled skipped
expect "unknown result string"        fail success success weird

echo "missing inputs must fail loudly rather than default to a pass"
for var in AUTHORIZE CHANGES RESULT; do
  # Set only the other two: naming the variable again after env -u would undo it.
  args=()
  for v in AUTHORIZE CHANGES RESULT; do
    [ "${v}" = "${var}" ] || args+=("${v}=success")
  done
  out=$(env -u "${var}" "${args[@]}" "${SCRIPT}" 2>&1) && rc=0 || rc=$?
  if [ "${rc}" -ne 0 ] && grep -q "${var}" <<<"${out}"; then
    echo "  ok: unset ${var} refuses"
    pass=$((pass + 1))
  else
    echo "  FAIL: unset ${var} did not refuse (rc=${rc}, out: ${out})"
    fail=$((fail + 1))
  fi
done

echo
echo "passed ${pass}, failed ${fail}"
[ "${fail}" -eq 0 ]
