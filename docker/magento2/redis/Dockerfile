FROM {{{redis/repository}}}:{{{redis/version}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{os/user/uid}}} -o redis && groupmod -g {{{os/user/guid}}} -o redis
