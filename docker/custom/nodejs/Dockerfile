FROM {{{nodejs/repository}}}:{{{nodejs/version}}}

RUN apt-get install -y gconf-service libasound2 libatk1.0-0 libc6 libcairo2 libcups2 libdbus-1-3 libexpat1 libfontconfig1 libgcc1 libgconf-2-4 libgdk-pixbuf2.0-0 libglib2.0-0 libgtk-3-0 libnspr4 libpango-1.0-0 libpangocairo-1.0-0 libstdc++6 libx11-6 libx11-xcb1 libxcb1 libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 libxss1 libxtst6 ca-certificates fonts-liberation libappindicator1 libnss3 lsb-release xdg-utils wget
RUN apt install chromium-bsu -y && npm install -g grunt-cli && npm install -g bower && npm install -g livereload

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog

RUN usermod -u {{{os/user/uid}}} -o node && groupmod -g {{{os/user/guid}}} -o node
RUN usermod -u {{{os/user/uid}}} -o www-data && groupmod -g {{{os/user/guid}}} -o www-data

RUN mkdir /var/yarn && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/yarn
WORKDIR /var/yarn

RUN yarn set version {{{nodejs/yarn/version}}}

WORKDIR {{{workdir}}}

RUN chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www
