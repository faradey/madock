FROM docker2021repos/nginx:latest

COPY ./proxy.conf /etc/nginx/conf.d/default.conf
