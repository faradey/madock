FROM node:22.19.0

RUN npm install -g @anthropic-ai/claude-code

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
RUN npm install -g grunt-cli

RUN usermod -u <UID> -o node && groupmod -g <GID> -o node
RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data

WORKDIR /var/www/html

RUN chown <UID>:<GID> /var/www

CMD ["node"]