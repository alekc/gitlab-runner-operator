#!/usr/bin/env bash
# Regression cover for runner-release-watch.sh. gh and curl are stubbed, so
# nothing reaches GitHub or Docker Hub. The cases that matter most are the
# refusals: a parse failure, an empty tag list, a stale mirror or an empty key
# extraction must all refuse or say so, because a silent pass means a release
# goes unnoticed forever.
set -uo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT="${HERE}/runner-release-watch.sh"
ROOT=$(mktemp -d)
trap 'rm -rf "${ROOT}"' EXIT
STUB="${ROOT}/bin"
mkdir -p "${STUB}"

cat > "${STUB}/gh" <<'STUBEOF'
#!/usr/bin/env bash
echo "gh $*" >> "${GH_CALLS}"
# Faithful enough to catch flag regressions: --jq, --json field projection,
# --state, --label, the Accept header and the endpoint path all matter to the
# real gh, so a stub that ignores them turns assertions vacuous.
jqf=""; json=""; state=""; label=""; sv=""; accept=""; prev=""
for a in "$@"; do
  case "${prev}" in
    --jq) jqf="${a}" ;;
    --json) json="${a}" ;;
    --state) state="${a}" ;;
    --label) label="${a}" ;;
    --search) sv="${a}" ;;
    -H) accept="${a}" ;;
  esac
  prev="${a}"
done
emit() { if [ -n "${jqf}" ]; then jq -r "${jqf}"; else cat; fi; }
# Project to the requested --json fields, as the real gh does.
project() {
  if [ -z "${json}" ]; then cat; return; fi
  jq --arg f "${json}" '[.[] | with_entries(select(.key | inside($f)))]'
}
case "$*" in
  *"/tags?per_page="*) echo "unexpected mirror tag query" >&2; exit 1 ;;
  *"contents/common/config.go"*)
    [ "${GH_CONFIG_FAIL:-}" = "1" ] && exit 1
    case "$*" in *"repos/${EXPECT_MIRROR}/"*) ;; *) echo "wrong mirror" >&2; exit 1 ;; esac
    # Without the raw Accept header the real API returns base64 JSON.
    case "${accept}" in *vnd.github.raw*) ;; *) echo '{"content":"cGFja2FnZQo="}'; exit 0 ;; esac
    case "$*" in
      *"ref=v${PINNED_V}"*) cat "${GH_CONFIG_OLD}" ;;
      *) cat "${GH_CONFIG_NEW}" ;;
    esac
    exit 0 ;;
esac
if [ "$1" = "issue" ] && [ "$2" = "list" ]; then
  case "$*" in *"--repo ${GITHUB_REPOSITORY}"*) ;; *) echo "wrong repo" >&2; exit 1 ;; esac
  if [ -n "${sv}" ]; then
    [ "${GH_SEARCH_FAIL:-}" = "1" ] && exit 1
    # The real search only sees closed issues when asked for all states.
    if [ "${GH_TRACKED_STATE:-all}" = "closed" ] && [ "${state}" != "all" ]; then
      echo '[]' | project; exit 0
    fi
    case "${sv}" in
      *"${EXPECT_TITLE:-__none__}"*) project < "${GH_SEARCH_FIXTURE}" ;;
      *) echo '[]' | project ;;
    esac
    exit 0
  fi
  [ "${GH_LABEL_FAIL:-}" = "1" ] && exit 1
  # A label filter is a filter: without it the real API returns other issues.
  if [ -z "${label}" ]; then project < "${GH_UNLABELLED_FIXTURE}"; exit 0; fi
  if [ "${GH_TRACKED_STATE:-all}" = "closed" ] && [ "${state}" != "all" ]; then
    echo '[]' | project; exit 0
  fi
  project < "${GH_LIST_FIXTURE}"
  exit 0
fi
if [ "$1" = "label" ] && [ "$2" = "create" ]; then exit 0; fi
if [ "$1" = "issue" ] && [ "$2" = "create" ]; then
  [ -z "${label}" ] && { echo "create without --label" >&2; exit 1; }
  seen_title=""; prev=""
  for a in "$@"; do
    [ "${prev}" = "--body" ] && printf '%s' "${a}" > "${GH_BODY_CAPTURE}"
    [ "${prev}" = "--title" ] && seen_title="${a}"
    prev="${a}"
  done
  [ -z "${seen_title}" ] && { echo "create without --title" >&2; exit 1; }
  printf '%s' "${seen_title}" > "${GH_TITLE_CAPTURE}"
  echo "https://example.invalid/1"; exit 0
