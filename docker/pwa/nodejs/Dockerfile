FROM {{{nodejs/repository}}}:{{{nodejs/version}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{os/user/uid}}} -o node && groupmod -g {{{os/user/guid}}} -o node
RUN usermod -u {{{os/user/uid}}} -o www-data && groupmod -g {{{os/user/guid}}} -o www-data

RUN mkdir /var/yarn && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/yarn
WORKDIR /var/yarn

RUN yarn set version {{{nodejs/yarn/version}}}

WORKDIR {{{workdir}}}

RUN chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www