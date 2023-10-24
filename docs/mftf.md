With `madock` you can use MFTF tests.

Enable MFTF tests for the project.
```
madock service:enable mftf
```
Set up the dev/tests/acceptance/.env file
```
MAGENTO_BASE_URL=https://my.site.test/
MAGENTO_BACKEND_NAME=admin
MAGENTO_ADMIN_USERNAME=admin
SELENIUM_CLOSE_ALL_SESSIONS=true
BROWSER=chrome
MODULE_ALLOWLIST=Magento_Framework,ConfigurableProductWishlist,ConfigurableProductCatalogSearch
WAIT_TIMEOUT=60
BROWSER_LOG_BLOCKLIST=other
ELASTICSEARCH_VERSION=7
MAGENTO_ADMIN_PASSWORD=admin123
SELENIUM_HOST=selenium
SELENIUM_PORT=4444
SELENIUM_PROTOCOL=http
SELENIUM_PATH=/
```

Set up the dev/tests/acceptance/.credentials file (only required parameters are indicated here, you can add or uncomment the rest as needed)
```
magento/tfa/OTP_SHARED_SECRET=MFZWIZTHNBVGW3D2
magento/MAGENTO_ADMIN_PASSWORD=admin123
```

Init MFTF configuration
```
madock mftf:init
```

Generate MFTF tests
```
madock mftf generate:tests 
```

Run MFTF test by name(s)
```
madock mftf run:test AdminLoginSuccessfulTest StorefrontPersistedCustomerLoginTest -r
```
or run for group
```
madock mftf run:group product -r
```

(Optional) To see what is happening inside the container, head to
```
https://my.site.test/mftf-selenium/?autoconnect=1&resize=scale&password=secret
```