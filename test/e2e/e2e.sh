#!/bin/sh
#
# Driver for the end-to-end suite. See lima.yaml for why the tests need a
# machine of their own.
#
#   ./test/e2e/e2e.sh up            build the VM (a few minutes, once)
#   ./test/e2e/e2e.sh run           build madock for linux, run the whole suite
#   ./test/e2e/e2e.sh run -run Foo  the same, narrowed to one test
#   ./test/e2e/e2e.sh shell         a shell inside the VM
#   ./test/e2e/e2e.sh down          shut it down, keep the disk
#   ./test/e2e/e2e.sh reset         delete it and build it again
#
# The suite is behind the `e2e` build tag, so `go test ./...` does not compile
# it and the pre-push hook stays fast. Nothing here runs on your machine's
# Docker daemon.

set -eu

VM_NAME="madock-e2e"

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
binary="$script_dir/.bin/madock"

die() {
    printf '\n\033[31m%s\033[0m\n\n' "$1" >&2
    exit 1
}

need_lima() {
    command -v limactl >/dev/null 2>&1 || die "limactl is not installed: brew install lima"
}

vm_exists() {
    limactl list --quiet 2>/dev/null | grep -qx "$VM_NAME"
}

vm_running() {
    [ "$(limactl list --format '{{.Status}}' "$VM_NAME" 2>/dev/null)" = "Running" ]
}

ensure_running() {
    vm_exists || die "the VM does not exist yet: $0 up"
    vm_running || limactl start "$VM_NAME"
}

# The guest runs the same architecture as the host — vz virtualises, it does not
# emulate — so the host's own arch is the right target.
build_binary() {
    goarch=$(go env GOHOSTARCH)
    mkdir -p "$script_dir/.bin"
    printf 'building madock for linux/%s\n' "$goarch"
    ( cd "$repo_root" && GOOS=linux GOARCH="$goarch" go build -o "$binary" . )
}

cmd_up() {
    need_lima
    if vm_exists; then
        vm_running && { printf '%s is already running\n' "$VM_NAME"; return 0; }
        limactl start "$VM_NAME"
        return 0
    fi
    limactl start --name="$VM_NAME" --tty=false "$script_dir/lima.yaml"

    # The provisioning adds the user to the docker group, but the SSH session
    # lima holds open was established before that happened and keeps the old
    # group set. Every madock command in the guest would then be denied the
    # docker socket. One restart is the whole fix, and it is only needed the
    # first time.
    printf 'restarting so the docker group takes effect\n'
    limactl stop "$VM_NAME" >/dev/null
    limactl start "$VM_NAME" >/dev/null
}

cmd_run() {
    need_lima
    ensure_running
    build_binary

    # -count=1 for the same reason the pre-push hook passes it: these tests are
    # driven by a binary and by docker, neither of which the build cache knows
    # anything about. A cached `ok` here would be worthless.
    #
    # MADOCK_E2E_BIN is how the tests find the binary. The repository is mounted
    # into the guest at its host path, read-only, so the binary built a moment
    # ago is already visible there.
    limactl shell "$VM_NAME" -- env \
        MADOCK_E2E_BIN="$binary" \
        PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin" \
        sh -c "cd '$repo_root' && go test -tags=e2e -count=1 -v -timeout=30m ./test/e2e/... $*"
}

cmd_shell() {
    need_lima
    ensure_running
    limactl shell "$VM_NAME"
}

cmd_down() {
    need_lima
    vm_exists || return 0
    limactl stop "$VM_NAME"
}

cmd_reset() {
    need_lima
    if vm_exists; then
        vm_running && limactl stop -f "$VM_NAME"
        limactl delete "$VM_NAME"
    fi
    cmd_up
}

cmd_status() {
    need_lima
    limactl list "$VM_NAME" 2>/dev/null || printf 'no VM named %s\n' "$VM_NAME"
}

command=${1:-}
[ $# -gt 0 ] && shift

case "$command" in
    up)     cmd_up ;;
    run)    cmd_run "$@" ;;
    shell)  cmd_shell ;;
    down)   cmd_down ;;
    reset)  cmd_reset ;;
    status) cmd_status ;;
    *)
        sed -n '2,20p' "$0" | sed 's/^#\{1,2\} \{0,1\}//'
        exit 1
        ;;
esac
