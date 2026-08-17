FROM ubuntu:22.04

ARG DEBIAN_FRONTEND="noninteractive"
ARG DEBCONF_NOWARNINGS="yes"

RUN ln -snf /usr/share/zoneinfo/UTC /etc/localtime && echo UTC > /etc/timezone \
    && apt-get clean && apt-get -y --allow-releaseinfo-change update && apt-get install -y locales \
    curl \
    wget \
    ca-certificates \
    software-properties-common \
    git \
    zip \
    gzip \
    mc \
    mariadb-client \
    telnet \
    libmagickwand-dev \
    imagemagick \
    libmcrypt-dev \
    procps \
    openssh-client \
    lsof \
    openssl \
    msmtp \
    xdg-utils \
    libssh2-1-dev \
    libssh2-1 \
    jq \
    && locale-gen en_US.UTF-8 \
    && LC_ALL=en_US.UTF-8 add-apt-repository ppa:ondrej/php

RUN apt-get -y --allow-releaseinfo-change update && apt-get install -y php8.4-bcmath \
    php8.4-cli \
    php8.4-common \
    php8.4-curl \
    php8.4-dev \
    php8.4-fpm \
    php8.4-gd \
    php8.4-intl \
    php8.4-mbstring \
    php8.4-mysql \
    php8.4-soap \
    php8.4-sqlite3 \
    php8.4-xml \
    php8.4-xsl \
    php8.4-zip \
    php8.4-imagick \
    php8.4-ctype \
    php8.4-dom \
    php8.4-fileinfo \
    php8.4-iconv \
    php8.4-simplexml \
    php8.4-sockets \
    php8.4-tokenizer \
    php8.4-xmlwriter \
    php8.4-ssh2 \
    php8.4-redis

# Optional packages: not all PHP versions ship them as separate packages
# (e.g. php8.5-opcache is bundled into php8.5-common, php-xmlrpc was dropped
# from PHP core in 8.0 and may be missing in newer ondrej builds). Install
# each in its own line so a missing package does not abort the build.
RUN apt-get install -y php8.4-opcache || true
RUN apt-get install -y php8.4-xmlrpc || true

SHELL ["/bin/bash", "-c"]
RUN IFS='.' read major minor patch <<< "8.4" \
    && if [[ "${major}" -ge "9" ]] || [[ "${major}" = "8" && "${minor}" -ge "4" ]]; then \
        # PHP 8.4+ — no compatible pecl mcrypt release, skip it
        apt-get install -y pkg-config libmcrypt-dev \
        && pecl channel-update pecl.php.net \
        && echo "Skipping pecl mcrypt for PHP ${major}.${minor} (no compatible release)" \
    ; elif [[ "${major}" > "7" || ("${major}" = "7" && "${minor}" > "1") ]]; then \
        pecl install mcrypt-1.0.7 \
        && EXTENSION_DIR="$( php -i | grep ^extension_dir | awk -F '=>' '{print $2}' | xargs )" \
        && bash -c "echo extension=${EXTENSION_DIR}/mcrypt.so > /etc/php/8.4/cli/conf.d/mcrypt.ini" \
        && bash -c "echo extension=${EXTENSION_DIR}/mcrypt.so > /etc/php/8.4/fpm/conf.d/mcrypt.ini" \
    ; fi \
    && if [[ "${major}" < "7" || ("${major}" = "7" && "${minor}" < "2") ]]; then \
        apt-get install -y php8.4-mcrypt \
    ; fi \
    && if [[ "${major}" < "7" ]]; then \
        apt-get install -y php8.4-json \
    ; fi

RUN sed -i -e "s/pid =.*/pid = \/var\/run\/php8.4-fpm.pid/" /etc/php/8.4/fpm/php-fpm.conf \
    && sed -i -e "s/error_log =.*/error_log = \/proc\/self\/fd\/2/" /etc/php/8.4/fpm/php-fpm.conf \
    && sed -i -e "s/;daemonize\s*=\s*yes/daemonize = no/g" /etc/php/8.4/fpm/php-fpm.conf \
    && sed -i "s/listen = .*/listen = 9000/" /etc/php/8.4/fpm/pool.d/www.conf \
    && sed -i "s/;catch_workers_output = .*/catch_workers_output = yes/" /etc/php/8.4/fpm/pool.d/www.conf

# Unlock ImageMagick PDF coder (default Debian/Ubuntu policy blocks PDF reads,
# which breaks Imagick-based PDF previews in PHP apps). Local dev only.
RUN if [ -f /etc/ImageMagick-6/policy.xml ]; then \
        sed -i 's#rights="none" pattern="PDF"#rights="read|write" pattern="PDF"#' /etc/ImageMagick-6/policy.xml; \
    fi


