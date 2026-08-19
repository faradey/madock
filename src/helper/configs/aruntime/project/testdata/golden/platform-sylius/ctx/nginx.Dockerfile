FROM nginx:1.28

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u <UID> -o nginx && groupmod -g <GID> -o nginx

