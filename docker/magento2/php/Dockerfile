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
    php{{{PHP_VERSION}}}-xmlwriter

RUN if [ "{{{PHP_VERSION}}}" < "8.0" ]; then apt-get install -y php{{{PHP_VERSION}}}-json; fi

RUN sed -i -e "s/pid =.*/pid = \/var\/run\/php{{{PHP_VERSION}}}-fpm.pid/" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i -e "s/error_log =.*/error_log = \/proc\/self\/fd\/2/" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i -e "s/;daemonize\s*=\s*yes/daemonize = no/g" /etc/php/{{{PHP_VERSION}}}/fpm/php-fpm.conf \
    && sed -i "s/listen = .*/listen = 9000/" /etc/php/{{{PHP_VERSION}}}/fpm/pool.d/www.conf \
    && sed -i "s/;catch_workers_output = .*/catch_workers_output = yes/" /etc/php/{{{PHP_VERSION}}}/fpm/pool.d/www.conf

RUN if [ "{{{IONCUBE_ENABLED}}}" = "true" ]; then set -eux && EXTENSION_DIR="$( php -i | grep ^extension_dir | awk -F '=>' '{print $2}' | xargs )" \
    && curl -o ioncube.tar.gz http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_{{{OSARCH}}}.tar.gz \
    && tar xvfz ioncube.tar.gz \
    && cd ioncube \
    && cp ioncube_loader_lin_{{{PHP_VERSION}}}.so ${EXTENSION_DIR}/ioncube.so \
    && cd ../ \
    && rm -rf ioncube \
    && rm -rf ioncube.tar.gz \
    && echo "zend_extension=ioncube.so" >> /etc/php/{{{PHP_VERSION}}}/mods-available/ioncube.ini \
    && ln -s /etc/php/{{{PHP_VERSION}}}/mods-available/ioncube.ini /etc/php/{{{PHP_VERSION}}}/cli/conf.d/10-ioncube.ini \
    && ln -s /etc/php/{{{PHP_VERSION}}}/mods-available/ioncube.ini /etc/php/{{{PHP_VERSION}}}/fpm/conf.d/10-ioncube.ini; fi

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

RUN if [ "{{{MAGENTOCLOUD_ENABLED}}}" = "true" ]; then curl -sS https://accounts.magento.cloud/cli/installer | php \
    && cp -r /root/.magento-cloud/ /var/www/ && chown -R {{{UID}}}:{{{GUID}}} /var/www/.magento-cloud && ln -s /var/www/.magento-cloud/bin/magento-cloud /usr/bin/magento-cloud; fi

RUN apt-get install cron
RUN mkdir /var/www/.ssh/ && mkdir /var/www/.composer/ && mkdir /var/www/n98magerun/ && mkdir /var/www/scripts/ && mkdir /var/www/scripts/php && mkdir /var/www/patches/ && mkdir /var/www/var/ && mkdir /var/www/var/log/ && touch /var/www/var/log/xdebug.log && chmod 0777 /var/www/var/log/xdebug.log
RUN if [ "{{{NODEJS_ENABLED}}}" = "true" ]; then mkdir /var/www/.npm && chown {{{UID}}}:{{{GUID}}} /var/www/.npm && apt install nodejs npm -y && apt install chromium-bsu -y && npm install -g grunt-cli && npm install -g bower && npm install -g livereload; fi
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o www-data && groupmod -g {{{GUID}}} -o www-data
RUN chown -R {{{UID}}}:{{{GUID}}} /var/www/.ssh
RUN chown {{{UID}}}:{{{GUID}}} /var/www/patches
RUN chown {{{UID}}}:{{{GUID}}} /var/www/scripts/php
RUN chown {{{UID}}}:{{{GUID}}} /var/www/.composer
RUN chown {{{UID}}}:{{{GUID}}} /usr/bin/composer
RUN chown {{{UID}}}:{{{GUID}}} /var/www/n98magerun
RUN if [ "{{{MAGENTOCLOUD_ENABLED}}}" = "true" ]; then chown {{{UID}}}:{{{GUID}}} /usr/bin/magento-cloud; fi
RUN if [ "{{{N98MAGERUN_ENABLED}}}" = "true" ]; then cd /var/www/n98magerun && curl -sS -O https://files.magerun.net/n98-magerun2.phar && chown {{{UID}}}:{{{GUID}}} /var/www/n98magerun/n98-magerun2.phar && chmod +x /var/www/n98magerun/n98-magerun2.phar; fi
WORKDIR {{{WORKDIR}}}

EXPOSE 9001 9003 35729
CMD "php-fpm{{{PHP_VERSION}}}"
