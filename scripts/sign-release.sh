#!/usr/bin/env bash
# Sign, notarise and publish a release that CI has left as a draft.
#
#   scripts/sign-release.sh v4.1.25
#
# Runs on macOS, on the machine that holds the Developer ID key. That is the
# whole design: the key never leaves it, so the release cannot be completed
# anywhere else. The cost is deliberate and worth saying out loud — this machine
# is now the only place a release can be published from.
#
# WHAT IT DOES, and why each step is where it is:
#
#   1. downloads the draft's assets;
#   2. signs the two darwin binaries with the Developer ID;
#   3. notarises them. A bare executable CAN be notarised — measured 2026-09-02,
#      `notarytool` accepts it — but the ticket cannot be stapled to it
#      (`stapler` answers Error 73, and a zip is refused outright). So for a bare
#      binary Gatekeeper resolves the ticket ONLINE. That is enough for anyone
#      with a network and not enough for anyone without one;
#   4. builds one signed, notarised, STAPLED .dmg per architecture, which is the
#      only form measured to verify offline (`stapler validate` passes and
#      `spctl` reports "accepted, source=Notarized Developer ID");
#   5. regenerates SHA256SUMS **from the signed files**. Signing rewrites the
#      binary, so sums computed before it describe files that no longer exist;
#   6. publishes the draft.
#
# It refuses rather than half-finishing: no identity, no notary profile, no
# assets, a failed notarisation — each stops the run with what to do about it.
# A half-signed release is worse than an unsigned one, because it looks finished.

set -euo pipefail

TAG="${1:-}"
REPO="${REPO:-faradey/madock}"
SIGN_IDENTITY="${SIGN_IDENTITY:-}"
NOTARY_PROFILE="${NOTARY_PROFILE:-el-print-notary}"
# notarytool keeps profiles in the data-protection keychain, which a
# non-interactive shell cannot reach: without an explicit --keychain it answers
# "No Keychain password item found for profile" — the same words it uses when the
# credential really is absent. An hour was spent on that distinction on
# 2026-09-02, so the path is explicit here and overridable.
NOTARY_KEYCHAIN="${NOTARY_KEYCHAIN:-$HOME/Library/Keychains/login.keychain-db}"

die() { printf '%s\n' "$*" >&2; exit 1; }

[ -n "$TAG" ] || die "usage: scripts/sign-release.sh <tag>   (e.g. v4.1.25)"
[ "$(uname -s)" = "Darwin" ] || die "this runs on macOS: codesign, notarytool and stapler exist nowhere else"

command -v gh >/dev/null || die "gh is not installed"
gh auth status >/dev/null 2>&1 || [ -n "${GH_TOKEN:-}" ] || die \
  "gh cannot act on $REPO. The logged-in account is not the owner — run this as:
  php secrets.php --exec=github.faradey_pat -- scripts/sign-release.sh $TAG"

if [ -z "$SIGN_IDENTITY" ]; then
  SIGN_IDENTITY=$(security find-identity -v -p codesigning \
    | sed -n 's/.*"\(Developer ID Application:[^"]*\)".*/\1/p' | head -1)
fi
[ -n "$SIGN_IDENTITY" ] || die \
  "no Developer ID Application identity in the keychain. Nothing here can be signed;
  the certificate is the one Apple issues once — see VAULT.md before assuming it can be replaced."

xcrun notarytool history --keychain-profile "$NOTARY_PROFILE" --keychain "$NOTARY_KEYCHAIN" >/dev/null 2>&1 || die \
  "the notary profile \"$NOTARY_PROFILE\" does not answer in $NOTARY_KEYCHAIN.
  Recreate it from the App Store Connect key:
    xcrun notarytool store-credentials $NOTARY_PROFILE --key <AuthKey_*.p8> --key-id <id> --issuer <uuid> --keychain $NOTARY_KEYCHAIN"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

echo "==> downloading the draft assets of $TAG"
gh release download "$TAG" --repo "$REPO" --dir "$work" --clobber
ls "$work"/madock-darwin-arm64 "$work"/madock-darwin-amd64 >/dev/null 2>&1 || die \
  "the draft carries no darwin binaries — is $TAG the tag CI built?"

for arch in arm64 amd64; do
  bin="$work/madock-darwin-$arch"

  echo "==> signing madock-darwin-$arch"
  # --options runtime is what makes the binary eligible for notarisation at all,
  # and --timestamp is what keeps the signature valid after the certificate
  # expires. Neither is optional; both fail late rather than loudly if omitted.
  codesign --force --options runtime --timestamp --sign "$SIGN_IDENTITY" "$bin"
  codesign --verify --strict --verbose=2 "$bin"

  echo "==> notarising madock-darwin-$arch (online check only — nothing to staple to)"
  ditto -c -k --keepParent "$bin" "$work/$arch.zip"
  xcrun notarytool submit "$work/$arch.zip" \
    --keychain-profile "$NOTARY_PROFILE" --keychain "$NOTARY_KEYCHAIN" --wait

  echo "==> building the stapled disk image for $arch"
  root="$work/dmg-$arch"; mkdir -p "$root"
  cp "$bin" "$root/madock"
  dmg="$work/madock-${TAG}-darwin-${arch}.dmg"
  hdiutil create -quiet -volname "madock ${TAG}" -srcfolder "$root" -ov -format UDZO "$dmg"
  codesign --force --timestamp --sign "$SIGN_IDENTITY" "$dmg"
  xcrun notarytool submit "$dmg" \
    --keychain-profile "$NOTARY_PROFILE" --keychain "$NOTARY_KEYCHAIN" --wait
  xcrun stapler staple "$dmg"
  xcrun stapler validate "$dmg"
  # The verdict a user's Mac will reach, asked here rather than assumed.
  spctl -a -vvv -t open --context context:primary-signature "$dmg"
done

echo "==> regenerating SHA256SUMS from the signed files"
rm -f "$work/SHA256SUMS"
( cd "$work" && rm -f ./*.zip && rm -rf ./dmg-* && shasum -a 256 ./* | sed 's| \./| |' > SHA256SUMS.tmp && mv SHA256SUMS.tmp SHA256SUMS )
cat "$work/SHA256SUMS"

echo "==> uploading signed artifacts and publishing $TAG"
gh release upload "$TAG" --repo "$REPO" --clobber "$work"/madock-darwin-arm64 "$work"/madock-darwin-amd64 "$work"/*.dmg "$work"/SHA256SUMS
gh release edit "$TAG" --repo "$REPO" --draft=false

echo "done: $TAG is published, its macOS binaries are signed and notarised, and the disk images verify offline"
