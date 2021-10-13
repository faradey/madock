FROM node:latest

RUN npm install -g grunt-cli && npm install -g bower

RUN usermod -u {{{UID}}} -o node && groupmod -g {{{GUID}}} -o node
USER node
WORKDIR /var/www/html