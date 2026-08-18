#!/usr/bin/env bash
# Open one tracking issue per new upstream gitlab-runner release, carrying the
# executor config-key delta so a reviewer can tell whether the release needs new
# CRD fields. Review trigger, not a version bump.
set -euo pipefail
# comm and sort must agree on collation regardless of the runner's locale.
export LC_ALL=C

: "${GH_TOKEN:?GH_TOKEN must be set}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}"

LABEL=${RUNNER_WATCH_LABEL:-upstream-release}
TYPES_FILE=${RUNNER_WATCH_TYPES_FILE:-api/v1beta2/runner_types.go}
CONFIG_FILE=${RUNNER_WATCH_CONFIG_FILE:-api/v1beta2/gitlab_types.go}
MIRROR=${RUNNER_WATCH_MIRROR:-gitlabhq/gitlab-runner}
GITLAB_API=${RUNNER_WATCH_GITLAB_API:-https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab-runner}
HUB=${RUNNER_WATCH_HUB:-https://hub.docker.com/v2/repositories}
DRY_RUN=${RUNNER_WATCH_DRY_RUN:-}

# The pin is a Go constant rather than a dependency, so nothing else would
# notice a rename. Fail loudly instead of comparing against an empty string.
pinned=$(sed -nE \
  's/^const DefaultRunnerImage = "gitlab\/gitlab-runner:alpine-v([0-9]+\.[0-9]+\.[0-9]+)".*/\1/p' \
  "${TYPES_FILE}")
if [ -z "${pinned}" ]; then
  echo "could not parse DefaultRunnerImage from ${TYPES_FILE}" >&2
  exit 1
fi
# Read inside a command substitution below, where errexit does not apply, so a
# missing file would otherwise produce a nonsense issue body rather than a stop.
if [ ! -r "${CONFIG_FILE}" ]; then
  echo "cannot read ${CONFIG_FILE}; refusing to report a config delta" >&2
  exit 1
fi

# Field-wise so the result does not depend on GNU sort -V being present. 10# so
# a zero-padded component cannot abort the arithmetic as an invalid octal.
vgt() {
  local a b i
  IFS=. read -r -a a <<<"$1"
  IFS=. read -r -a b <<<"$2"
  for i in 0 1 2; do
    ((10#${a[i]:-0} > 10#${b[i]:-0})) && return 0
    ((10#${a[i]:-0} < 10#${b[i]:-0})) && return 1
  done
  return 1
}

# Releases come from gitlab.com, which is authoritative. Asking a GitHub mirror
# instead would report "up to date" forever if its tag refs stopped syncing.
releases=$(curl -sf "${GITLAB_API}/releases?per_page=100") || {
  echo "could not query ${GITLAB_API} for releases; refusing to report up to date" >&2
  exit 1
}
# The endpoint orders by released_at, not by version, so take the numeric max.
latest=$(printf '%s' "${releases}" \
  | jq -r '.[].tag_name | select(test("^v[0-9]+\\.[0-9]+\\.[0-9]+$")) | ltrimstr("v")' \
  | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)
if [ -z "${latest}" ]; then
  echo "no stable releases in the gitlab.com response; refusing to report up to date" >&2
  exit 1
fi

if ! vgt "${latest}" "${pinned}"; then
  echo "up to date: pinned v${pinned}, newest upstream v${latest}"
  exit 0
fi

marker="<!-- runner-release: v${latest} -->"
title="gitlab-runner v${latest} released: review CRD and behaviour"

# Two lookups on purpose. The label query is immediately consistent but a human
# can remove the label; the title search survives that but lags indexing. Each
# is captured separately because errexit does not reach inside $( ).
by_label=$(gh issue list --repo "${GITHUB_REPOSITORY}" --label "${LABEL}" --state all \
  --limit 100 --json number,body \
  | jq -r --arg m "${marker}" '.[] | select((.body // "") | contains($m)) | .number') || {
  echo "label lookup failed; refusing to risk filing a duplicate" >&2
  exit 1
}
# Quote the phrase: an unquoted title lets GitHub read punctuation as search
# syntax and silently return nothing, which would drop us to label-only.
by_title=$(gh issue list --repo "${GITHUB_REPOSITORY}" --state all --limit 100 \
  --search "\"${title}\" in:title" --json number,title \
  | jq -r --arg t "${title}" '.[] | select(.title == $t) | .number') || {
  echo "title lookup failed; refusing to risk filing a duplicate" >&2
  exit 1
}
tracked=$(printf '%s\n%s\n' "${by_label}" "${by_title}" | grep -E '^[0-9]+$' | sort -u || true)
count=$(printf '%s' "${tracked}" | grep -c . || true)
if [ "${count}" -gt 1 ]; then
  printf 'found %s issues for v%s (%s); refusing to guess\n' "${count}" "${latest}" \
    "$(printf '%s' "${tracked}" | tr '\n' ' ')" >&2
  exit 1
fi
if [ "${count}" -eq 1 ]; then
  echo "already tracked in #$(printf '%s' "${tracked}")"
  exit 0
fi

# Every struct reachable from a Kubernetes* root, not just the roots: the
# operator mirrors the nested types too, so a key added to one of those still
# needs a CRD field. Following references keeps that self-maintaining.
KEYS_AWK='
/^type [A-Za-z0-9_]+ struct \{$/ { t = $2; seen[t] = 1; next }
/^\}$/ { t = ""; next }
t != "" {
  line = $0
  if (match(line, /toml:"[^",]+/)) keys[t] = keys[t] " " substr(line, RSTART + 6, RLENGTH - 6)
  sub(/`.*$/, "", line)
  sub(/^[[:space:]]*[A-Za-z0-9_]+[[:space:]]*/, "", line)
  while (match(line, /[A-Z][A-Za-z0-9_]*/)) {
    refs[t] = refs[t] " " substr(line, RSTART, RLENGTH)
    line = substr(line, RSTART + RLENGTH)
  }
}
END {
  for (x in seen) if (x ~ /^Kubernetes/) queue[++n] = x
  for (i = 1; i <= n; i++) {
    cur = queue[i]
    if (cur in done) continue
    done[cur] = 1
    split(refs[cur], r, " ")
    for (j in r) if (r[j] != "" && (r[j] in seen) && !(r[j] in done)) queue[++n] = r[j]
  }
  for (x in done) { split(keys[x], kk, " "); for (j in kk) if (kk[j] != "") print kk[j] }
}'
kube_keys() { awk "${KEYS_AWK}" "$1" | sort -u; }

fetch_upstream_config() {
  gh api "repos/${MIRROR}/contents/common/config.go?ref=v$1" \
    -H "Accept: application/vnd.github.raw"
}

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

by_hand() {
  echo "**Compare the executor config by hand before closing this.**"
}

config_section() {
  local old="${tmp}/old.go" new="${tmp}/new.go"
  if ! fetch_upstream_config "${pinned}" >"${old}" 2>/dev/null ||
    ! fetch_upstream_config "${latest}" >"${new}" 2>/dev/null; then
    echo "Could not read the upstream executor config for v${pinned} or v${latest}."
    by_hand
    return
  fi
  # An empty extraction means the structs moved or were renamed. Saying "no keys
  # added" there would be the most misleading output this tool emits. All three
  # inputs are checked: a silent zero on any side skews the whole comparison.
  local f
  for f in "${old}:v${pinned}" "${new}:v${latest}" "${CONFIG_FILE}:${CONFIG_FILE}"; do
    if [ -z "$(kube_keys "${f%%:*}")" ]; then
      echo "Extracted no toml keys from ${f##*:}; the config structs may have moved."
      by_hand
      return
    fi
  done
  local added removed missing
  added=$(comm -13 <(kube_keys "${old}") <(kube_keys "${new}") | sed 's/^/- `/;s/$/`/')
  removed=$(comm -23 <(kube_keys "${old}") <(kube_keys "${new}") | sed 's/^/- `/;s/$/`/')
  missing=$(comm -23 <(kube_keys "${new}") <(kube_keys "${CONFIG_FILE}") | sed 's/^/- `/;s/$/`/')

  printf 'Upstream executor toml keys, v%s to v%s:\n\n' "${pinned}" "${latest}"
  if [ -n "${added}" ]; then
    printf '### Added upstream, so likely new CRD fields\n\n%s\n\n' "${added}"
  else
    printf '### No keys added upstream\n\nNo new CRD fields needed for the executor config.\n\n'
  fi
  [ -n "${removed}" ] && printf '### Removed upstream, so possibly dead here\n\n%s\n\n' "${removed}"
  if [ -n "${missing}" ]; then
    printf '### In upstream v%s but not exposed by `%s`\n\n%s\n\n' \
      "${latest}" "${CONFIG_FILE}" "${missing}"
  fi
}

helper_section() {
  local json rc=0 total shown flavours documented undocumented esc
  json=$(curl -sf "${HUB}/gitlab/gitlab-runner-helper/tags/?page_size=100&name=v${latest}") || rc=$?
  # A failed query is not evidence of absence, and conflating the two would
  # advise against a bump on no information at all.
  if [ "${rc}" -ne 0 ]; then
    printf 'Could not query Docker Hub for helper tags (curl exit %s). **Check the helper image by hand.**\n' "${rc}"
    return
  fi
  # A 200 carrying HTML, or JSON whose shape changed, must not read as an
  # answer: without this, renamed fields report "not published" on a full list.
  if ! printf '%s' "${json}" | jq -e '.results | type == "array"' >/dev/null 2>&1; then
    printf 'Docker Hub returned an unexpected payload for helper tags. **Check the helper image by hand.**\n'
    return
  fi
  total=$(printf '%s' "${json}" | jq -r '.count // (.results | length)')
  shown=$(printf '%s' "${json}" | jq -r '.results | length')
  if [ "${shown}" -eq 0 ]; then
    printf 'No `gitlab-runner-helper` tags for v%s. **The helper image has not published yet; do not bump until it has.**\n' "${latest}"
    return
  fi
  esc=${latest//./\\.}
  # The version-anchored grep already drops Windows tags, whose OS token comes
  # after the version; the filter is belt and braces if that shape changes.
  flavours=$(printf '%s' "${json}" | jq -r '.results[].name' \
    | grep -vE 'nanoserver|servercore' | sed -e 's/-pwsh$//' \
    | grep -E -- "(^|-)v${esc}\$" | sed -E "s/-?v${esc}\$//" \
    | sed -E 's/-?(x86_64|arm64|arm|ppc64le|riscv64|s390x)$//' \
    | sed -e 's/^$/(default)/' | sort -u)
  printf 'Helper image published for v%s (%s tags). Linux flavours:\n\n' "${latest}" "${total}"
  printf '%s\n' "${flavours}" | sed 's/^/- `/;s/$/`/'
  if [ "${total}" -gt "${shown}" ]; then
    printf '\nOnly the first %s of %s tags were read, so the flavour list may be short.\n' \
      "${shown}" "${total}"
  fi
  documented=$(grep -o 'Set helper image flavor ([^)]*)' "${CONFIG_FILE}" | head -1)
  if [ -z "${documented}" ]; then
    printf '\nCould not find the `helper_image_flavor` description in `%s` to compare against.\n' \
      "${CONFIG_FILE}"
    return
  fi
  undocumented=$(printf '%s\n' "${flavours}" | grep -v '(default)' | while read -r f; do
    # Compare on the alphabetic stem so alpine3.21 counts as alpine. An empty
    # stem would match anything, so treat it as undocumented instead.
    stem=${f%%[0-9.]*}
    if [ -z "${stem}" ]; then printf '%s\n' "${f}"; continue; fi
    case "${documented}" in *"${stem}"*) ;; *) printf '%s\n' "${f}" ;; esac
  done)
  if [ -n "${undocumented}" ]; then
    printf '\n`helper_image_flavor` describes itself as "%s", which omits:\n\n%s\n' \
      "${documented#Set helper image flavor }" \
      "$(printf '%s\n' "${undocumented}" | sed 's/^/- `/;s/$/`/')"
  fi
}

body=$(
  cat <<BODY
Upstream released **v${latest}**; \`DefaultRunnerImage\` pins **v${pinned}**.

- [Release notes](https://gitlab.com/gitlab-org/gitlab-runner/-/releases/v${latest})
- [CHANGELOG](https://gitlab.com/gitlab-org/gitlab-runner/-/blob/v${latest}/CHANGELOG.md)
- Runner image: \`gitlab/gitlab-runner:alpine-v${latest}\`

## Executor config

$(config_section)

## Helper image

$(helper_section)

## Before closing

- [ ] Decide whether any added key needs a CRD field in \`api/v1beta2\`.
- [ ] Check the release notes for behaviour changes the reconciler assumes.
- [ ] Bump \`DefaultRunnerImage\` and let e2e exercise the new runner.

${marker}
BODY
)

if [ -n "${DRY_RUN}" ] && [ "${DRY_RUN}" != "0" ]; then
  printf '%s\n\n%s\n' "${title}" "${body}"
  exit 0
fi
# Self-provision the label, as hack/cve-report.sh does: gh resolves label names
# before the create, so a missing one fails after all the work is done.
gh label create "${LABEL}" --repo "${GITHUB_REPOSITORY}" --force \
  --color fbca04 --description "New upstream gitlab-runner release" >/dev/null
gh issue create --repo "${GITHUB_REPOSITORY}" --label "${LABEL}" \
  --title "${title}" --body "${body}"
