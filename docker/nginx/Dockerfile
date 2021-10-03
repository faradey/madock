FROM docker2021repos/nginx:latest

COPY nginx.conf /etc/nginx/conf.d/default.conf

RUN usermod -u {{{UID}}} -o nginx && groupmod -g {{{GUID}}} -o nginx
