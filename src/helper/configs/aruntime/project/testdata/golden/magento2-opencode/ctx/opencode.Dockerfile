FROM node:22.19.0

RUN apt-get update && apt-get install -y --no-install-recommends openssh-client \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g opencode-ai

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u <UID> -o node && groupmod -g <GID> -o node
RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data

# The project's tools live in opencode-exec. madock-exec runs a command there,
# in the directory it was called from: both containers mount the code at the
# same path, so the directory exists on the other side too.
RUN mkdir -p /etc/ssh/ssh_config.d && cat > /etc/ssh/ssh_config.d/madock-opencode.conf <<'MADOCK_EOF'
Host opencode-exec
    User opencode
    IdentityFile /etc/madock-opencode/id_ed25519
    IdentitiesOnly yes
    UserKnownHostsFile /etc/madock-opencode/known_hosts
    StrictHostKeyChecking yes
    LogLevel ERROR
MADOCK_EOF
RUN cat > /usr/local/bin/madock-exec <<'MADOCK_EOF' && chmod 755 /usr/local/bin/madock-exec
#!/bin/bash
if [ $# -eq 0 ]; then
    echo "usage: madock-exec <command> [args...]" >&2
    exit 64
fi
dir=$PWD
case "$dir" in
    /var/www/html|/var/www/html/*) ;;
    *) dir=/var/www/html ;;
esac
exec ssh -T opencode-exec "cd $(printf '%q' "$dir") && exec $(printf '%q ' "$@")"
MADOCK_EOF
# One shim, linked under every tool's name: it runs the name it was called by.
# bin/magento and other scripts need no shim of their own — their shebang
# finds php here.
RUN printf '#!/bin/bash\nexec /usr/local/bin/madock-exec "$(basename "$0")" "$@"\n' > /usr/local/bin/madock-exec-shim \
    && chmod 755 /usr/local/bin/madock-exec-shim \
    && for tool in php composer go gofmt python python3 pip pip3 ruby bundle rake; do ln -sf madock-exec-shim /usr/local/bin/$tool; done

WORKDIR /var/www/html

RUN chown <UID>:<GID> /var/www \
    && mkdir -p /var/www/.local/share/opencode \
    && chown -R <UID>:<GID> /var/www/.local

CMD ["sleep", "infinity"]
