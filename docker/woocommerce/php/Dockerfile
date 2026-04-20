{{{include snippets/dockerfile/php/header}}}
{{{include snippets/dockerfile/php/ioncube}}}
{{{include snippets/dockerfile/php/composer}}}
{{{include snippets/dockerfile/php/xdebug}}}
{{{include snippets/dockerfile/php/ini}}}
{{{include snippets/dockerfile/php/cron}}}
{{{include snippets/dockerfile/php/mkdir}}}
RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar \
    && chmod +x wp-cli.phar \
    && mv wp-cli.phar /usr/local/bin/wp
{{{include snippets/dockerfile/php/chown}}}
{{{include snippets/dockerfile/php/nodejs}}}
{{{include snippets/dockerfile/php/footer}}}