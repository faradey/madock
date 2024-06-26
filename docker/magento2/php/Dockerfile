FROM {{{os/name}}}:{{{os/version}}}

ARG DEBIAN_FRONTEND="noninteractive"
ARG DEBCONF_NOWARNINGS="yes"

RUN ln -snf /usr/share/zoneinfo/{{{php/timezone}}} /etc/localtime && echo {{{php/timezone}}} > /etc/timezone

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
    && locale-gen en_US.UTF-8 \
    && LC_ALL=en_US.UTF-8 add-apt-repository ppa:ondrej/php

RUN apt-get update && apt-get install -y php{{{php/version}}}-bcmath \
    php{{{php/version}}}-cli \
    php{{{php/version}}}-common \
    php{{{php/version}}}-curl \
    php{{{php/version}}}-dev \
    php{{{php/version}}}-fpm \
    php{{{php/version}}}-gd \
    php{{{php/version}}}-intl \
    php{{{php/version}}}-mbstring \
    php{{{php/version}}}-mysql \
    php{{{php/version}}}-opcache \
    php{{{php/version}}}-soap \
    php{{{php/version}}}-sqlite3 \
    php{{{php/version}}}-xml \
    php{{{php/version}}}-xmlrpc \
    php{{{php/version}}}-xsl \
    php{{{php/version}}}-zip \
    php{{{php/version}}}-imagick \
    php{{{php/version}}}-ctype \
    php{{{php/version}}}-dom \
    php{{{php/version}}}-fileinfo \
    php{{{php/version}}}-iconv \
    php{{{php/version}}}-simplexml \
    php{{{php/version}}}-sockets \
    php{{{php/version}}}-tokenizer \
    php{{{php/version}}}-xmlwriter \
    php{{{php/version}}}-ssh2

SHELL ["/bin/bash", "-c"]
RUN IFS='.' read major minor patch <<< "{{{php/version}}}" \
    && if [[ "${major}" > "7" || ("${major}" = "7" && "${minor}" > "1") ]]; then pecl install mcrypt-1.0.7 \
    && EXTENSION_DIR="$( php -i | grep ^extension_dir | awk -F '=>' '{print $2}' | xargs )" \
    && bash -c "echo extension=${EXTENSION_DIR}/mcrypt.so > /etc/php/{{{php/version}}}/cli/conf.d/mcrypt.ini" \
    && bash -c "echo extension=${EXTENSION_DIR}/mcrypt.so > /etc/php/{{{php/version}}}/fpm/conf.d/mcrypt.ini"; fi \
    && if [[ "${major}" < "7" || ("${major}" = "7" && "${minor}" < "2") ]]; then apt-get install -y php{{{php/version}}}-mcrypt; fi \
    && if [[ "${major}" < "7" ]]; then apt-get install -y php{{{php/version}}}-json; fi

RUN sed -i -e "s/pid =.*/pid = \/var\/run\/php{{{php/version}}}-fpm.pid/" /etc/php/{{{php/version}}}/fpm/php-fpm.conf \
    && sed -i -e "s/error_log =.*/error_log = \/proc\/self\/fd\/2/" /etc/php/{{{php/version}}}/fpm/php-fpm.conf \
    && sed -i -e "s/;daemonize\s*=\s*yes/daemonize = no/g" /etc/php/{{{php/version}}}/fpm/php-fpm.conf \
    && sed -i "s/listen = .*/listen = 9000/" /etc/php/{{{php/version}}}/fpm/pool.d/www.conf \
    && sed -i "s/;catch_workers_output = .*/catch_workers_output = yes/" /etc/php/{{{php/version}}}/fpm/pool.d/www.conf

RUN if [[ "{{{php/ioncube/enabled}}}" = "true" ]]; then set -eux && EXTENSION_DIR="$( php -i | grep ^extension_dir | awk -F '=>' '{print $2}' | xargs )" \
    && curl -o ioncube.tar.gz http://downloads3.ioncube.com/loader_downloads/ioncube_loaders_lin_{{{os/arch}}}.tar.gz \
    && tar xvfz ioncube.tar.gz \
    && cd ioncube \
    && cp ioncube_loader_lin_{{{php/version}}}.so ${EXTENSION_DIR}/ioncube.so \
    && cd ../ \
    && rm -rf ioncube \
    && rm -rf ioncube.tar.gz \
    && echo "zend_extension=ioncube.so" >> /etc/php/{{{php/version}}}/mods-available/ioncube.ini \
    && ln -s /etc/php/{{{php/version}}}/mods-available/ioncube.ini /etc/php/{{{php/version}}}/cli/conf.d/10-ioncube.ini \
    && ln -s /etc/php/{{{php/version}}}/mods-available/ioncube.ini /etc/php/{{{php/version}}}/fpm/conf.d/10-ioncube.ini; fi

RUN is_composer_version_one="" \
    && if [[ "{{{php/composer/version}}}" = "2" ]]; then is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer; fi && if [[ "{{{php/composer/version}}}" = "1" ]]; then  is_composer_version_one="1" && php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer && composer self-update --1; fi \
    && if [[ -z "${is_composer_version_one}" ]]; then php -r "readfile('http://getcomposer.org/installer');" | php -- --install-dir=/usr/bin/ --filename=composer --version={{{php/composer/version}}}; fi

