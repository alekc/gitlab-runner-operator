#!/usr/bin/env bash
# Render the CRD API reference from the Go types in api/.
#
# Prepends a meta description and a banner saying the page is generated, since
# the theme's edit button is global and an edit made through it is reverted by
# the next run. Material has no front-matter switch to hide that button, so the
# banner is the honest fix. Shared by `make docs` and `make docs-verify` so the
# two cannot render differently and make the drift check lie.
set -euo pipefail

: "${CRD_REF_DOCS:?CRD_REF_DOCS must point at the crd-ref-docs binary (make docs)}"

out=${1:?usage: gen-api-docs.sh <output-path>}

body=$(mktemp)
trap 'rm -f "${body}"' EXIT

"${CRD_REF_DOCS}" \
  --source-path=./api \
  --config=hack/crd-ref-docs.yaml \
  --renderer=markdown \
  --max-depth=14 \
  --output-path="${body}"

{
  printf '%s\n' \
    '---' \
    'description: >-' \
    '  Every field of the Runner and MultiRunner CRDs, generated from the Go' \
    '  types: authentication, executor config, volumes, resources and status.' \
    '---' \
    '' \
    '!!! info "Generated page"' \
    '' \
    '    Rendered from the Go types in `api/` by `make docs`. Edits here are' \
    '    reverted by the next run; change the types instead.' \
    ''
  cat "${body}"
} >"${out}"
