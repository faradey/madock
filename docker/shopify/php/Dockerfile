{{{include snippets/dockerfile/php/header}}}
{{{include snippets/dockerfile/php/ioncube}}}
{{{include snippets/dockerfile/php/composer}}}
{{{include snippets/dockerfile/php/xdebug}}}
{{{include snippets/dockerfile/php/ini}}}
{{{include snippets/dockerfile/php/cron}}}
{{{include snippets/dockerfile/php/mkdir}}}
RUN curl -sL https://deb.nodesource.com/setup_{{{nodejs/major_version}}}.x | bash -
RUN mkdir /var/www/.npm && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.npm && apt install nodejs -y
ENV YARN_VERSION=1.22.19

RUN apt-get install xdg-utils --yes

RUN set -ex && for key in 6A010C5166006599AA17F08146C2130DFD2497F5 ; do gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys "$key" || gpg --batch --keyserver keyserver.ubuntu.com --recv-keys "$key" ; done && curl -fsSLO --compressed "https://yarnpkg.com/downloads/$YARN_VERSION/yarn-v$YARN_VERSION.tar.gz" && curl -fsSLO --compressed "https://yarnpkg.com/downloads/$YARN_VERSION/yarn-v$YARN_VERSION.tar.gz.asc" && gpg --batch --verify yarn-v$YARN_VERSION.tar.gz.asc yarn-v$YARN_VERSION.tar.gz && mkdir -p /opt && tar -xzf yarn-v$YARN_VERSION.tar.gz -C /opt/ && ln -s /opt/yarn-v$YARN_VERSION/bin/yarn /usr/local/bin/yarn && ln -s /opt/yarn-v$YARN_VERSION/bin/yarnpkg /usr/local/bin/yarnpkg && rm yarn-v$YARN_VERSION.tar.gz.asc yarn-v$YARN_VERSION.tar.gz && yarn --version
RUN yarn set version {{{nodejs/yarn/version}}}

{{{include snippets/dockerfile/php/chown}}}
{{{include snippets/dockerfile/php/footer}}}
