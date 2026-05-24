FROM ruby:{{{ruby/version}}}-slim

# System deps: build toolchain for native gems (pg, nokogiri, bcrypt,
# etc.), PostgreSQL client for `rails db:*` and migrations probes,
# libvips for Active Storage image processing, node/yarn for asset
# pipelines that still rely on them, plus the usual operational tools.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    libssl-dev \
    libffi-dev \
    libyaml-dev \
    libjpeg-dev \
    libpng-dev \
    libwebp-dev \
    libvips42 \
    libmagic1 \
    zlib1g-dev \
    gettext \
    git \
    curl \
    ca-certificates \
    postgresql-client \
    sqlite3 \
    cron \
 && rm -rf /var/lib/apt/lists/*

# Bundler 4.x matches the Gemfile.lock that current Spree starter
# releases ship with. Older Ruby bases (3.3+) bundle 2.5.x by default.
RUN gem install bundler --no-document

# Match host UID/GID so bind-mounted files stay editable on host and
# container as the same user. The base image ships no `node`/`www-data`
# user we can re-use, so create a dedicated `ruby` user.
ARG MADOCK_UID={{{os/user/uid}}}
ARG MADOCK_GID={{{os/user/guid}}}
RUN groupadd -g ${MADOCK_GID} ruby 2>/dev/null || true \
 && useradd -u ${MADOCK_UID} -g ${MADOCK_GID} -ms /bin/bash ruby 2>/dev/null || true \
 && mkdir -p /var/www && chown ${MADOCK_UID}:${MADOCK_GID} /var/www

ENV WORKDIR={{{workdir}}}
ENV BUNDLE_PATH=/var/www/html/vendor/bundle
ENV BUNDLE_BIN=/var/www/html/vendor/bundle/bin
ENV PATH=/var/www/html/vendor/bundle/bin:/var/www/html/bin:$PATH
WORKDIR {{{workdir}}}

# Smart entrypoint: waits for the project clone, then for `bundle
# install` to populate vendor/bundle, then exec's the Rails dev server.
# Same pattern as the Medusa/Saleor entrypoints.
RUN cat > /usr/local/bin/madock-entrypoint <<'MADOCK_EOF' && chmod +x /usr/local/bin/madock-entrypoint
#!/bin/sh
set -e

cd "${WORKDIR:-/var/www/html}" 2>/dev/null || cd /var/www/html

# Wait for the project code to appear. madock setup -d clones the
# starter AFTER the container starts; rather than idling forever the
# moment the container boots, poll for project markers and continue
# the moment they show up.
if [ ! -f Gemfile ]; then
  echo "[madock] no Gemfile in $(pwd) — waiting for project code."
  while [ ! -f Gemfile ]; do
    sleep 5
  done
  echo "[madock] project files detected — continuing."
fi

deps_ready() {
  # `bundle check` is the only reliable signal that gem deps are fully
  # installed for the current Gemfile.lock. Checking for vendor/bundle
  # alone is not enough because the dir is created mid-install.
  bundle check >/dev/null 2>&1
}

if ! deps_ready; then
  echo "[madock] gem deps not ready in $(pwd) — waiting."
  echo "[madock] Run \"madock install\" (or bundle install) to bootstrap; the dev server will start automatically once the install completes."
  while ! deps_ready; do
    sleep 5
  done
  echo "[madock] gem deps ready — starting dev server."
fi

# Source .env right before exec so the Rails dev server sees
# DATABASE_URL / REDIS_URL / SECRET_KEY_BASE etc. in process env. We
# can't source earlier because madock setup -d writes .env only during
# the install phase, which runs AFTER the container starts.
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

# Rails leaves a tmp/pids/server.pid behind if the previous container
# was killed mid-boot. `rails server` then refuses to start with
# "A server is already running". Best-effort cleanup before exec.
rm -f tmp/pids/server.pid 2>/dev/null || true

if [ -x bin/rails ]; then
  echo "[madock] starting: bin/rails server -b 0.0.0.0 -p 3000"
  exec bin/rails server -b 0.0.0.0 -p 3000
fi

if command -v bundle >/dev/null 2>&1 && [ -f Gemfile ]; then
  echo "[madock] starting: bundle exec rails server -b 0.0.0.0 -p 3000"
  exec bundle exec rails server -b 0.0.0.0 -p 3000
fi

echo "[madock] Gemfile present but no rails entrypoint detected — sleeping."
exec sleep infinity
MADOCK_EOF

USER ruby

CMD ["/usr/local/bin/madock-entrypoint"]
