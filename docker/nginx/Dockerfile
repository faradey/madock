FROM nginx:latest

#COPY conf/default.conf /etc/nginx/conf.d/default.conf
{{{COPY_NGINX_CONF}}}
ARG UID=33
ARG GUID=33
RUN usermod -u ${UID} -o nginx && groupmod -g ${GUID} -o nginx
