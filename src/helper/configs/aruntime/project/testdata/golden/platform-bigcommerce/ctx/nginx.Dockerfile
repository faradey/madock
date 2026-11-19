FROM nginx:1.30.7

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u <UID> -o nginx && groupmod -g <GID> -o nginx

# No version in the Server header or on nginx's own error pages. The shared
# proxy hides this header, but an error page's body passes through it.
RUN echo 'server_tokens off;' > /etc/nginx/conf.d/00-server-tokens.conf

