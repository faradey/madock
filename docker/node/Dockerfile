FROM node:{{{NODEJS_VERSION}}}

RUN apt update -y
RUN apt install chromium -y
RUN npm install -g grunt-cli && npm install -g bower

RUN mkdir -p /home/app \
    && usermod -d /home/app -l app node \
    && chown -R app /home/app

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN groupmod -g {{{GUID}}} -o node
RUN usermod -u {{{UID}}} -o app
USER app