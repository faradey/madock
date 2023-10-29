FROM ubuntu:{{{UBUNTU_VERSION}}}

ARG DEBIAN_FRONTEND="noninteractive"
ARG DEBCONF_NOWARNINGS="yes"

RUN ln -snf /usr/share/zoneinfo/{{{PHP_TZ}}} /etc/localtime && echo {{{PHP_TZ}}} > /etc/timezone

RUN apt-get clean && apt-get -y update && apt-get install -y locales \
    curl \
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
    libssh2-1-dev \
    libssh2-1 \
    wget \
    && locale-gen en_US.UTF-8 \
    && LC_ALL=en_US.UTF-8 add-apt-repository ppa:ondrej/php

RUN apt-get update && apt-get install -y php{{{PHP_VERSION}}}-bcmath \
    php{{{PHP_VERSION}}}-cli \
    php{{{PHP_VERSION}}}-common \
    php{{{PHP_VERSION}}}-curl \
    php{{{PHP_VERSION}}}-dev \
    php{{{PHP_VERSION}}}-fpm \
    php{{{PHP_VERSION}}}-gd \
    php{{{PHP_VERSION}}}-intl \
    php{{{PHP_VERSION}}}-mbstring \
    php{{{PHP_VERSION}}}-mysql \
    php{{{PHP_VERSION}}}-opcache \
    php{{{PHP_VERSION}}}-soap \
    php{{{PHP_VERSION}}}-sqlite3 \
    php{{{PHP_VERSION}}}-xml \
    php{{{PHP_VERSION}}}-xmlrpc \
    php{{{PHP_VERSION}}}-xsl \
    php{{{PHP_VERSION}}}-zip \
    php{{{PHP_VERSION}}}-imagick \
    php{{{PHP_VERSION}}}-ctype \
    php{{{PHP_VERSION}}}-dom \
    php{{{PHP_VERSION}}}-fileinfo \
    php{{{PHP_VERSION}}}-iconv \
    php{{{PHP_VERSION}}}-simplexml \
    php{{{PHP_VERSION}}}-sockets \
    php{{{PHP_VERSION}}}-tokenizer \
    php{{{PHP_VERSION}}}-xmlwriter \
    php{{{PHP_VERSION}}}-ssh2

RUN if [ "{{{PHP_VERSION}}}" < "8.0" ]; then apt-get install -y php{{{PHP_VERSION}}}-json; fi

RUN sed -i -e "s/pid =.*/pid = \/var\/run\/php{{{PHP_VERSION}}}-fpm.pid/" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i -e "s/error_log =.*/error_log = \/proc\/self\/fd\/2/" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i -e "s/;daemonize\s*=\s*yes/daemonize = no/g" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i "s/listen = .*/listen = 9000/" /etc/php/{{{PHP_VERSION}}}/fpm/pool.d/www.conf \
    && sed -i "s/;catch_workers_output = .*/catch_workers_output = yes/" /etc/php/{{{PHP_VERSION}}}/fpm/pool.d/www.conf

RUN is_composer_version_one="" \
    && if [ "{{{PHP_COMPOSER_VERSION}}}" = "2" ]; then is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer; fi && if [ "{{{PHP_COMPOSER_VERSION}}}" = "1" ]; then  is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer && composer self-update --1; fi \
    && if [ -z "${is_composer_version_one}" ]; then php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer --version={{{PHP_COMPOSER_VERSION}}}; fi

RUN if [ "{{{XDEBUG_ENABLED}}}" = "true" ]; then pecl install -f xdebug-{{{XDEBUG_VERSION}}} \
    && touch /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "zend_extension=xdebug.so" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.mode={{{XDEBUG_MODE}}}" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.output_dir={{{WORKDIR}}}/var" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.profiler_output_name=cachegrind.out.%t" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_enable=1" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.start_with_request=on" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_autostart=on" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.idekey={{{XDEBUG_IDE_KEY}}}" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.client_host={{{XDEBUG_REMOTE_HOST}}}" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_host={{{XDEBUG_REMOTE_HOST}}}" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_port=9003" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.client_port=9003" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.log=/var/www/var/log/xdebug.log" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && echo "xdebug.log_level=7" >> /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini \
    && ln -s /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini /etc/php/{{{PHP_VERSION}}}/cli/conf.d/11-xdebug.ini \
    && ln -s /etc/php/{{{PHP_VERSION}}}/mods-available/xdebug.ini /etc/php/{{{PHP_VERSION}}}/fpm/conf.d/11-xdebug.ini; fi

