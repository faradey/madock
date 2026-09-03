#!/bin/sh
#
# Refuse a release whose number is not one we publish.
#
#   check-release-version.sh <tag>        # e.g. v4.2.0
#
# Reads src/version/version.go from the working directory. Prints nothing when
# the tag is fine; explains and exits non-zero when it is not.
#
# WHY THIS EXISTS. On 2026-09-02 v4.1.25 was published: a number taken from the
# internal `-norelease` line, which madock-pro imports and nobody downloads. The
# public line has always been single-digit in every segment — 3.8.x, 3.9.0,
# 4.0.0 — so the release page jumped from 4.0.0 to 4.1.25 in one step, and the
# release notes filled with `Release 4.1.2x` entries describing versions that
# were never published. It was withdrawn and reissued as 4.2.0.
#
# Nothing caught it. The tag was accepted, the workflow built, and the page went
# up. That is what this script is: the check that was missing, run before
# anything is built, so a wrong number costs a failed job rather than a
# withdrawn release.
#
# The second rule matters as much as the first. `const Version` is what the
# binary reports; the tag is what people download. When they disagree, everyone
# who checks `madock version` against the release page finds a discrepancy that
# no file explains.
set -eu

tag="${1:?usage: check-release-version.sh <tag>}"
version_file="src/version/version.go"

fail() { printf '%s\n' "$*" >&2; exit 1; }

# Single digit per segment. Not a style preference: it is what every published
# release has been, and it is what keeps the public numbering separate from the
# internal one, where the patch runs into the twenties.
case "$tag" in
    v[0-9].[0-9].[0-9]) ;;
    *) fail "refusing to release \"$tag\".

Published releases are single-digit in every segment: v3.8.6, v3.9.0, v4.0.0, v4.2.0.
Two digits anywhere means this is an internal number — those are tagged
\"<version>-norelease\" and never published.

If the next number would need two digits, the segment above it rolls over:
after v4.9.9 comes v5.0.0." ;;
esac

[ -f "$version_file" ] || fail "cannot read $version_file from $(pwd)"

declared=$(sed -n 's/^const Version = "\(.*\)"$/\1/p' "$version_file")
[ -n "$declared" ] || fail "no \`const Version\` found in $version_file"

if [ "v$declared" != "$tag" ]; then
    fail "the tag and the binary disagree about the version.

  tag:            $tag
  $version_file:  $declared

Whoever runs \`madock version\` on this release would see a number that appears
on no release page. Bump $version_file and tag the commit that carries it."
fi