RUN if [[ "false" = "true" ]]; then set -eux && EXTENSION_DIR="$( php -i | grep ^extension_dir | awk -F '=>' '{print $2}' | xargs )" \
    && curl -o ioncube.tar.gz http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_<ARCH>.tar.gz \
    && tar xvfz ioncube.tar.gz \
    && cd ioncube \
    && cp ioncube_loader_lin_8.4.so ${EXTENSION_DIR}/ioncube.so \
    && cd ../ \
    && rm -rf ioncube \
    && rm -rf ioncube.tar.gz \
    && echo "zend_extension=ioncube.so" >> /etc/php/8.4/mods-available/ioncube.ini \
    && ln -s /etc/php/8.4/mods-available/ioncube.ini /etc/php/8.4/cli/conf.d/10-ioncube.ini \
    && ln -s /etc/php/8.4/mods-available/ioncube.ini /etc/php/8.4/fpm/conf.d/10-ioncube.ini; fi
RUN is_composer_version_one="" \
    && if [[ "2" = "2" ]]; then is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer; fi && if [[ "2" = "1" ]]; then  is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer && composer self-update --1; fi \
    && if [[ -z "${is_composer_version_one}" ]]; then php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer --version=2; fi
RUN if [[ "false" = "true" ]]; then pecl install -f xdebug-3.4.4 \
    && touch /etc/php/8.4/mods-available/xdebug.ini \
    && echo "zend_extension=xdebug.so" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.mode=debug" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.output_dir=/var/www/html/var" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.profiler_output_name=\"cachegrind.out.%t\"" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.remote_enable=1" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.start_with_request=on" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.remote_autostart=on" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.idekey=PHPSTORM" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.client_host=host.docker.internal" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.remote_host=host.docker.internal" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.remote_port=9003" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.client_port=9003" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.log=/var/www/var/log/xdebug.log" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.log_level=7" >> /etc/php/8.4/mods-available/xdebug.ini \
    && ln -s /etc/php/8.4/mods-available/xdebug.ini /etc/php/8.4/cli/conf.d/11-xdebug.ini \
    && ln -s /etc/php/8.4/mods-available/xdebug.ini /etc/php/8.4/fpm/conf.d/11-xdebug.ini; fi

RUN if [[ "false" = "true" && "debug" = "profile" ]]; then echo "xdebug.profiler_enable=1" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.profiler_output_dir=/var/www/html/var" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.xdebug.profiler_enable_trigger=0" >> /etc/php/8.4/mods-available/xdebug.ini \
    && echo "xdebug.profiler_append=0" >> /etc/php/8.4/mods-available/xdebug.ini; fi
RUN sed -i 's/session.cookie_lifetime = 0/session.cookie_lifetime = 2592000/g' /etc/php/8.4/fpm/php.ini \
    && sed -i 's/post_max_size = 8M/post_max_size = 80M/g' /etc/php/8.4/fpm/php.ini \
    && sed -i 's/upload_max_filesize = 2M/upload_max_filesize = 50M/g' /etc/php/8.4/fpm/php.ini \
    && sed -i 's/;max_input_vars = 1000/max_input_vars = 50000/g' /etc/php/8.4/fpm/php.ini

# Where PHP's mail() sends. Written only when there is somewhere for it to go.
#
# This used to be unconditional and hardcoded to the mailpit port, which meant
# that with mailpit disabled every mail() call handed the message to msmtp,
# which connected to a port nobody was listening on. Mail did not arrive and
# nothing said so — the setting looked right in php.ini, and the port was the
# only wrong part of it.
#
# php/sendmail/host and php/sendmail/port point somewhere else when they are set:
# a real relay on the host, another container, a smarthost. Editing php.ini
# inside a running container is not an alternative — the image is rebuilt from
# this file and the edit goes with it, which is exactly how a working mail
# configuration disappears at the next rebuild.
#
# php/sendmail/from is the envelope sender. msmtp refuses to send without one —
# `msmtp: envelope-from address is missing`, exit 78 — and there is no msmtprc
# here to supply a default. A mail transport passes it (PHP's mail() takes it as
# the fifth argument, `-f`), so Magento and Laravel are fine either way; a plain
# mail() call with four arguments is not, and that is what anyone testing their
# own site tries first. Setting this makes both work.

WORKDIR /var/www

RUN apt-get install -y cron
RUN mkdir /var/www/.ssh/ && mkdir /var/www/.composer/ && mkdir /var/www/scripts/ && mkdir /var/www/scripts/php && mkdir /var/www/patches/ && mkdir /var/www/var/ && mkdir /var/www/var/log/ && touch /var/www/var/log/xdebug.log && chmod 0777 /var/www/var/log/xdebug.log


RUN if [ "false" = "true" ]; then curl -sS https://accounts.magento.cloud/cli/installer | php \
    && cp -r /root/.magento-cloud/ /var/www/ && chown -R <UID>:<GID> /var/www/.magento-cloud && ln -s /var/www/.magento-cloud/bin/magento-cloud /usr/bin/magento-cloud; fi
RUN if [ "false" = "true" ]; then chown <UID>:<GID> /usr/bin/magento-cloud; fi

RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data \
    && chown -R <UID>:<GID> /var/www \
    && chown -R <UID>:<GID> /usr/bin/composer
WORKDIR /var/www/html

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && rm -f /var/log/faillog && rm -f /var/log/lastlog

EXPOSE 9001 9003 35729 5173 9998 9999


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
CMD ["bash", "-c", "exec php-fpm8.4"]
