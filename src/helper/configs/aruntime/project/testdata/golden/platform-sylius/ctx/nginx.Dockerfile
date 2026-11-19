FROM nginx:1.30.7

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u <UID> -o nginx && groupmod -g <GID> -o nginx

