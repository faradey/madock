FROM node:latest

RUN apt-get update -y
RUN npm install -g grunt-cli && npm install -g bower

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN usermod -u {{{UID}}} -o node && groupmod -g {{{GUID}}} -o node
WORKDIR /var/www/html
