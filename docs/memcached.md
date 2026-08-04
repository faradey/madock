**Memcached**

A plain in-memory cache server, off by default. Enable it when the application
expects Memcached specifically — Magento session storage, a legacy PrestaShop or
WooCommerce cache backend, a framework whose driver list has no Redis.

For anything new, prefer Redis or Valkey: they cover the same ground, madock
already ships them, and they persist. Memcached is here for the code that asks
for it by name.

To enable the service execute the command:
```
madock service:enable memcached
```
To disable it:
```
madock service:disable memcached
```

Both commands rebuild the project. Enabling it also rebuilds the PHP image,
because the `php-memcached` extension is installed only while the service is on.

Configuration
* Host: `memcached`
* Port: `11211`

The port is not published to the host — the container is reachable from the
other containers of the project by the service name.

**Version and memory**

```
madock config:set --name memcached/repository --value memcached
```
```
madock config:set --name memcached/version --value 1.6.39-alpine
```
Cache size in megabytes (`-m`), 256 by default:
```
madock config:set --name memcached/memory --value 512
```
Maximum simultaneous connections (`-c`), 1024 by default:
```
madock config:set --name memcached/max_connections --value 2048
```

`service:enable memcached` also accepts the version directly, skipping the
interactive picker:
```
madock service:enable memcached --service-version 1.6.39-alpine
```

After any configuration change, rebuild the project:
```
madock rebuild
```

**PHP extension**

`php{version}-memcached` is installed into the PHP image while the service is
enabled, and left out otherwise. The install is tolerant of a missing package —
ondrej's repository lags a release or two behind for the newest PHP versions, and
a hard failure there would break the image build for everyone. If `php -m` does
not list `memcached` after a rebuild, that package is not published for your PHP
version yet; pin an older PHP version or wait for the repository to catch up.

Note that PHP has two separate extensions: `memcached` (libmemcached-based, what
madock installs, what Magento uses) and the older `memcache`. They are not
interchangeable.

**Magento example**

Session storage in `app/etc/env.php`:
```php
'session' => [
    'save' => 'memcached',
    'save_path' => 'memcached:11211',
],
```

**Isolation mode**

With `isolation/enabled` the container joins the `isolated` network alongside the
rest of the stack, so nothing outside the project can reach it. See
[Isolation mode](isolation.md).