RUN if [[ "{{{php/xdebug/enabled}}}" = "true" ]]; then pecl install -f xdebug-{{{php/xdebug/version}}} \
    && touch /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "zend_extension=xdebug.so" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.mode={{{php/xdebug/mode}}}" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.output_dir={{{workdir}}}/var" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.profiler_output_name=\"cachegrind.out.%t\"" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_enable=1" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.start_with_request=on" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_autostart=on" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.idekey={{{php/xdebug/ide_key}}}" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.client_host={{{php/xdebug/remote_host}}}" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_host={{{php/xdebug/remote_host}}}" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.remote_port=9003" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.client_port=9003" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.log=/var/www/var/log/xdebug.log" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.log_level=7" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && ln -s /etc/php/{{{php/version}}}/mods-available/xdebug.ini /etc/php/{{{php/version}}}/cli/conf.d/11-xdebug.ini \
    && ln -s /etc/php/{{{php/version}}}/mods-available/xdebug.ini /etc/php/{{{php/version}}}/fpm/conf.d/11-xdebug.ini; fi

RUN if [[ "{{{php/xdebug/enabled}}}" = "true" && "{{{php/xdebug/mode}}}" = "profile" ]]; then echo "xdebug.profiler_enable=1" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.profiler_output_dir={{{workdir}}}/var" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.xdebug.profiler_enable_trigger=0" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini \
    && echo "xdebug.profiler_append=0" >> /etc/php/{{{php/version}}}/mods-available/xdebug.ini; fi

RUN sed -i 's/session.cookie_lifetime = 0/session.cookie_lifetime = 2592000/g' /etc/php/{{{php/version}}}/fpm/php.ini \
    && sed -i 's/post_max_size = 8M/post_max_size = 80M/g' /etc/php/{{{php/version}}}/fpm/php.ini \
    && sed -i 's/upload_max_filesize = 2M/upload_max_filesize = 50M/g' /etc/php/{{{php/version}}}/fpm/php.ini \
    && sed -i 's/;max_input_vars = 1000/max_input_vars = 50000/g' /etc/php/{{{php/version}}}/fpm/php.ini \
    && sed -i 's/;sendmail_path =/sendmail_path = "\/usr\/bin\/msmtp -t --port=1025 --host=host.docker.internal"/g' /etc/php/{{{php/version}}}/fpm/php.ini \
    && sed -i 's/;sendmail_path =/sendmail_path = "\/usr\/bin\/msmtp -t --port=1025 --host=host.docker.internal"/g' /etc/php/{{{php/version}}}/cli/php.ini

WORKDIR /var/www

RUN if [ "{{{magento/cloud/enabled}}}" = "true" ]; then curl -sS https://accounts.magento.cloud/cli/installer | php \
    && cp -r /root/.magento-cloud/ /var/www/ && chown -R {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.magento-cloud && ln -s /var/www/.magento-cloud/bin/magento-cloud /usr/bin/magento-cloud; fi

RUN apt-get install cron \
    && mkdir /var/www/.ssh/ && mkdir /var/www/.composer/ && mkdir /var/www/scripts/ && mkdir /var/www/scripts/php && mkdir /var/www/patches/ && mkdir /var/www/var/ && mkdir /var/www/var/log/ && touch /var/www/var/log/xdebug.log && chmod 0777 /var/www/var/log/xdebug.log
RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{os/user/uid}}} -o www-data && groupmod -g {{{os/user/guid}}} -o www-data \
    && chown -R {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.ssh \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/patches \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/scripts/php \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.composer \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /usr/bin/composer
RUN if [ "{{{magento/cloud/enabled}}}" = "true" ]; then chown {{{os/user/uid}}}:{{{os/user/guid}}} /usr/bin/magento-cloud; fi

<<<if{{{magento/n98magerun/enabled}}}>>>
RUN mkdir /var/www/n98magerun/ \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/n98magerun \
    && cd /var/www/n98magerun \
    && IFS='.' read major minor patch <<< "{{{php/version}}}" && n98downfilename="n98-magerun2.phar" \
    && if [[ "${major}" = "7" && "${minor}" = "3" ]]; then n98downfilename="n98-magerun2-6.1.1.phar"; fi \
    && if [[ "${major}" = "7" && "${minor}" = "2" ]]; then n98downfilename="n98-magerun2-4.7.0.phar"; fi \
    && if [[ "${major}" < "7" || ("${major}" = "7" && "${minor}" < "2") ]]; then n98downfilename="n98-magerun2-2.3.3.phar"; fi \
    && curl -sS -o ./n98-magerun2.phar https://files.magerun.net/${n98downfilename} \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/n98magerun/n98-magerun2.phar \
    && chmod +x /var/www/n98magerun/n98-magerun2.phar
<<<endif>>>

<<<if{{{php/nodejs/enabled}}}>>>RUN curl -sL https://deb.nodesource.com/setup_{{{nodejs/major_version}}}.x | bash -<<<endif>>>
<<<if{{{php/nodejs/enabled}}}>>>RUN mkdir /var/www/.npm && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.npm && apt install nodejs -y<<<endif>>>


WORKDIR {{{workdir}}}
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

EXPOSE 9001 9003 35729
CMD "php-fpm{{{php/version}}}"
