FROM node:{{{NODEJS_VERSION}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{UID}}} -o node && groupmod -g {{{GUID}}} -o node

RUN npm install --global yarn

