FROM node:{{{NODEJS_VERSION}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o node && groupmod -g {{{GUID}}} -o node
RUN usermod -u {{{UID}}} -o www-data && groupmod -g {{{GUID}}} -o www-data

RUN yarn set version {{{YARN_VERSION}}}

WORKDIR {{{WORKDIR}}}

RUN chown {{{UID}}}:{{{GUID}}} /var/www