fi
exit 0
STUBEOF
chmod +x "${STUB}/gh"

cat > "${STUB}/curl" <<'STUBEOF'
#!/usr/bin/env bash
url=""
for a in "$@"; do case "${a}" in http*) url="${a}" ;; esac; done
echo "curl ${url}" >> "${GH_CALLS}"
case "${url}" in
  *"/releases"*)
    [ "${GITLAB_FAIL:-}" = "1" ] && exit 22
    cat "${GITLAB_FIXTURE}"; exit 0 ;;
  *gitlab-runner-helper*)
    [ "${HUB_FAIL:-}" = "1" ] && exit 22
    # The name filter is part of the request the real API honours.
    case "${url}" in *"name=v${EXPECT_VERSION}"*) ;; *) echo '{"count":0,"results":[]}'; exit 0 ;; esac
    cat "${HUB_FIXTURE}"; exit 0 ;;
esac
echo "unexpected url: ${url}" >&2
exit 1
STUBEOF
chmod +x "${STUB}/curl"

export PATH="${STUB}:${PATH}"
export GH_TOKEN=stub
export GITHUB_REPOSITORY=alekc/gitlab-runner-operator

pass=0
fail=0
check() {
  if grep -qF -- "$2" "$3"; then echo "  ok: $1"; pass=$((pass + 1))
  else echo "  FAIL: $1 (expected '$2')"; fail=$((fail + 1)); fi
}
absent() {
  if grep -qF -- "$2" "$3"; then echo "  FAIL: $1 (unexpected '$2')"; fail=$((fail + 1))
  else echo "  ok: $1"; pass=$((pass + 1)); fi
}
rc_is() {
  if [ "$2" = "$3" ]; then echo "  ok: $1"; pass=$((pass + 1))
  else echo "  FAIL: $1 (rc $2, want $3)"; fail=$((fail + 1)); fi
}
nonzero() { [ "$1" -ne 0 ] && echo nonzero || echo zero; }
# Grep inside one "### " section, so a key cannot satisfy an assertion by
# appearing under a different heading.
in_section() {
  local name=$1 needle=$2 file=$3
  if awk -v h="### ${name}" '$0 ~ "^"h {f=1;next} /^### /{f=0} f' "${file}" \
    | grep -qF -- "${needle}"; then
    echo "  ok: '${needle}' under '${name}'"; pass=$((pass + 1))
  else
    echo "  FAIL: '${needle}' not under '${name}'"; fail=$((fail + 1))
  fi
}

not_in_section() {
  local name=$1 needle=$2 file=$3
  if awk -v h="### ${name}" '$0 ~ "^"h {f=1;next} /^### /{f=0} f' "${file}" \
    | grep -qF -- "${needle}"; then
    echo "  FAIL: '${needle}' under '${name}'"; fail=$((fail + 1))
  else
    echo "  ok: '${needle}' not under '${name}'"; pass=$((pass + 1))
  fi
}

