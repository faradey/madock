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

# Internal version bumps are dropped, and that is the whole of this filter.
#
# Releases are cut from a line that also carries `-norelease` tags — versions
# built for madock-pro to import, never published to anybody. The commits that
# raise those numbers are called `Release 4.1.23` or, in older history, just
# `4.1.17`, and a public release listing forty commits used to open with a dozen
# of them: version numbers a reader cannot download, describing steps between two
# releases they never saw. v4.1.25 shipped that way before it was withdrawn.
#
# What is dropped is only the bump commit itself. Everything it collected — the
# real changes — is already in the list under its own subject.
drop_version_bumps() {
    grep -Ev '^- (Release )?v?[0-9]+\.[0-9]+\.[0-9]+$' || true
}

echo "## What's Changed"
echo ""
if [ -n "$prev" ]; then
    git log --no-merges --pretty='- %s' "${prev}..${current}" | drop_version_bumps
else
    git log --no-merges --pretty='- %s' "${current}" | drop_version_bumps
fi
echo ""
if [ -n "$prev" ]; then
    echo "**Full Changelog**: https://github.com/${GITHUB_REPOSITORY:-faradey/madock}/compare/${prev}...${current}"
fi
