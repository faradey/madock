FROM ubuntu:22.04

ARG DEBIAN_FRONTEND="noninteractive"
ARG DEBCONF_NOWARNINGS="yes"

RUN ln -snf /usr/share/zoneinfo/UTC /etc/localtime && echo UTC > /etc/timezone \
    && apt-get clean && apt-get -y --allow-releaseinfo-change update && apt-get install -y \
    locales \
    curl \
    wget \
    ca-certificates \
    software-properties-common \
    git \
    zip \
    gzip \
    mc \
    mariadb-client \
    procps \
    openssh-client \
    lsof \
    openssl \
    jq \
    build-essential \
    && locale-gen en_US.UTF-8
RUN apt-get install -y \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev

RUN set -eux; \
    curl -fsSL https://deb.nodesource.com/setup_18.x -o /tmp/nodesource_setup.sh; \
    bash /tmp/nodesource_setup.sh; \
    rm -f /tmp/nodesource_setup.sh; \
    apt-get -y --allow-releaseinfo-change update; \
    apt-get install -y nodejs; \
    command -v npm >/dev/null 2>&1 || apt-get install -y npm; \
    node -v; npm -v
RUN mkdir -p /var/www/.npm && chown <UID>:<GID> /var/www/.npm
RUN npm install -g grunt-cli

RUN apt-get install -y cron
RUN mkdir -p /var/www/.ssh/ && mkdir -p /var/www/scripts/ && mkdir -p /var/www/var/ && mkdir -p /var/www/var/log/
RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data \
    && chown -R <UID>:<GID> /var/www
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && rm -f /var/log/faillog && rm -f /var/log/lastlog
WORKDIR /var/www/html

EXPOSE 8000 5000 3000


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

CMD ["bash"]