RUN sed -i 's/session.cookie_lifetime = 0/session.cookie_lifetime = 2592000/g' /etc/php/{{{PHP_VERSION}}}/fpm/php.ini \
    && sed -i 's/post_max_size = 8M/post_max_size = 80M/g' /etc/php/{{{PHP_VERSION}}}/fpm/php.ini \
    && sed -i 's/upload_max_filesize = 2M/upload_max_filesize = 50M/g' /etc/php/{{{PHP_VERSION}}}/fpm/php.ini \
    && sed -i 's/;max_input_vars = 1000/max_input_vars = 50000/g' /etc/php/{{{PHP_VERSION}}}/fpm/php.ini \
    && sed -i 's/;sendmail_path =/sendmail_path = "\/usr\/bin\/msmtp -t --port=1025 --host=host.docker.internal"/g' /etc/php/{{{PHP_VERSION}}}/fpm/php.ini

WORKDIR /var/www

RUN apt-get install cron
RUN curl -sL https://deb.nodesource.com/setup_{{{NODEJS_MAJOR_VERSION}}}.x | bash -
RUN mkdir /var/www/.ssh/ && mkdir /var/www/.composer/ && mkdir /var/www/scripts/ && mkdir /var/www/scripts/php && mkdir /var/www/patches/ && mkdir /var/www/var/ && mkdir /var/www/var/log/ && touch /var/www/var/log/xdebug.log && chmod 0777 /var/www/var/log/xdebug.log
RUN mkdir /var/www/.npm && chown {{{UID}}}:{{{GUID}}} /var/www/.npm && apt install nodejs -y
ENV YARN_VERSION=1.22.19

RUN set -ex && for key in 6A010C5166006599AA17F08146C2130DFD2497F5 ; do gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys "$key" || gpg --batch --keyserver keyserver.ubuntu.com --recv-keys "$key" ; done && curl -fsSLO --compressed "https://yarnpkg.com/downloads/$YARN_VERSION/yarn-v$YARN_VERSION.tar.gz" && curl -fsSLO --compressed "https://yarnpkg.com/downloads/$YARN_VERSION/yarn-v$YARN_VERSION.tar.gz.asc" && gpg --batch --verify yarn-v$YARN_VERSION.tar.gz.asc yarn-v$YARN_VERSION.tar.gz && mkdir -p /opt && tar -xzf yarn-v$YARN_VERSION.tar.gz -C /opt/ && ln -s /opt/yarn-v$YARN_VERSION/bin/yarn /usr/local/bin/yarn && ln -s /opt/yarn-v$YARN_VERSION/bin/yarnpkg /usr/local/bin/yarnpkg && rm yarn-v$YARN_VERSION.tar.gz.asc yarn-v$YARN_VERSION.tar.gz && yarn --version
RUN yarn set version {{{YARN_VERSION}}}

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o www-data && groupmod -g {{{GUID}}} -o www-data \
    && chown -R {{{UID}}}:{{{GUID}}} /var/www \
    && chown -R {{{UID}}}:{{{GUID}}} /var/www/.ssh \
    && chown {{{UID}}}:{{{GUID}}} /var/www/patches \
    && chown {{{UID}}}:{{{GUID}}} /var/www/scripts/php \
    && chown {{{UID}}}:{{{GUID}}} /var/www/.composer \
    && chown {{{UID}}}:{{{GUID}}} /usr/bin/composer

WORKDIR {{{WORKDIR}}}

EXPOSE 9001 9003 35729
CMD "php-fpm{{{PHP_VERSION}}}"
