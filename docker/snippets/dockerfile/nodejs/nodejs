FROM {{{nodejs/repository}}}:{{{nodejs/version}}}

RUN rm -f /var/log/faillog && rm -f /var/log/lastlog
RUN npm install -g grunt-cli

RUN usermod -u {{{os/user/uid}}} -o node && groupmod -g {{{os/user/guid}}} -o node
RUN usermod -u {{{os/user/uid}}} -o www-data && groupmod -g {{{os/user/guid}}} -o www-data

WORKDIR {{{workdir}}}

RUN chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www

CMD ["node"]