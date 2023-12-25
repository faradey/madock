FROM nginx:1.21.4

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{os/user/uid}}} -o nginx && groupmod -g {{{os/user/guid}}} -o nginx
