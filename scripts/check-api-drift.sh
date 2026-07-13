#!/usr/bin/env bash
#
# check-api-drift.sh — detect drift between the OpenAPI documents Energinet
# currently serves and the snapshots committed under api/.
#
# This library exists to track an API that has already been caught disagreeing
# with its own specification (the spec documents time series resolutions as
# P1D/P1Y while the live API sends PT1D/PT1Y, and it declares
# getchargelinkswithcharges on both APIs while the live API answers 404 for it).
# So a change to the published spec is a signal worth acting on, not noise.
#
# Usage:
#   scripts/check-api-drift.sh check              # default; diff live vs snapshots
#   scripts/check-api-drift.sh update             # rewrite the snapshots from live
#   scripts/check-api-drift.sh check --summary out.md
#
# Exit codes:
#   0  no drift (or, in update mode, snapshots written)
#   1  drift detected
#   2  the check could not be carried out (fetch failed, or the document that
#      came back is not a plausible Eloverblik OpenAPI document)
#
# Exit code 2 is deliberately distinct from 1. If api.eloverblik.dk serves an
# error page or a redirect, a naive differ would report "every endpoint was
# removed" and open a very alarming and very wrong issue. Anything that does not
# validate is an infrastructure problem, not drift.

set -euo pipefail

readonly BASE_URL="https://api.eloverblik.dk"
readonly APIS=(customerapi thirdpartyapi)

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT
readonly SNAPSHOT_DIR="${REPO_ROOT}/api"

MODE="check"
SUMMARY_FILE=""

die() {
	echo "error: $*" >&2
	exit 2
}

usage() {
	sed -n '2,26p' "${BASH_SOURCE[0]}" | sed 's/^#\{1,2\} \{0,1\}//'
	exit "${1:-0}"
}

# --- argument parsing -------------------------------------------------------

while [[ $# -gt 0 ]]; do
	case "$1" in
	check | update)
		MODE="$1"
		shift
		;;
	--summary)
		[[ $# -ge 2 ]] || die "--summary needs a file path"
		SUMMARY_FILE="$2"
		shift 2
		;;
	-h | --help)
		usage 0
		;;
	*)
		echo "error: unknown argument: $1" >&2
		usage 2
		;;
	esac
done

for tool in curl jq diff; do
	command -v "$tool" >/dev/null 2>&1 || die "required tool not found: ${tool}"
done

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# --- fetching and normalising ----------------------------------------------

# spec_url <api> — the document docs.eloverblik.dk renders for that API.
spec_url() {
	printf '%s/%s/swagger/%s-v1.0/swagger.json' "${BASE_URL}" "$1" "$1"
}

# fetch_spec <api> <dest> — download and sanity-check one OpenAPI document.
#
# The validation below is what stops a bad gateway or a moved host from being
# misreported as drift: the response must parse as a JSON object, declare an
# OpenAPI version, carry a non-empty schema section, and contain the two paths
# that have always been there. A document missing those is not trustworthy input
# for a diff, whatever its HTTP status said.
fetch_spec() {
	local api="$1" dest="$2"
	local url http_code rc=0

	url="$(spec_url "${api}")"
	echo "==> fetching ${url}" >&2

	http_code="$(
		curl --silent --show-error --location \
			--retry 5 --retry-delay 10 --retry-all-errors --retry-connrefused \
			--fail-with-body --max-time 60 \
			--user-agent 'go-eloverblik api-drift check (+https://github.com/slimcdk/go-eloverblik)' \
			--write-out '%{http_code}' \
			--output "${dest}" \
			"${url}"
	)" || rc=$?

	if [[ ${rc} -ne 0 || "${http_code}" != "200" ]]; then
		die "GET ${url} returned HTTP ${http_code:-<none>} (curl exit ${rc}); refusing to treat this as drift"
	fi

	jq -e 'type == "object"' "${dest}" >/dev/null 2>&1 ||
		die "${url} did not return a JSON object"
	jq -e 'has("openapi") and has("paths")' "${dest}" >/dev/null 2>&1 ||
		die "${url} returned JSON without .openapi/.paths; not an OpenAPI document"
	jq -e '(.components.schemas // {}) | length > 0' "${dest}" >/dev/null 2>&1 ||
		die "${url} returned an OpenAPI document with no schemas"

	local expected
	for expected in "/${api}/api/token" "/${api}/api/isalive"; do
		jq -e --arg p "${expected}" '.paths | has($p)' "${dest}" >/dev/null 2>&1 ||
			die "${url} is missing the expected path ${expected}; refusing to treat this as drift"
	done
}

