{{{template "snippets/dockerfile/php/header" .}}}
{{{template "snippets/dockerfile/php/ioncube" .}}}
{{{template "snippets/dockerfile/php/composer" .}}}
{{{template "snippets/dockerfile/php/xdebug" .}}}
{{{template "snippets/dockerfile/php/ini" .}}}
{{{template "snippets/dockerfile/php/cron" .}}}
{{{template "snippets/dockerfile/php/mkdir" .}}}
RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar \
    && chmod +x wp-cli.phar \
    && mv wp-cli.phar /usr/local/bin/wp
{{{template "snippets/dockerfile/php/chown" .}}}
{{{template "snippets/dockerfile/php/nodejs" .}}}
{{{template "snippets/dockerfile/php/footer" .}}}