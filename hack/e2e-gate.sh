#!/usr/bin/env bash
# Decide whether the e2e suite result should let a PR merge. Named the single
# required status check because its name is stable, unlike the per-version e2e
# legs whose matrix is derived at run time.
set -euo pipefail

: "${AUTHORIZE:?AUTHORIZE must be set (the authorize job result)}"
: "${CHANGES:?CHANGES must be set (the changes job result)}"
: "${RESULT:?RESULT must be set (the e2e job result)}"

# A broken upstream job also skips e2e, so judging only RESULT would read that
# skip as "nothing to do" and pass a PR whose suite never ran.
for stage in "authorize:${AUTHORIZE}" "changes:${CHANGES}"; do
  case "${stage#*:}" in
    failure | cancelled)
      echo "${stage%%:*} ${stage#*:}; cannot judge the suite" >&2
      exit 1
      ;;
  esac
done

# skipped is a pass on purpose, covering two legitimate cases: a PR touching no
# Go code, and a fork PR with no secrets until a maintainer comments
# "/launch e2e". Failing either would leave those PRs permanently unmergeable.
case "${RESULT}" in
  success)
    echo "e2e passed"
    ;;
  skipped)
    echo "e2e skipped (go changes: ${NEEDED:-n/a}); nothing to gate on"
    ;;
  *)
    echo "e2e result is '${RESULT}'" >&2
    exit 1
    ;;
esac
