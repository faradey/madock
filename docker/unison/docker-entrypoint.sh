#!/usr/bin/env bash
set -eo pipefail

log_heading() {
  echo ""
  echo "==> $*"
}

log_info() {
  echo "-----> $*"
}

log_error_exit() {
  echo " !  Error:"
  echo " !     $*"
  echo " !     Aborting!"
  exit 1
}

log_heading "Configuration:"
log_info "UNISON_USER: $UNISON_USER"
log_info "UNISON_GROUP: $UNISON_GROUP"
log_info "UNISON_UID: $UNISON_UID"
log_info "UNISON_GID: $UNISON_GID"
log_info "SYNC_SOURCE_BASE_PATH: $SYNC_SOURCE_BASE_PATH"
log_info "SYNC_DESTINATION_BASE_PATH: $SYNC_DESTINATION_BASE_PATH"
log_info "SYNC_PREFER: $SYNC_PREFER"
log_info "SYNC_SILENT: $SYNC_SILENT"
log_info "SYNC_MAX_INOTIFY_WATCHES: $SYNC_MAX_INOTIFY_WATCHES"
log_info "SYNC_EXTRA_UNISON_PROFILE_OPTS: $SYNC_EXTRA_UNISON_PROFILE_OPTS"
log_info "SYNC_NODELETE_SOURCE: $SYNC_NODELETE_SOURCE"

log_heading "Creating user ${UNISON_USER}."
if [[ ! $(getent group "${UNISON_GROUP}") ]]; then
    log_info "Group name ${UNISON_GROUP}"
    addgroup -g ${UNISON_GID} -S ${UNISON_GROUP}
fi
if [[ ! $(getent passwd "${UNISON_USER}") ]]; then
    log_info "User name ${UNISON_USER}"
    adduser -u ${UNISON_UID} -D -S -G ${UNISON_GROUP} ${UNISON_USER}
fi
log_info "Setting up ~/.unison dir"
HOME=$(eval echo ~${UNISON_USER})
mkdir -p ${HOME}/.unison
chown -R ${UNISON_USER}:${UNISON_GROUP} ${HOME}

log_heading "Checking sync directories."
log_info "SYNC_SOURCE_BASE_PATH: $SYNC_SOURCE_BASE_PATH"
log_info "SYNC_DESTINATION_BASE_PATH: $SYNC_DESTINATION_BASE_PATH"

[[ -d "$SYNC_SOURCE_BASE_PATH" ]] || log_error_exit "Source directory does not exist!"
[[ -d "$SYNC_DESTINATION_BASE_PATH" ]] || log_error_exit "Destination directory does not exist!"
[[ "$SYNC_SOURCE_BASE_PATH" != "$SYNC_DESTINATION_BASE_PATH" ]] || log_error_exit "Source and destination must be different directories!"

if [[ -n "${SYNC_MAX_INOTIFY_WATCHES}" ]]; then
  if [[ -z "$(sysctl -p)" ]]; then
    echo fs.inotify.max_user_watches=$SYNC_MAX_INOTIFY_WATCHES | tee -a /etc/sysctl.conf && sysctl -p
  else
    log_info "Looks like /etc/sysctl.conf already has fs.inotify.max_user_watches defined."
    log_info "Skipping this step."
  fi
fi

prefer="prefer=newer"
if [[ -z "${SYNC_PREFER}" ]]; then
  prefer="prefer=${SYNC_PREFER}"
fi

silent="silent=false"
if [[ "$SYNC_SILENT" == "1" ]]; then
  silent="silent=true"
fi

nodeletion=""
if [[ "$SYNC_NODELETE_SOURCE" == "1" ]]; then
  nodeletion="nodeletion=${SYNC_SOURCE_BASE_PATH}"
fi

echo "
# Sync roots
root = $SYNC_SOURCE_BASE_PATH
root = $SYNC_DESTINATION_BASE_PATH

# Sync options
auto=true
backups=false
batch=true
contactquietly=true
fastcheck=true
maxthreads=10
$prefer
$silent
$nodeletion

# Files to ignore
ignore = Path .git/*
ignore = Path .idea/*
ignore = Name *___jb_tmp___*
ignore = Name {.*,*}.sw[pon]

# Additional user configuration
$SYNC_EXTRA_UNISON_PROFILE_OPTS

" > ${HOME}/.unison/common

echo "
include common

" > ${HOME}/.unison/sync

echo "
repeat=watch
include common

" > ${HOME}/.unison/watch

exec "$@"