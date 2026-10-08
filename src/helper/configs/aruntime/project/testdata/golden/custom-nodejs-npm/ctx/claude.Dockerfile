FROM node:22.19.0

RUN npm install -g @anthropic-ai/claude-code

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
# npm packages the project adds: nodejs/npm/extra. All at once first; when that
# fails, one by one, for the reason given in common/packages-extra — a rebuild
# that stops halfway on a server leaves the site down.
RUN { npm install -g pnpm tsx \
        || for p in pnpm tsx; do npm install -g "$p" || echo "madock: npm package $p could not be installed"; done; } \
    && npm cache clean --force

RUN usermod -u <UID> -o node && groupmod -g <GID> -o node
RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data

WORKDIR /var/www/html

RUN chown <UID>:<GID> /var/www

CMD ["node"]