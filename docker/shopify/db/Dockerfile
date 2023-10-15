FROM {{{DB_REPOSITORY}}}:{{{DB_VERSION}}}

ENV MYSQL_ROOT_PASSWORD={{{DB_ROOT_PASSWORD}}}
ENV MYSQL_DATABASE={{{DB_DATABASE}}}
ENV MYSQL_USER={{{DB_USER}}}
ENV MYSQL_PASSWORD={{{DB_PASSWORD}}}
RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
RUN usermod -u {{{UID}}} -o mysql && groupmod -g {{{GUID}}} -o mysql
