FROM mariadb:11.4


RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
RUN usermod -u <UID> -o mysql && groupmod -g <GID> -o mysql
