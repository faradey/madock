FROM python:{{{python/version}}}-slim

# System deps: build toolchain for native wheels (psycopg, Pillow, etc),
# postgres client for `manage.py dbshell` and migrations probes, and the
# things that make uv + django-cron-style usage work.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    libssl-dev \
    libffi-dev \
    libjpeg-dev \
    zlib1g-dev \
    libwebp-dev \
    libmagic1 \
    gettext \
    git \
    curl \
    ca-certificates \
    postgresql-client \
    cron \
 && rm -rf /var/lib/apt/lists/*

# uv — the package manager Saleor 3.x uses (replaces poetry).
RUN pip install --no-cache-dir uv

# Match host UID/GID so bind-mounted files stay editable on host and
# container as the same user. The base image ships no `node`/`www-data`
# user we can re-use, so create a dedicated `saleor` user.
ARG MADOCK_UID={{{os/user/uid}}}
ARG MADOCK_GID={{{os/user/guid}}}
RUN groupadd -g ${MADOCK_GID} saleor 2>/dev/null || true \
 && useradd -u ${MADOCK_UID} -g ${MADOCK_GID} -ms /bin/bash saleor 2>/dev/null || true \
 && mkdir -p /var/www && chown ${MADOCK_UID}:${MADOCK_GID} /var/www

ENV WORKDIR={{{workdir}}}
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
WORKDIR {{{workdir}}}

# Smart entrypoint: prefer uv-managed `manage.py runserver`, fall back to
# uvicorn for ASGI, fall back to a bash shell if no recognisable Python
# project is present. Refuses to start when deps are not installed yet.
# This RUN must execute as root (root can write to /usr/local/bin); the
# USER directive moves below so the container runs as the saleor user.
RUN cat > /usr/local/bin/madock-entrypoint <<'MADOCK_EOF' && chmod +x /usr/local/bin/madock-entrypoint
#!/bin/sh
set -e

cd "${WORKDIR:-/var/www/html}" 2>/dev/null || cd /var/www/html

# Saleor (and most Django projects) read configuration from process env
# rather than from .env directly. Source the .env file when present so
# DATABASE_URL / REDIS_URL / SECRET_KEY end up in the environment.
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

# Detect if this looks like a Saleor / Django project.
has_manage=0
[ -f manage.py ] && has_manage=1

has_pyproject=0
[ -f pyproject.toml ] && has_pyproject=1

# Project not detected — drop to a shell so the user can poke around.
if [ "$has_manage" -eq 0 ] && [ "$has_pyproject" -eq 0 ]; then
  echo "[madock] no manage.py / pyproject.toml found in $(pwd) — sleeping; nothing to run."
  exec sleep infinity
fi

# Dependencies must be present. uv installs into .venv by default; pip
# users have site-packages or a venv. We probe for either.
# Prefer uv if a uv.lock / pyproject is present. Otherwise plain python.
# Picking the prefix BEFORE the deps probe so that `python -c "import …"`
# below runs through `uv run` and actually sees the .venv-installed
# packages rather than the bare system interpreter.
RUN_PREFIX=""
if [ -f uv.lock ] || [ -f pyproject.toml ]; then
  if command -v uv >/dev/null 2>&1; then
    RUN_PREFIX="uv run"
  fi
fi

deps_ready() {
  [ -d .venv ] && return 0
  $RUN_PREFIX python -c 'import saleor' 2>/dev/null && return 0
  $RUN_PREFIX python -c 'import django' 2>/dev/null && return 0
  return 1
}

if ! deps_ready; then
  echo "[madock] Python deps missing in $(pwd) — waiting."
  echo "[madock] Run \"madock install\" to bootstrap (uv sync + migrate + populatedb); the server will start automatically once deps are installed."
  while ! deps_ready; do
    sleep 5
  done
  echo "[madock] Python deps detected — starting server."
fi

# Run the Saleor / Django dev server. Saleor 3.x always ships
# saleor/asgi.py; if that file is present we go through uvicorn (the
# upstream default), otherwise we fall back to Django's runserver.
if [ "$has_manage" -eq 1 ] && [ -f saleor/asgi.py ]; then
  echo "[madock] starting: uvicorn saleor.asgi:application on 0.0.0.0:8000"
  exec $RUN_PREFIX uvicorn saleor.asgi:application --host 0.0.0.0 --port 8000 --reload --lifespan auto --ws none
fi

if [ "$has_manage" -eq 1 ]; then
  echo "[madock] starting: python manage.py runserver 0.0.0.0:8000"
  exec $RUN_PREFIX python manage.py runserver 0.0.0.0:8000
fi

echo "[madock] pyproject.toml present but no manage.py / asgi entrypoint detected — sleeping."
exec sleep infinity
MADOCK_EOF

USER saleor

CMD ["/usr/local/bin/madock-entrypoint"]
