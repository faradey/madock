## Database

madock supports multiple database engines: **MariaDB**, **MySQL**, **PostgreSQL**, and **MongoDB**.

### Choosing a database engine

During `madock setup`, you'll be prompted to select a database engine and version:

1. **MariaDB** (default) — versions 11.4, 11.1, 10.6, 10.4, 10.3, 10.2
2. **MySQL** — versions 9.2, 9.1, 8.4, 8.0
3. **PostgreSQL** — versions 17, 16, 15, 14, 13
4. **MongoDB** — versions 8.0, 7.0, 6.0, 5.0

### Configuration

The database type is stored in `config.xml`:

```xml
<db>
    <type>mysql</type>          <!-- mysql, postgresql, or mongodb -->
    <repository>mariadb</repository>
    <version>10.6</version>
    ...
</db>
```

If `db/type` is not set, madock auto-detects the type from `db/repository`:
- `mariadb`, `mysql` → mysql
- `postgres`, `postgresql` → postgresql
- `mongo`, `mongodb` → mongodb

### Switching from MariaDB to MySQL manually

By default madock uses MariaDB. To switch to MySQL, edit `config.xml` in your project directory (`madock/projects/{project_name}/config.xml`):

```xml
<db>
    <type>mysql</type>
    <repository>mysql</repository>
    <version>8.4</version>
    ...
</db>
```

Then rebuild the containers:
```
madock rebuild
```

The same approach works for any engine — just set `type`, `repository`, and `version` to the desired values:

| Engine | type | repository | Example versions |
|---|---|---|---|
| MariaDB | `mysql` | `mariadb` | 11.4, 10.6, 10.4 |
| MySQL | `mysql` | `mysql` | 9.2, 8.4, 8.0 |
| PostgreSQL | `postgresql` | `postgres` | 17, 16, 15, 14 |
| MongoDB | `mongodb` | `mongo` | 8.0, 7.0, 6.0 |

### Commands

All database commands work automatically based on the configured engine:

| Command | MySQL/MariaDB | PostgreSQL | MongoDB |
|---|---|---|---|
| `madock db:export` | `mysqldump` / `mariadb-dump` | `pg_dump` | `mongodump --archive --gzip` |
| `madock db:import` | `mysql` / `mariadb` | `psql` | `mongorestore --archive --gzip` |
| `madock db:info` | Shows all credentials + root password | Shows credentials (no root password) | Shows credentials (no root password) |
| `madock remote:sync:db` | Remote `db:export` (if madock installed) or remote mysqldump via SSH | Remote `db:export` or remote pg_dump via SSH | Remote `db:export` or remote mongodump via SSH |

### Admin UI services

Each database engine has a corresponding admin UI:

| Engine | Admin UI | Config key | Enable command |
|---|---|---|---|
| MySQL/MariaDB | phpMyAdmin | `db/phpmyadmin` | `madock service:enable phpmyadmin` |
| PostgreSQL | pgAdmin | `db/pgadmin` | `madock service:enable pgadmin` |
| MongoDB | Mongo Express | `db/mongo_express` | `madock service:enable mongo_express` |

pgAdmin config keys in `config.xml`:
```xml
<db>
    <pgadmin>
        <enabled>false</enabled>
        <repository>dpage/pgadmin4</repository>
        <version>latest</version>
        <email>admin@admin.com</email>
        <password>admin</password>
    </pgadmin>
</db>
```

### Importing and exporting

With `madock` you can import the database from your dev site.

First, add ssh data in the project file `madock/projects/{projects_name}/config.xml`

List of ssh options:
* SSH_HOST
* SSH_PORT
* SSH_USERNAME
* SSH_KEY_PATH
* SSH_SITE_ROOT_PATH
* SSH_AUTH_TYPE (key or password)
* SSH_PASSWORD (enter only if you do not use a ssh key)
* See [examples](./ssh_example.md)

Run the following commands:
```
madock remote:sync:db
```
```
madock db:import
```
After the successful execution of these commands, the database will be imported.

`remote:sync:db` resolves the remote database in one of three ways, in order:

1. **Explicit credentials** — if you pass `--db-host`, `--db-port`, `--db-user`, `--db-password` and `--db-name`, they are used directly and the dump is created on the remote host with `mysqldump`/`pg_dump`/`mongodump`.
2. **Remote madock** — if `madock` is installed on the remote host, the dump is produced **inside the remote container** via `madock db:export` (run from `ssh/site_root_path`), then downloaded. No PHP or database client is required on the remote host.
3. **Config fallback** — otherwise the DB credentials are read from the application's own config file (fetched with `cat` and parsed locally — no PHP/Node/Python runtime needed on the remote host), and the dump is created with the remote database client. Supported per platform:

   | Platform | Config file (relative to `ssh/site_root_path`) |
   |---|---|
   | magento2 | `app/etc/env.php` (`db.connection.default`) |
   | shopware, sylius, medusa, saleor, spree | `.env` / `.env.local` (`DATABASE_URL`) |
   | woocommerce | `wp-config.php` (`DB_*` constants) |
   | prestashop | `app/config/parameters.php` (`database_*`) |
   | shopify, bigcommerce, custom | no standard DB config — use option 1 or 2 |

> For a remote host that has only `madock` installed (no PHP, no `mysqldump`), option 2 is used automatically — make sure `ssh/site_root_path` points to the madock project root on the server.

For DB exporting run command:
```
madock db:export
```
The files of DB dumps are stored in `madock/projects/{projects_name}/backup/db`

### Sharing a database between two projects

If you need two projects to use the same database (for example, a storefront and an admin panel working with one DB), you can do this without any extra configuration in madock.

1. Start **Project A** (the one that owns the database)
2. Get the database connection info:
   ```
   cd /path/to/project-a
   madock db:info
   ```
   Note the **port** number from the output.

3. In **Project B**, configure your application to connect to Project A's database using:
   - **Host:** `host.docker.internal`
   - **Port:** the port from step 2
   - **User/Password/Database:** credentials from Project A's `db:info` output

For example, in Magento 2 `app/etc/env.php`:
```php
'db' => [
    'connection' => [
        'default' => [
            'host' => 'host.docker.internal:17004', // port from Project A
            'dbname' => 'magento',
            'username' => 'magento',
            'password' => 'magento',
        ],
    ],
],
```

> **Note:** Project B will still start its own database container. You can ignore it — your application will connect to Project A's database through the host-mapped port.
