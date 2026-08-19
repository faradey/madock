FROM redis:7.2

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u <UID> -o redis && groupmod -g <GID> -o redis

