FROM node:{{{NODEJS_VERSION}}}

RUN apt-get update -y
RUN npm install -g grunt-cli && npm install -g bower

RUN mkdir -p /home/app \
    && usermod -d /home/app -l app node \
    && chown -R app /home/app

RUN groupmod -g {{{GUID}}} -o node
RUN usermod -u {{{UID}}} -o app
USER app