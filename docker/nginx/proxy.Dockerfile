FROM nginx:latest

COPY ./proxy.conf /etc/nginx/conf.d/default.conf
