FROM {{{db/repository}}}:{{{db/version}}}

ENV MYSQL_ROOT_PASSWORD={{{db/root_password}}}
ENV MYSQL_DATABASE={{{db/database}}}
ENV MYSQL_USER={{{db/user}}}
ENV MYSQL_PASSWORD={{{db/password}}}
RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
RUN usermod -u {{{os/user/uid}}} -o mysql && groupmod -g {{{os/user/guid}}} -o mysql