mk_types() {
  printf 'const DefaultRunnerImage = "gitlab/gitlab-runner:alpine-v%s"\n' "$1" > "${ROOT}/types.go"
}
# Two structs per fixture: a nested Kubernetes* type is included so narrowing
# the awk back to KubernetesConfig alone fails a test.
mk_config_at() {
  local out=$1 nested=$2; shift 2
  { echo 'type KubernetesConfig struct {'
    # A field referencing Referenced makes the closure walk reach it; a plain
    # sibling struct would be found by a roots-only extraction too.
    printf '\tCSI KubernetesCSI `toml:"csi,omitempty"`\n'
    printf '\tSvc Referenced `toml:"svc,omitempty"`\n'
    for k in "$@"; do printf '\tField string `toml:"%s,omitempty"`\n' "${k}"; done
    echo '}'
    echo 'type KubernetesCSI struct {'
    printf '\tField string `toml:"%s,omitempty"`\n' "${nested}"
    echo '}'
    echo 'type Referenced struct {'
    printf '\tField string `toml:"%s_via_ref,omitempty"`\n' "${nested}"
    echo '}'
  } > "${out}"
}
# Same three structs, but each nested struct's key is set independently, so a
# name can sit on one struct and not another. A flat name set cannot see that
# gap; a (struct, key) set can.
mk_config_pairs() {
  local out=$1 csi=$2 ref=$3; shift 3
  { echo 'type KubernetesConfig struct {'
    printf '\tCSI KubernetesCSI `toml:"csi,omitempty"`\n'
    printf '\tSvc Referenced `toml:"svc,omitempty"`\n'
    for k in "$@"; do printf '\tField string `toml:"%s,omitempty"`\n' "${k}"; done
    echo '}'
    echo 'type KubernetesCSI struct {'
    printf '\tField string `toml:"%s,omitempty"`\n' "${csi}"
    echo '}'
    echo 'type Referenced struct {'
    printf '\tField string `toml:"%s,omitempty"`\n' "${ref}"
    echo '}'
    printf 'Set helper image flavor (alpine, ubuntu)\n'
  } > "${out}"
}

export GH_CALLS="${ROOT}/calls.log"
export GH_BODY_CAPTURE="${ROOT}/body.md"
export GITLAB_FIXTURE="${ROOT}/releases.json"
export GH_UNLABELLED_FIXTURE="${ROOT}/unlabelled.json"
export GH_TITLE_CAPTURE="${ROOT}/title.txt"
export GH_LIST_FIXTURE="${ROOT}/list.json"
export GH_SEARCH_FIXTURE="${ROOT}/search.json"
export GH_CONFIG_OLD="${ROOT}/old.go"
export GH_CONFIG_NEW="${ROOT}/new.go"
export HUB_FIXTURE="${ROOT}/hub.json"
export RUNNER_WATCH_TYPES_FILE="${ROOT}/types.go"
export RUNNER_WATCH_CONFIG_FILE="${ROOT}/ours.go"
# Pinned at a fixture, or a run from the repo root would read the real
# exclusion list and a run from anywhere else would not.
export RUNNER_WATCH_SUPPRESS_FILE="${ROOT}/suppress"

reset() {
  : > "${GH_CALLS}"; : > "${GH_BODY_CAPTURE}"; : > "${GH_TITLE_CAPTURE}"
  # Deliberately not version-ordered: the real endpoint sorts by released_at,
  # and v19.10.0 also breaks a lexicographic maximum.
  printf '[{"tag_name":"v19.2.2"},{"tag_name":"v19.10.0"},{"tag_name":"v19.9.0"},{"tag_name":"v19.11.0-rc1"}]\n' \
    > "${GITLAB_FIXTURE}"
  printf '[]\n' > "${GH_LIST_FIXTURE}"
  printf '[]\n' > "${GH_SEARCH_FIXTURE}"
  # What an unfiltered list returns: dropping --label must not read as a hit.
  printf '[{"number":99,"title":"unrelated","body":"no marker here"}]\n' > "${GH_UNLABELLED_FIXTURE}"
  # x86_64-v… is the default flavour; ubi-fips is undocumented, so the
  # undocumented-flavour block is exercised rather than always empty.
  printf '{"count":4,"results":[{"name":"x86_64-v19.10.0"},{"name":"ubuntu-x86_64-v19.10.0"},{"name":"ubi-fips-v19.10.0"},{"name":"x86_64-v19.10.0-nanoserver1809"}]}\n' \
    > "${HUB_FIXTURE}"
  mk_config_at "${GH_CONFIG_OLD}" nested_shared shared_key gone_upstream
  mk_config_at "${GH_CONFIG_NEW}" nested_shared shared_key brand_new_key
  mk_config_at "${ROOT}/ours.go" nested_shared shared_key
  printf 'Set helper image flavor (alpine, ubuntu)\n' >> "${ROOT}/ours.go"
  : > "${ROOT}/suppress"
  export RUNNER_WATCH_SUPPRESS_FILE="${ROOT}/suppress"
  unset GH_CONFIG_FAIL HUB_FAIL GITLAB_FAIL GH_LABEL_FAIL GH_SEARCH_FAIL
  unset RUNNER_WATCH_DRY_RUN GH_TRACKED_STATE
  export PINNED_V=19.1.0
  export EXPECT_VERSION=19.10.0
  export EXPECT_MIRROR=gitlabhq/gitlab-runner
  export EXPECT_TITLE="gitlab-runner v19.10.0 released: review CRD and behaviour"
}

