FROM nginx:1.21.4

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o nginx && groupmod -g {{{GUID}}} -o nginx
