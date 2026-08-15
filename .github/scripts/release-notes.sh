#!/bin/sh
#
# Build the body of a GitHub Release.
#
# The first section is the one that exists because a reader gets five lines of
# attention and a release can carry forty-four versions: everything the CHANGELOG
# marked `Upgrading:` between the previous published release and this tag, hoisted
# to the top. v3.9.0 shipped without it and three behaviour changes — the front door
# following the main service, static files served no-cache, and commands refusing to
# run outside a project — sat inside long entries where nobody would meet them
# before their scripts did.
#
# Everything below that is the commit subjects, as before.
#
#   release-notes.sh <current-tag> [previous-tag]
#
# Reads CHANGELOG.md from the working directory. Prints the notes on stdout.
set -eu

current="$1"
prev="${2:-}"

changelog="CHANGELOG.md"

# Version headings look like `**v3.9.0**` on a line of their own. Section labels
# inside a version are `Added:`, `Fixed:`, `Changed:`, `Released:`, `Upgrading:`.
upgrading_notes() {
    [ -f "$changelog" ] || return 0

    awk -v current="$current" -v prev="$prev" '
        function version_of(line,   v) {
            v = line
            gsub(/^\*\*|\*\*$/, "", v)
            return v
        }
        /^\*\*v[0-9]+\.[0-9]+\.[0-9]+/ {
            version = version_of($0)
            # Stop at the previous published release: everything above it is what
            # this release adds.
            if (prev != "" && version == prev) { exit }
            in_range = 1
            in_upgrading = 0
            next
        }
        # A new section label ends the previous one.
        /^[A-Z][A-Za-z ]*:$/ {
            in_upgrading = ($0 == "Upgrading:")
            next
        }
        in_range && in_upgrading && /^- / { print }
    ' "$changelog"
}

notes=$(upgrading_notes)

if [ -n "$notes" ]; then
    echo "## Before you upgrade"
    echo ""
    echo "$notes"
    echo ""
fi

echo "## What's Changed"
echo ""
if [ -n "$prev" ]; then
    git log --no-merges --pretty='- %s' "${prev}..${current}"
else
    git log --no-merges --pretty='- %s' "${current}"
fi
echo ""
if [ -n "$prev" ]; then
    echo "**Full Changelog**: https://github.com/${GITHUB_REPOSITORY:-faradey/madock}/compare/${prev}...${current}"
fi
