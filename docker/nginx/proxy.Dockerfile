FROM nginx:1.21.3

COPY ./proxy.conf /etc/nginx/conf.d/default.conf
