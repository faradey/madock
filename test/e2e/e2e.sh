#!/bin/sh
#
# Driver for the end-to-end suite. See lima.yaml for why the tests need a
# machine of their own.
#
#   ./test/e2e/e2e.sh up            build the VM (a few minutes, once)
#   ./test/e2e/e2e.sh run           build madock for linux, run the whole suite
#   ./test/e2e/e2e.sh run -run Foo  the same, narrowed to one test
#   ./test/e2e/e2e.sh run --platforms   include the platform tests (an hour)
#   ./test/e2e/e2e.sh shell         a shell inside the VM
#   ./test/e2e/e2e.sh down          shut it down, keep the disk
#   ./test/e2e/e2e.sh reset         delete it and build it again
#   ./test/e2e/e2e.sh auth          give the VM your composer credentials
#   ./test/e2e/e2e.sh run-local     no VM — only where the daemon is disposable
#
# The suite is behind the `e2e` build tag, so `go test ./...` does not compile
# it and the pre-push hook stays fast. Nothing here touches your machine's own
# Docker daemon — except run-local, which is for CI runners and refuses to start
# anywhere that has not said it is one.

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

# ensure_free refuses to start while another suite is running anywhere on this
# machine.
#
# It began as a check inside one VM, when madock and madock-pro shared it: the
# proxy is one container per Docker daemon and every installation's port
# registry starts at the same number, so the second run took the first one's
# proxy and then failed to bind. It cost three runs in one afternoon, and the
# leftovers look like a defect in whatever was being tested rather than like a
# collision.
#
# madock-pro then moved into a VM of its own, which ends that collision: the
# tests run inside the guest, so 127.0.0.1:443 is the guest's own loopback and
# each VM has its own proxy. (Checked rather than assumed — on this machine the
# host's 443 belongs to OrbStack and neither suite goes near it.)
#
# The check still spans every running instance, for a smaller and duller reason:
# each VM is four CPUs and six gigabytes, and two of them installing a Magento
# at once on a laptop makes every timeout in the suite a lie. A run that fails
# because the machine was busy looks exactly like a run that found something.
#
# It looks for the test binary rather than for containers: containers left
# behind are a mess to clean up, but a running `e2e.test` is somebody's work in
# progress.
ensure_free() {
    running=""
    for instance in $(limactl list --quiet 2>/dev/null); do
        found=$(limactl shell "$instance" -- pgrep -af 'e2e\.test' 2>/dev/null || true)
        [ -n "$found" ] && running="$running
  $instance: $found"
    done
    [ -z "$running" ] && return 0

    [ "${MADOCK_E2E_IGNORE_BUSY:-}" = "yes" ] && {
        printf 'another suite is running and MADOCK_E2E_IGNORE_BUSY=yes — going ahead anyway\n' >&2
        return 0
    }

    die "another end-to-end suite is already running on this machine:
$running

Two suites at once share one laptop's CPUs and memory, and every timeout in
this suite then measures the contention rather than the code. Wait for it.
To insist anyway: MADOCK_E2E_IGNORE_BUSY=yes $0 run"
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

    # --platforms as a flag rather than an environment variable: an inline
    # assignment in front of a command is invisible to shell history, to a
    # transcript and to anything that matches commands by prefix, and this is
    # the one switch that changes a five-minute run into an hour-long one.
    platforms=""
    if [ "${1:-}" = "--platforms" ]; then
        platforms="yes"
        shift
    fi
    [ "${MADOCK_E2E_PLATFORMS:-}" = "yes" ] && platforms="yes"

    ensure_running
    ensure_free
    build_binary

    # -count=1 for the same reason the pre-push hook passes it: these tests are
    # driven by a binary and by docker, neither of which the build cache knows
    # anything about. A cached `ok` here would be worthless.
    #
    # MADOCK_E2E_BIN is how the tests find the binary. The repository is mounted
    # into the guest at its host path, read-only, so the binary built a moment
    # ago is already visible there.
    # Arguments reach the guest through `sh -c`, so they are quoted one by one
    # here. Without it `-run "A|B"` arrives as two commands and the shell tries
    # to execute the second one.
    #
    # 45m rather than the 30m this started with: the suite outgrew it. A whole
    # run on a freshly built VM also pulls every image, and the timeout is not
    # a budget — it is the point at which Go decides the run is hung and dumps
    # every goroutine, which is unreadable and says nothing about which test
    # was slow. run-local passes 40m for the same reason.
    quoted=""
    for arg in "$@"; do
        escaped=$(printf '%s' "$arg" | sed "s/'/'\\\\''/g")
        quoted="$quoted '$escaped'"
    done

    # A platform test installs a real store: a composer create-project, an
    # OpenSearch that has to come up, a setup:install. That is twenty minutes on
    # its own, so the whole run gets an hour and a half when they are asked for
    # and stays at 45m when they are not.
    timeout="45m"
    [ "$platforms" = "yes" ] && timeout="90m"

    limactl shell "$VM_NAME" -- env \
        MADOCK_E2E_BIN="$binary" \
        MADOCK_E2E_PLATFORMS="$platforms" \
        PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin" \
        sh -c "cd '$repo_root' && go test -tags=e2e -count=1 -v -timeout=$timeout ./test/e2e/...$quoted"
}

