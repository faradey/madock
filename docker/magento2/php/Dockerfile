{{{include snippets/dockerfile/php/header}}}
{{{include snippets/dockerfile/php/ioncube}}}
{{{include snippets/dockerfile/php/composer}}}
{{{include snippets/dockerfile/php/xdebug}}}
{{{include snippets/dockerfile/php/ini}}}
{{{include snippets/dockerfile/php/cron}}}
{{{include snippets/dockerfile/php/mkdir}}}
{{{include snippets/dockerfile/php/nodejs}}}
RUN if [ "{{{magento/cloud/enabled}}}" = "true" ]; then curl -sS https://accounts.magento.cloud/cli/installer | php \
    && cp -r /root/.magento-cloud/ /var/www/ && chown -R {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/.magento-cloud && ln -s /var/www/.magento-cloud/bin/magento-cloud /usr/bin/magento-cloud; fi
RUN if [ "{{{magento/cloud/enabled}}}" = "true" ]; then chown {{{os/user/uid}}}:{{{os/user/guid}}} /usr/bin/magento-cloud; fi

<<<if{{{magento/n98magerun/enabled}}}>>>
RUN mkdir /var/www/n98magerun/ \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/n98magerun \
    && cd /var/www/n98magerun \
    && IFS='.' read major minor patch <<< "{{{php/version}}}" && n98downfilename="n98-magerun2.phar" \
    && if [[ "${major}" = "7" && "${minor}" = "3" ]]; then n98downfilename="n98-magerun2-6.1.1.phar"; fi \
    && if [[ "${major}" = "7" && "${minor}" = "2" ]]; then n98downfilename="n98-magerun2-4.7.0.phar"; fi \
    && if [[ "${major}" < "7" || ("${major}" = "7" && "${minor}" < "2") ]]; then n98downfilename="n98-magerun2-2.3.3.phar"; fi \
    && curl -sS -o ./n98-magerun2.phar https://files.magerun.net/${n98downfilename} \
    && chown {{{os/user/uid}}}:{{{os/user/guid}}} /var/www/n98magerun/n98-magerun2.phar \
    && chmod +x /var/www/n98magerun/n98-magerun2.phar
<<<endif>>>

{{{include snippets/dockerfile/php/chown}}}
{{{include snippets/dockerfile/php/footer}}}