echo "case 1: pin already current, no issue created"
reset; mk_types 19.10.0
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o1"
check "reports up to date" "up to date" "${ROOT}/o1"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 2: newer release opens one issue with a correct delta"
reset; mk_types 19.1.0
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
check "creates an issue" "issue create" "${GH_CALLS}"
check "picks the numeric maximum" "v19.10.0" "${GH_BODY_CAPTURE}"
absent "ignores rc tags" "19.11.0" "${GH_BODY_CAPTURE}"
check "carries the marker" "<!-- runner-release: v19.10.0 -->" "${GH_BODY_CAPTURE}"
check "has an added heading" "### Added to the Kubernetes executor config upstream" "${GH_BODY_CAPTURE}"
in_section "Added to the Kubernetes executor config upstream" "KubernetesConfig.brand_new_key" "${GH_BODY_CAPTURE}"
check "has a removed heading" "### Removed from the Kubernetes executor config upstream" "${GH_BODY_CAPTURE}"
in_section "Removed from the Kubernetes executor config upstream" "KubernetesConfig.gone_upstream" "${GH_BODY_CAPTURE}"
absent "no false 'nothing added'" "No keys added to the Kubernetes executor config" "${GH_BODY_CAPTURE}"
check "reports helper flavours" "ubuntu" "${GH_BODY_CAPTURE}"
absent "flavours exclude arch tokens" '- `x86_64`' "${GH_BODY_CAPTURE}"
check "flags an undocumented flavour" "which omits" "${GH_BODY_CAPTURE}"
# Scoped to after "which omits": the flavour list above mentions every flavour,
# so a whole-body grep would pass even if the omission list were inverted.
if awk '/which omits/{f=1;next} f' "${GH_BODY_CAPTURE}" | grep -qF -- '- `ubi-fips`'; then
  echo "  ok: names the undocumented flavour"; pass=$((pass + 1))
else
  echo "  FAIL: ubi-fips not listed as omitted"; fail=$((fail + 1))
fi
if awk '/which omits/{f=1;next} f' "${GH_BODY_CAPTURE}" | grep -qF -- '- `ubuntu`'; then
  echo "  FAIL: documented flavour ubuntu wrongly listed as omitted"; fail=$((fail + 1))
else
  echo "  ok: documented flavour not listed as omitted"; pass=$((pass + 1))
fi

echo "case 3: nested Kubernetes* types are in scope"
reset; mk_types 19.1.0
mk_config_at "${GH_CONFIG_NEW}" nested_added shared_key
out=$("${SCRIPT}" 2>&1)
check "diffs a nested-type key" "KubernetesCSI.nested_added" "${GH_BODY_CAPTURE}"
# Referenced is reachable only through a field type, so a roots-only extraction
# would miss it entirely.
check "follows type references" "Referenced.nested_added_via_ref" "${GH_BODY_CAPTURE}"

echo "case 3b: a per-struct gap a flat name set cannot see"
reset; mk_types 19.1.0
# `shared_key` is present in both files, so a bare-name comparison is satisfied.
# Only upstream puts it on KubernetesCSI, and only we put `other_key` there.
mk_config_pairs "${GH_CONFIG_NEW}" shared_key ref_key shared_key
mk_config_pairs "${ROOT}/ours.go" other_key ref_key shared_key
out=$("${SCRIPT}" 2>&1)
in_section "In the upstream v19.10.0 Kubernetes executor config but not exposed by" "KubernetesCSI.shared_key" \
  "${GH_BODY_CAPTURE}"