# cmd_run_local runs the suite against the Docker daemon of the machine it is
# invoked on. That is right on a CI runner, which is a disposable Linux box with
# a daemon nobody else is using, and wrong on a laptop, where the proxy stack it
# takes over is the one your own projects are behind.
#
# The guard is the CI variable every provider sets. It can be overridden, and
# the flag to do it is spelled out rather than short, because the failure mode is
# somebody's afternoon rather than a message.
cmd_run_local() {
    if [ "${CI:-}" != "true" ] && [ "${MADOCK_E2E_ALLOW_LOCAL_DOCKER:-}" != "yes" ]; then
        die "run-local takes over this machine's Docker daemon, including the proxy your own projects use.
On a CI runner that is fine and CI=true says so. Anywhere else use: $0 run
To insist anyway: MADOCK_E2E_ALLOW_LOCAL_DOCKER=yes $0 run-local"
    fi

    # Same switch as `run`, so the CI job that wants the platform tests asks for
    # them the same way a person does.
    platforms=""
    if [ "${1:-}" = "--platforms" ]; then
        platforms="yes"
        shift
    fi
    [ "${MADOCK_E2E_PLATFORMS:-}" = "yes" ] && platforms="yes"

    goarch=$(go env GOHOSTARCH)
    mkdir -p "$script_dir/.bin"
    printf 'building madock for linux/%s\n' "$goarch"
    ( cd "$repo_root" && GOOS=linux GOARCH="$goarch" go build -o "$binary" . )

    # 40m was the suite's own running time, not a margin. It passed at 33–36
    # minutes for months and then began being killed mid-test on a slower
    # runner, with `panic: test timed out` — which reads exactly like a hung
    # command and is nothing of the sort. The suite grows; the budget has to
    # leave room for that, and for a runner having a bad morning.
    timeout="60m"
    [ "$platforms" = "yes" ] && timeout="90m"

    MADOCK_E2E_BIN="$binary" MADOCK_E2E_PLATFORMS="$platforms" \
        go test -tags=e2e -count=1 -v -timeout="$timeout" ./test/e2e/... "$@"
}

# cmd_auth copies the invoking user's composer credentials into the guest.
#
# The platform tests install a real store, and Magento is downloaded from
# repo.magento.com, which refuses anonymous requests. madock symlinks
# ~/.composer into the project and bind-mounts it into the php container, so the
# credentials come from whichever machine runs madock — here the guest, whose
# home is its own and empty.
#
# Lima mounts the host home read-only at the same absolute path, so the file is
# already visible from inside; this only puts it where composer looks. Nothing
# leaves this laptop: the guest is local, the file is never committed, and CI
# has neither the mount nor the file, which is why the Magento test skips there
# instead of failing.
#
# Run it again after `reset` — a rebuilt VM has an empty home.
cmd_auth() {
    need_lima
    ensure_running

    host_auth="$HOME/.composer/auth.json"
    [ -f "$host_auth" ] || die "no composer credentials on this machine: $host_auth does not exist.
The platform tests need them to download Magento; everything else runs without."

    limactl shell "$VM_NAME" -- sh -c "mkdir -p ~/.composer && cp '$host_auth' ~/.composer/auth.json && chmod 600 ~/.composer/auth.json"
    printf 'copied %s into %s:~/.composer/auth.json\n' "$host_auth" "$VM_NAME"

    # Named, never printed: the point is to confirm which service the guest can
    # now reach, not to put a password in a terminal or a transcript.
    limactl shell "$VM_NAME" -- sh -c "grep -o '\"[a-z0-9.-]*\\.com\"' ~/.composer/auth.json | tr -d '\"' | sort -u | sed 's/^/  credentials for /'"
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
    up)        cmd_up ;;
    run)       cmd_run "$@" ;;
    run-local) cmd_run_local "$@" ;;
    auth)   cmd_auth ;;
    shell)  cmd_shell ;;
    down)   cmd_down ;;
    reset)  cmd_reset ;;
    status) cmd_status ;;
    *)
        sed -n '2,20p' "$0" | sed 's/^#\{1,2\} \{0,1\}//'
        exit 1
        ;;
esac
