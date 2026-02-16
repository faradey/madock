FROM {{{os/name}}}:{{{os/version}}}

ARG DEBIAN_FRONTEND="noninteractive"
ARG DEBCONF_NOWARNINGS="yes"

RUN ln -snf /usr/share/zoneinfo/{{{php/timezone}}} /etc/localtime && echo {{{php/timezone}}} > /etc/timezone \
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
    ruby-full \
    build-essential \
    && locale-gen en_US.UTF-8

RUN gem install bundler

RUN apt-get install -y cron

RUN mkdir -p /var/www/.ssh/ && mkdir -p /var/www/scripts/ && mkdir -p /var/www/var/ && mkdir -p /var/www/var/log/

RUN usermod -u {{{os/user/uid}}} -o www-data && groupmod -g {{{os/user/guid}}} -o www-data \
    && chown -R {{{os/user/uid}}}:{{{os/user/guid}}} /var/www

WORKDIR {{{workdir}}}

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && rm -f /var/log/faillog && rm -f /var/log/lastlog

EXPOSE 3000 4567

CMD ["bash"]