in_section "Exposed by" "KubernetesCSI.other_key" "${GH_BODY_CAPTURE}"
# Pins the stale heading text: in_section above matches only its prefix, so
# without this the scope wording could be reverted with the suite still green.
check "stale heading names the scope" "but gone from the upstream v19.10.0 Kubernetes executor config" \
  "${GH_BODY_CAPTURE}"
# The exposed placement of the same name is reported nowhere, which is the
# whole point: the pair is what is compared, not the name.
absent "does not flag the exposed placement" "KubernetesConfig.shared_key" "${GH_BODY_CAPTURE}"
check "states the comparison performed" "Compared as (struct, key) pairs" "${GH_BODY_CAPTURE}"

echo "case 3c: an exclusion drops a pair from both directions"
reset; mk_types 19.1.0
mk_config_pairs "${GH_CONFIG_NEW}" shared_key ref_key shared_key
mk_config_pairs "${ROOT}/ours.go" other_key ref_key shared_key
printf '# why\nKubernetesCSI.shared_key\nKubernetesCSI.other_key\n' > "${ROOT}/suppress"
out=$("${SCRIPT}" 2>&1)
check "reports full exposure instead" "Every upstream Kubernetes executor key is exposed" "${GH_BODY_CAPTURE}"
absent "drops the stale section too" "### Exposed by" "${GH_BODY_CAPTURE}"
check "keeps the exclusion visible" "with 2 of them excluded by" "${GH_BODY_CAPTURE}"
# Upstream-against-itself is not filtered: a key added to a skipped subtree is
# still a real upstream change, and it is reported once, not every release.
in_section "Added to the Kubernetes executor config upstream" "KubernetesCSI.shared_key" "${GH_BODY_CAPTURE}"

echo "case 3d: an exclusion that matches nothing is reported"
reset; mk_types 19.1.0
printf 'KubernetesCSI.no_such_key\n' > "${ROOT}/suppress"
out=$("${SCRIPT}" 2>&1)
check "flags the dead entry" "that matched nothing" "${GH_BODY_CAPTURE}"
in_section "Exclusions in" "KubernetesCSI.no_such_key" "${GH_BODY_CAPTURE}"

echo "case 3e: a malformed exclusion refuses before filing"
reset; mk_types 19.1.0
printf '# comment\nnot-a-pair\n' > "${ROOT}/suppress"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o3e"
check "names the file" "malformed entries in" "${ROOT}/o3e"
check "quotes the entry" "not-a-pair" "${ROOT}/o3e"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 3f: no exclusion file is stated, not silently treated as empty"
reset; mk_types 19.1.0; rm -f "${ROOT}/suppress"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
check "still creates an issue" "issue create" "${GH_CALLS}"
check "says the file is absent" "no exclusion file at" "${GH_BODY_CAPTURE}"

echo "case 3g: a directory as the exclusion path refuses"
reset; mk_types 19.1.0
# A directory is readable and BSD sed exits 0 on one, so without the -f guard
# this reports an exclusion file that excludes nothing.
mkdir -p "${ROOT}/suppress.d"
export RUNNER_WATCH_SUPPRESS_FILE="${ROOT}/suppress.d"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o3g"
check "says it is not a file" "is not a readable file" "${ROOT}/o3g"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 4: label-matched issue suppresses a duplicate"
reset; mk_types 19.1.0
printf '[{"number":9,"body":"x\\n<!-- runner-release: v19.10.0 -->"}]\n' > "${GH_LIST_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o4"
check "says already tracked" "already tracked in #9" "${ROOT}/o4"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 5: title match suppresses even with the label removed"
reset; mk_types 19.1.0
printf '[{"number":11,"title":"gitlab-runner v19.10.0 released: review CRD and behaviour"}]\n' \
  > "${GH_SEARCH_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o5"
check "says already tracked" "already tracked in #11" "${ROOT}/o5"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 6: closed issues count as tracked, on both lookups"
reset; mk_types 19.1.0
printf '[{"number":12,"body":"<!-- runner-release: v19.10.0 -->"}]\n' > "${GH_LIST_FIXTURE}"
export GH_TRACKED_STATE=closed
out=$("${SCRIPT}" 2>&1)
printf '%s\n' "${out}" > "${ROOT}/o6a"
check "label lookup finds it closed" "already tracked in #12" "${ROOT}/o6a"
absent "does not re-file" "issue create" "${GH_CALLS}"
# Same, with the label removed so only the title search can find it.
reset; mk_types 19.1.0
printf '[{"number":13,"title":"gitlab-runner v19.10.0 released: review CRD and behaviour"}]\n' \
  > "${GH_SEARCH_FIXTURE}"
