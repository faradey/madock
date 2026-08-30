FROM node:22.14.0

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN apt-get update && apt-get install -y --no-install-recommends xdg-utils && rm -rf /var/lib/apt/lists/*
RUN npm install -g grunt-cli

RUN apt-get update && apt-get install -y cron && rm -rf /var/lib/apt/lists/*

RUN usermod -u <UID> -o node && groupmod -g <GID> -o node
RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data
RUN mkdir -p /var/www && chown <UID>:<GID> /var/www
WORKDIR /var/www/html


# madock: permissive umask (002) for cross-user file writes in dev — makes
# new files group-writable so root-created files don't block www-data and
# vice versa. Sourced by interactive shells, non-interactive bash -c
# (via BASH_ENV), and the container CMD wrapper. Toggle via
# permissions/umask/permissive=false in config for prod-like setups.
ENV BASH_ENV=/etc/madock-umask.sh
RUN printf 'umask 0002\n' > /etc/madock-umask.sh \
    && mkdir -p /etc/profile.d \
    && printf 'umask 0002\n' > /etc/profile.d/madock-umask.sh \
    && touch /etc/bash.bashrc \
    && printf '\numask 0002\n' >> /etc/bash.bashrc \
    && chmod 644 /etc/madock-umask.sh /etc/profile.d/madock-umask.sh

# madock smart entrypoint:
#   - if nodejs/script is set, run that (see nodejs/script_type)
#   - else pick from package.json: "dev" in development, "start" in production
#   - else fall back to plain `node` REPL
# Picks the package manager from lockfiles (yarn/pnpm/npm).
# Refuses to start the dev server when node_modules is missing —
# the user should run `madock install` first.
RUN cat > /usr/local/bin/madock-entrypoint <<'MADOCK_EOF' && chmod +x /usr/local/bin/madock-entrypoint
#!/bin/sh
set -e
umask 0002

cd "${WORKDIR:-/var/www/html}" 2>/dev/null || cd /var/www/html

# Start cron when this container has jobs and nothing else would.
#
# The same hole as the php image, and this is the side the reported failure was
# actually on: a Shopify or Medusa project runs its application here, not in a
# php container, and the production machine that came back from a reboot with no
# scheduler was running two Node apps. Fixing only php would have left the
# measured case exactly as it was — found by running it rather than by the test,
# which passes on an image the failure never touched.
#
# The condition is the crontab rather than the config, for the reason written out
# in the php footer: config is baked in at build time, jobs in the crontab are
# true when the container starts, and cron:disable removes them.
#
# Before anything else in this entrypoint, and never fatal: `set -e` is on, the
# steps below wait on code madock writes after start, and a scheduler that
# cannot start must not be what stops the application from serving.
if command -v crontab >/dev/null 2>&1; then
    if crontab -u www-data -l 2>/dev/null | grep -qE '^[^#[:space:]]' \
        || crontab -u root -l 2>/dev/null | grep -qE '^[^#[:space:]]'; then
        service cron start >/dev/null 2>&1 || true
    fi
fi

# run_app execs the long-running command as the application user.
#
# The entrypoint itself needs root: it waits for code that madock writes after
# the container starts, and reads files whose ownership is not settled yet. The
# dev server does not, and running it as root is how every file it writes ends
# up owned by root on the host — `.medusa/client/` was the one that made this
# visible, and the user could not delete their own project afterwards.
#
# This is the arrangement the php side has always had: the php-fpm master starts
# as root and its workers run as www-data, whose uid is remapped to the host
# user at build time. `node` is remapped the same way, so dropping to it here
# makes the two languages agree.
#
# HOME comes from passwd rather than being assumed: yarn and npm write caches
# and logs into it, and leaving root's HOME behind would send them somewhere the
# app user cannot write.
run_app() {
  if [ "$(id -u)" = "0" ] && id node >/dev/null 2>&1; then
    HOME=$(getent passwd node | cut -d: -f6)
    export HOME
    exec setpriv --reuid=node --regid=node --init-groups "$@"
  fi
  exec "$@"
}

# Wait for the project code to appear. madock setup -d clones the
# starter AFTER the container starts; rather than dropping to a Node
# REPL and never recovering, idle here until package.json shows up.
# A bare `node` container (no project) waits forever — that's fine,
# the user can `docker exec` in or run madock setup -d to populate.
if [ ! -f package.json ]; then
  echo "[madock] no package.json in $(pwd) — waiting for project code."
  while [ ! -f package.json ]; do
    sleep 5
  done
  echo "[madock] package.json detected — continuing."
fi

# has_script <name> — true when package.json declares that script.
#
# NODE_OPTIONS is cleared for it deliberately. This helper is a Node process
# like any other, so an inspector flag in the environment applies to it too —
# and with --inspect-brk it stops before its first line and waits for a debugger
# that is not coming. Its stderr goes to /dev/null, so the container would hang
# here with nothing said at all.
has_script() {
  NODE_OPTIONS= node -e 'try{var p=require("./package.json").scripts||{};process.exit(p[process.argv[1]]?0:1)}catch(e){process.exit(1)}' "$1" 2>/dev/null
}

# enable_inspector — put the debugger on the process this entrypoint is about to
# become. Only ever called where that process is the application itself.
enable_inspector() {
  flag="--inspect"
  if [ "${MADOCK_DEBUG_BREAK:-}" = "true" ]; then
    flag="--inspect-brk"
  fi

  # 0.0.0.0 and not node's default loopback: the IDE connects from outside the
  # container, where a debugger bound to 127.0.0.1 inside it is reachable by
  # nothing.
  NODE_OPTIONS="${NODE_OPTIONS:+$NODE_OPTIONS }$flag=0.0.0.0:${MADOCK_DEBUG_PORT}"
  export NODE_OPTIONS

  echo "[madock] debugger listening on 0.0.0.0:${MADOCK_DEBUG_PORT} — madock info:ports has the port on your side"
  if [ "${MADOCK_DEBUG_BREAK:-}" = "true" ]; then
    echo "[madock] nodejs/debug/break is on: nothing runs until a debugger attaches."
  fi
}

# explain_no_inspector — say why debugging was asked for and not arranged.
#
# Refusing to start would be worse: the container is otherwise fine, and the
# person asked for a debugger, not for the project to stop.
explain_no_inspector() {
  echo "[madock] debugging is enabled, and this project starts through $1, which is a Node program itself."
  echo "[madock] It would take the inspector port and the dev server would start without a debugger,"
  echo "[madock] so the IDE would attach to $1 rather than to your application. Nothing was changed."
  echo "[madock] Set nodejs/script_type=command with the command to run, or add --inspect to the script in package.json."
}

configured=""
configured_type="auto"
# madock's own, not NODE_ENV: the value decides which script to start and
# nothing else, and under the standard name every tool in the container obeyed
# it — `next build` built a production bundle in development mode. NODE_ENV is
# still honoured when a project sets it itself, so an existing project that
# relies on it keeps working.
node_env="${MADOCK_NODE_ENV:-${NODE_ENV:-development}}"

# mode is "package" (run through the project's package manager) or "command"
# (hand it to a shell). A path to a file is a command.
mode="package"

if [ -n "$configured" ]; then
  case "$configured_type" in
    package)
      mode="package"
      ;;
    command)
      mode="command"
      ;;
    *)
      # auto: a name package.json declares is a script, anything else is a
      # command. Said out loud, because the difference decides what runs and
      # a typo would otherwise become a shell command that fails at exec.
      if has_script "$configured"; then
        mode="package"
      else
        mode="command"
        echo "[madock] \"$configured\" is not a script in package.json — running it as a command."
      fi
      ;;
  esac
  script="$configured"
else
  # No script configured. In production "dev" is the wrong guess: for a
  # Shopify app it is `shopify app dev`, which prints a verification code and
  # waits for a human. On a server nobody logs in, it gives up, and the
  # container dies with it — start reported success minutes earlier.
  if [ "$node_env" = "production" ]; then
    if has_script start; then script="start"; elif has_script dev; then script="dev"; fi
  else
    if has_script dev; then script="dev"; elif has_script start; then script="start"; fi
  fi
fi

if [ -z "$script" ]; then
  # A bare REPL is the application here, so the inspector belongs on it.
  if [ -n "${MADOCK_DEBUG_PORT:-}" ]; then
    enable_inspector
  fi
  run_app node
fi

if [ "$mode" = "package" ] && ! has_script "$script"; then
  echo "[madock] package.json has no \"$script\" script (nodejs/script_type=package)."
  echo "[madock] Set nodejs/script to a script it declares, or nodejs/script_type=command to run it as a command."
  exit 1
fi

deps_installed() {
  # node_modules dir is created EARLY in yarn/npm install (during the
  # link step) — it's not a reliable "install finished" signal. Look
  # for a real completion marker instead:
  #   yarn 4: .yarn/install-state.gz written on successful install
  #   yarn 1: .yarn-integrity hash file in node_modules
  #   npm:    node_modules/.package-lock.json
  #   pnpm:   node_modules/.modules.yaml
  [ -d node_modules ] || return 1
  [ -f .yarn/install-state.gz ] && return 0
  [ -f node_modules/.yarn-integrity ] && return 0
  [ -f node_modules/.package-lock.json ] && return 0
  [ -f node_modules/.modules.yaml ] && return 0
  return 1
}

if ! deps_installed; then
  echo "[madock] node_modules not ready in $(pwd) — waiting."
  echo "[madock] Run \"madock install\" (or yarn install) to bootstrap; the dev server will start automatically once the install completes."
  while ! deps_installed; do
    sleep 5
  done
  echo "[madock] node_modules ready — starting dev server."
fi

if [ -f yarn.lock ]; then
  pm="yarn"
  pm_name="yarn"
elif [ -f pnpm-lock.yaml ]; then
  pm="pnpm"
  pm_name="pnpm"
else
  pm="npm run"
  # The name on its own, because $pm carries the subcommand: a message built
  # from it read "starts through npm run, which is a Node program itself".
  pm_name="npm"
fi

# Source .env right before exec so the dev server sees DATABASE_URL /
# REDIS_URL / SECRET_KEY etc. in process env. We can't source earlier
# because madock setup -d writes .env only during the install phase,
# which runs AFTER the container starts — at boot the file does not
# exist yet. By this point the install has completed (we already
# waited for its marker), so .env is guaranteed to be present.
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

if [ "$mode" = "command" ]; then
  # The command is the application, so the first Node process it starts is the
  # one worth attaching to.
  if [ -n "${MADOCK_DEBUG_PORT:-}" ]; then
    enable_inspector
  fi
  echo "[madock] starting: $script"
  run_app sh -c "$script"
fi

if [ -n "${MADOCK_DEBUG_PORT:-}" ]; then
  explain_no_inspector "$pm_name"
fi

echo "[madock] starting: $pm $script"
run_app $pm $script
MADOCK_EOF

ENV WORKDIR=/var/www/html

CMD ["/usr/local/bin/madock-entrypoint"]