# normalise <src> <dest> — pretty-print with recursively sorted object keys.
#
# `jq -S` sorts every object's keys by codepoint, so the byte layout of the
# snapshot depends only on the document's content and not on the order the
# server happened to serialise it in. That is what keeps the committed files
# from churning and keeps the diff readable. Array order is left alone on
# purpose: the order of `enum`, `required` and `parameters` entries is part of
# the document, and a reordering there is drift we want to see.
normalise() {
	jq -S --indent 2 '.' "$1" >"$2"
}

# --- reporting --------------------------------------------------------------

# operations <file> — sorted "METHOD /path" lines, one per documented operation.
operations() {
	jq -r '
		(.paths // {}) | to_entries[] as $path
		| $path.value | to_entries[]
		| select(.key | ascii_downcase
			| IN("get","put","post","delete","patch","head","options","trace"))
		| "\(.key | ascii_upcase) \($path.key)"
	' "$1" | LC_ALL=C sort
}

# schema_names <file> — sorted names of every component schema.
schema_names() {
	jq -r '(.components.schemas // {}) | keys[]' "$1" | LC_ALL=C sort
}

# schema_body <file> <name> — one schema, canonicalised, for value comparison.
schema_body() {
	jq -S -c --arg n "$2" '.components.schemas[$n]' "$1"
}

# bullets <heading> <file> — emit a markdown list, or nothing when the file is empty.
#
# The single quotes below are load-bearing and the backticks inside them are
# markdown, not command substitution — which is exactly why they must stay
# single-quoted. shellcheck cannot tell the difference, hence the disable.
# shellcheck disable=SC2016
bullets() {
	[[ -s "$2" ]] || return 0
	printf '%s\n\n' "$1"
	sed 's/^/- `/; s/$/`/' "$2"
	printf '\n'
}

# summarise <api> <old> <new> — human-readable account of what changed.
#
# As in bullets(): backticks in the format strings are markdown. Keep them
# single-quoted.
# shellcheck disable=SC2016
summarise() {
	local api="$1" old="$2" new="$3"
	local w="${WORK_DIR}/${api}"
	mkdir -p "${w}"

	operations "${old}" >"${w}/ops.old"
	operations "${new}" >"${w}/ops.new"
	schema_names "${old}" >"${w}/schemas.old"
	schema_names "${new}" >"${w}/schemas.new"

	comm -13 "${w}/ops.old" "${w}/ops.new" >"${w}/ops.added"
	comm -23 "${w}/ops.old" "${w}/ops.new" >"${w}/ops.removed"
	comm -13 "${w}/schemas.old" "${w}/schemas.new" >"${w}/schemas.added"
	comm -23 "${w}/schemas.old" "${w}/schemas.new" >"${w}/schemas.removed"

	# A schema that exists on both sides but whose definition differs — the
	# category that quietly breaks unmarshalling, so it is worth calling out
	# separately from added/removed.
	local name
	: >"${w}/schemas.changed"
	while IFS= read -r name; do
		[[ -n "${name}" ]] || continue
		if [[ "$(schema_body "${old}" "${name}")" != "$(schema_body "${new}" "${name}")" ]]; then
			printf '%s\n' "${name}" >>"${w}/schemas.changed"
		fi
	done < <(comm -12 "${w}/schemas.old" "${w}/schemas.new")

	printf '### `%s`\n\n' "${api}"

	bullets '**Endpoints added**' "${w}/ops.added"
	bullets '**Endpoints removed**' "${w}/ops.removed"
	bullets '**Schemas added**' "${w}/schemas.added"
	bullets '**Schemas removed**' "${w}/schemas.removed"
	bullets '**Schemas changed**' "${w}/schemas.changed"

	if [[ ! -s "${w}/ops.added" && ! -s "${w}/ops.removed" &&
		! -s "${w}/schemas.added" && ! -s "${w}/schemas.removed" &&
		! -s "${w}/schemas.changed" ]]; then
		printf 'No endpoint or schema changed. The difference is in descriptions, examples or other metadata.\n\n'
	fi

	# `diff` exits 1 when the files differ, which is the whole point here, so its
	# status must not be allowed to trip `set -e`. Write it to a file rather than
	# piping into `head`: under `pipefail` that pipeline would also report 1.
	diff -u --label "a/api/${api}.json" --label "b/api/${api}.json" \
		"${old}" "${new}" >"${w}/diff.txt" || true

	printf '<details>\n<summary>Diff of <code>api/%s.json</code></summary>\n\n```diff\n' "${api}"
	# Cap the diff: a GitHub issue body is limited to 65536 characters and a
	# rewritten description block alone can run to thousands of lines.
	head -n 300 "${w}/diff.txt"
	if [[ "$(wc -l <"${w}/diff.txt")" -gt 300 ]]; then
		printf '... diff truncated at 300 lines; run the job or the script locally for the rest.\n'
	fi
	printf '```\n\n</details>\n\n'
}

# --- main -------------------------------------------------------------------

mkdir -p "${SNAPSHOT_DIR}"

drift=0
for api in "${APIS[@]}"; do
	fetch_spec "${api}" "${WORK_DIR}/${api}.raw"
	normalise "${WORK_DIR}/${api}.raw" "${WORK_DIR}/${api}.json"
done

if [[ "${MODE}" == "update" ]]; then
	for api in "${APIS[@]}"; do
		cp "${WORK_DIR}/${api}.json" "${SNAPSHOT_DIR}/${api}.json"
		echo "==> wrote api/${api}.json" >&2
	done
	echo "snapshots updated; review and commit api/*.json" >&2
	exit 0
fi

summary="${WORK_DIR}/summary.md"
: >"${summary}"

for api in "${APIS[@]}"; do
	snapshot="${SNAPSHOT_DIR}/${api}.json"
	[[ -f "${snapshot}" ]] || die "missing snapshot ${snapshot}; run 'scripts/check-api-drift.sh update' to create it"

	if diff -q "${snapshot}" "${WORK_DIR}/${api}.json" >/dev/null; then
		echo "==> ${api}: no drift" >&2
		continue
	fi

	echo "==> ${api}: DRIFT" >&2
	drift=1
	summarise "${api}" "${snapshot}" "${WORK_DIR}/${api}.json" >>"${summary}"
done

if [[ ${drift} -eq 0 ]]; then
	echo "no drift: the published OpenAPI documents match the snapshots in api/" >&2
	exit 0
fi

# A fingerprint of the *live* documents. Two runs that see the same upstream
# change produce the same fingerprint, which is how the workflow tells "this is
# the change I already filed an issue for" from "upstream changed again".
fingerprint="$(cat "${WORK_DIR}/customerapi.json" "${WORK_DIR}/thirdpartyapi.json" |
	sha256sum | cut -c1-12)"

# Markdown again: every backtick below is a code fence or a code span.
# shellcheck disable=SC2016
{
	printf '<!-- api-drift-fingerprint: %s -->\n' "${fingerprint}"
	printf 'The OpenAPI documents published by Energinet no longer match the snapshots committed in `api/`.\n\n'
	printf 'Source documents:\n\n'
	for api in "${APIS[@]}"; do
		printf -- '- [`%s`](%s)\n' "${api}" "$(spec_url "${api}")"
	done
	printf '\n'
	cat "${summary}"
	printf -- '---\n\n'
	printf '**If this change is intentional / expected**, refresh the snapshots:\n\n'
	printf '```sh\nscripts/check-api-drift.sh update\ngit add api/\ngit commit -m "chore(api): refresh OpenAPI snapshots"\n```\n\n'
	printf 'Then check whether `v1/` needs to follow the change — and remember that the\n'
	printf 'published spec has been wrong before, so confirm against the live API before\n'
	printf 'changing the client to match the document.\n'
} >"${WORK_DIR}/issue.md"

if [[ -n "${SUMMARY_FILE}" ]]; then
	cp "${WORK_DIR}/issue.md" "${SUMMARY_FILE}"
fi

cat "${WORK_DIR}/issue.md"

exit 1