export GH_TRACKED_STATE=closed
out=$("${SCRIPT}" 2>&1)
printf '%s\n' "${out}" > "${ROOT}/o6b"
check "title search finds it closed" "already tracked in #13" "${ROOT}/o6b"
absent "does not re-file" "issue create" "${GH_CALLS}"

echo "case 6b: both lookups hitting the same issue is not a conflict"
reset; mk_types 19.1.0
printf '[{"number":21,"body":"<!-- runner-release: v19.10.0 -->"}]\n' > "${GH_LIST_FIXTURE}"
printf '[{"number":21,"title":"gitlab-runner v19.10.0 released: review CRD and behaviour"}]\n' \
  > "${GH_SEARCH_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o6c"
check "dedupes to one" "already tracked in #21" "${ROOT}/o6c"
absent "does not refuse" "refusing to guess" "${ROOT}/o6c"

echo "case 6c: a null body among results does not break the lookup"
reset; mk_types 19.1.0
printf '[{"number":5,"body":null},{"number":22,"body":"<!-- runner-release: v19.10.0 -->"}]\n' \
  > "${GH_LIST_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o6d"
check "still finds the tracked issue" "already tracked in #22" "${ROOT}/o6d"

echo "case 6d: a failed label lookup refuses rather than risking a duplicate"
reset; mk_types 19.1.0
printf '[{"number":23,"body":"<!-- runner-release: v19.10.0 -->"}]\n' > "${GH_LIST_FIXTURE}"
export GH_LABEL_FAIL=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o6e"
check "says why" "refusing to risk filing a duplicate" "${ROOT}/o6e"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 6e: a failed title lookup also refuses"
reset; mk_types 19.1.0; export GH_SEARCH_FAIL=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 7: two matches refuses to guess"
reset; mk_types 19.1.0
printf '[{"number":9,"body":"<!-- runner-release: v19.10.0 -->"}]\n' > "${GH_LIST_FIXTURE}"
printf '[{"number":10,"title":"gitlab-runner v19.10.0 released: review CRD and behaviour"}]\n' \
  > "${GH_SEARCH_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o7"
check "refuses to guess" "refusing to guess" "${ROOT}/o7"

echo "case 8: unparseable constant refuses"
reset; printf 'const Something = "nope"\n' > "${ROOT}/types.go"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o8"
check "names the constant" "could not parse DefaultRunnerImage" "${ROOT}/o8"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 9: unreadable config file refuses before writing anything"
reset; mk_types 19.1.0; export RUNNER_WATCH_CONFIG_FILE="${ROOT}/absent.go"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o9"
check "names the file" "cannot read" "${ROOT}/o9"
absent "does not create" "issue create" "${GH_CALLS}"
export RUNNER_WATCH_CONFIG_FILE="${ROOT}/ours.go"

echo "case 10: empty release list refuses rather than reporting current"
reset; mk_types 19.1.0; printf '[]\n' > "${GITLAB_FIXTURE}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o10"
check "refuses explicitly" "refusing to report up to date" "${ROOT}/o10"

echo "case 11: an unreachable gitlab.com refuses rather than reporting current"
reset; mk_types 19.1.0; export GITLAB_FAIL=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits non-zero" "$(nonzero "${rc}")" "nonzero"
printf '%s\n' "${out}" > "${ROOT}/o11"
check "names the source" "could not query" "${ROOT}/o11"
absent "does not create" "issue create" "${GH_CALLS}"

echo "case 12: empty key extraction is not reported as 'nothing added'"
reset; mk_types 19.1.0; printf 'package common\n' > "${GH_CONFIG_NEW}"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
check "still opens the issue" "issue create" "${GH_CALLS}"
check "says the structs moved" "structs may have moved" "${GH_BODY_CAPTURE}"
check "demands a manual compare" "Compare the executor config by hand" "${GH_BODY_CAPTURE}"
absent "no false reassurance" "No new CRD fields needed" "${GH_BODY_CAPTURE}"
check "marker survives" "<!-- runner-release: v19.10.0 -->" "${GH_BODY_CAPTURE}"

