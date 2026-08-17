{{{template "snippets/dockerfile/php/header" .}}}
{{{template "snippets/dockerfile/php/ioncube" .}}}
{{{template "snippets/dockerfile/php/composer" .}}}
{{{template "snippets/dockerfile/php/xdebug" .}}}
{{{template "snippets/dockerfile/php/ini" .}}}
{{{template "snippets/dockerfile/php/cron" .}}}
{{{template "snippets/dockerfile/php/mkdir" .}}}
{{{template "snippets/dockerfile/common/nodejs" .}}}
RUN if [ "{{{.magento.cloud.enabled}}}" = "true" ]; then curl -sS https://accounts.magento.cloud/cli/installer | php \
    && cp -r /root/.magento-cloud/ /var/www/ && chown -R {{{.os.user.uid}}}:{{{.os.user.guid}}} /var/www/.magento-cloud && ln -s /var/www/.magento-cloud/bin/magento-cloud /usr/bin/magento-cloud; fi
RUN if [ "{{{.magento.cloud.enabled}}}" = "true" ]; then chown {{{.os.user.uid}}}:{{{.os.user.guid}}} /usr/bin/magento-cloud; fi

{{{- if .magento.n98magerun.enabled}}}
RUN mkdir /var/www/n98magerun/ \
    && chown {{{.os.user.uid}}}:{{{.os.user.guid}}} /var/www/n98magerun \
    && cd /var/www/n98magerun \
    && IFS='.' read major minor patch <<< "{{{.php.version}}}" && n98downfilename="n98-magerun2.phar" \
    && if [[ "${major}" = "7" && "${minor}" = "3" ]]; then n98downfilename="n98-magerun2-6.1.1.phar"; fi \
    && if [[ "${major}" = "7" && "${minor}" = "2" ]]; then n98downfilename="n98-magerun2-4.7.0.phar"; fi \
    && if [[ "${major}" < "7" || ("${major}" = "7" && "${minor}" < "2") ]]; then n98downfilename="n98-magerun2-2.3.3.phar"; fi \
    && curl -sS -o ./n98-magerun2.phar https://files.magerun.net/${n98downfilename} \
    && chown {{{.os.user.uid}}}:{{{.os.user.guid}}} /var/www/n98magerun/n98-magerun2.phar \
    && chmod +x /var/www/n98magerun/n98-magerun2.phar
{{{- end}}}

{{{template "snippets/dockerfile/php/chown" .}}}
{{{template "snippets/dockerfile/php/footer" .}}}