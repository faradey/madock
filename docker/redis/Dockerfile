FROM redis:{{{REDIS_VERSION}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o redis && groupmod -g {{{GUID}}} -o redis