echo "case 13: upstream config fetch failure is stated, not hidden"
reset; mk_types 19.1.0; export GH_CONFIG_FAIL=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
check "flags the gap loudly" "Compare the executor config by hand" "${GH_BODY_CAPTURE}"
check "marker survives" "<!-- runner-release: v19.10.0 -->" "${GH_BODY_CAPTURE}"

echo "case 14: Docker Hub failure is not reported as 'not published'"
reset; mk_types 19.1.0; export HUB_FAIL=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
check "says the query failed" "Could not query Docker Hub" "${GH_BODY_CAPTURE}"
absent "does not claim absence" "has not published yet" "${GH_BODY_CAPTURE}"

echo "case 14b: a wrong-shaped Docker Hub payload is not read as an answer"
reset; mk_types 19.1.0
printf '{"cnt":9,"tags":[{"name":"alpine-v19.10.0"}]}\n' > "${HUB_FIXTURE}"
out=$("${SCRIPT}" 2>&1)
check "says the payload was unexpected" "unexpected payload" "${GH_BODY_CAPTURE}"
absent "does not claim absence" "has not published yet" "${GH_BODY_CAPTURE}"

echo "case 14c: Windows-only tags do not render an empty flavour list"
reset; mk_types 19.1.0
printf '{"count":2,"results":[{"name":"x86_64-v19.10.0-nanoserver1809"},{"name":"x86_64-v19.10.0-servercore21H2"}]}\n' \
  > "${HUB_FIXTURE}"
out=$("${SCRIPT}" 2>&1)
check "says no Linux flavour matched" "none matches a Linux flavour" "${GH_BODY_CAPTURE}"
absent "no empty bullet" '- ``' "${GH_BODY_CAPTURE}"
absent "does not claim a flavour list" "Linux flavours:" "${GH_BODY_CAPTURE}"

echo "case 14d: a config file without the flavour description says so"
reset; mk_types 19.1.0
mk_config_at "${ROOT}/ours.go" nested_shared shared_key
out=$("${SCRIPT}" 2>&1)
check "reports the missing description" "Could not find the" "${GH_BODY_CAPTURE}"
check "still lists flavours" "Linux flavours:" "${GH_BODY_CAPTURE}"

echo "case 15: genuinely absent helper tags warn against bumping"
reset; mk_types 19.1.0; printf '{"count":0,"results":[]}\n' > "${HUB_FIXTURE}"
out=$("${SCRIPT}" 2>&1)
check "warns about the helper" "has not published yet" "${GH_BODY_CAPTURE}"

echo "case 16: a truncated helper tag page is disclosed"
reset; mk_types 19.1.0
printf '{"count":250,"results":[{"name":"alpine-v19.10.0"}]}\n' > "${HUB_FIXTURE}"
out=$("${SCRIPT}" 2>&1)
check "discloses truncation" "flavour list may be short" "${GH_BODY_CAPTURE}"

echo "case 17: zero-padded versions do not abort or mis-compare"
reset; mk_types 19.1.0
printf '[{"tag_name":"v19.08.0"},{"tag_name":"v19.2.2"}]\n' > "${GITLAB_FIXTURE}"
export EXPECT_VERSION=19.8.0
export EXPECT_TITLE="gitlab-runner v19.08.0 released: review CRD and behaviour"
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o17"
absent "no arithmetic error" "value too great" "${ROOT}/o17"

echo "case 18: dry-run prints instead of creating"
reset; mk_types 19.1.0; export RUNNER_WATCH_DRY_RUN=1
out=$("${SCRIPT}" 2>&1); rc=$?
rc_is "exits 0" "${rc}" 0
printf '%s\n' "${out}" > "${ROOT}/o18"
check "prints the title" "gitlab-runner v19.10.0 released" "${ROOT}/o18"
absent "does not create" "issue create" "${GH_CALLS}"

echo
echo "passed ${pass}, failed ${fail}"
[ "${fail}" -eq 0 ]
