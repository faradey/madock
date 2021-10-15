FROM node:latest

RUN apt update -y
RUN npm install -g grunt-cli && npm install -g bower

RUN apt clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN usermod -u {{{UID}}} -o node && groupmod -g {{{GUID}}} -o node
RUN mkdir -p /home/app \
    && usermod -d /home/app node \
    && chown -R {{{UID}}}  /home/app
USER node
WORKDIR /var/www/html