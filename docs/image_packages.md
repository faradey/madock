# Image packages

What the php and application images install is a setting, not a list inside a
template. A project takes the tools it needs without copying a Dockerfile into
`.madock/docker/` — a copy that would then stop receiving every later fix.

## The keys

| Key | Default | Image |
|---|---|---|
| `php/packages/default` | the apt list the php image always installed | php, first layer |
| `php/packages/extra` | empty | php, last layer |
| `php/extensions/default` | `bcmath cli common curl dev fpm gd intl mbstring mysql soap sqlite3 xml xsl zip imagick ctype dom fileinfo iconv simplexml sockets tokenizer xmlwriter ssh2 redis` | php, second layer, each name installed as `php<version>-<name>` |
| `php/extensions/extra` | empty | php, last layer, `php<version>-<name>` |
| `app/packages/default` | the apt list the ubuntu-based application image always installed | `none`, `python`, `ruby` languages, first layer |
| `app/packages/extra` | empty | `none`, `python`, `ruby`, `golang`, last layer |
| `nodejs/packages/extra` | empty | the node container, last layer |

Lists are space-separated names:

```bash
madock config:set --name php/packages/extra --value "htop poppler-utils"
madock config:set --name php/extensions/extra --value "gmp bz2"
madock rebuild --changed
```

`php_without_xdebug` takes the `extra` keys too; its first layers keep their own
list.

## `extra` — adding a tool

The `extra` packages are installed as the image's **last** layers, after
everything projects share. Docker keeps one copy of a layer for every image
that has it, so the heavy layers below stay shared between the projects on a
machine whatever each adds on top.

All names are installed in one go first. When that fails, each is installed on
its own, and a name the package index does not have is reported in the build
output — `madock: <name> is not available in this index` — and skipped. It never
stops the build: a rebuild that stops halfway on a server leaves the site down.
Read the build output after adding a name.

## `default` — owning the list

Setting `default` replaces the first layer's list whole. Two consequences, both
deliberate:

- **The project stops receiving packages madock adds to the default later.** The
  setting holds the list as written; a name added to madock's default in a later
  release does not reach a project that has its own. To add a tool, use `extra`.
- **The first layer differs from every other project's**, so this project no
  longer shares the heavy layers with them on the same machine.

Nothing in the list is enforced. The names below are what madock and madock-pro
rely on, and removing one is the project's decision.

## What stops working without a name

Checked against the madock and madock-pro code on 2026-10-07.

### php image

| Name | What depends on it |
|---|---|
| `locales` | `locale-gen` in the same layer — the build fails without it |
| `software-properties-common` | `add-apt-repository ppa:ondrej/php` in the same layer — the build fails, and no PHP package can be installed |
| `ca-certificates` | every HTTPS download during the build: the composer installer, NodeSource, ionCube |
| `curl` | the ionCube download (`php/ioncube/enabled`), NodeSource (`nodejs/embedded/enabled`), the Magento Cloud CLI and n98-magerun downloads |
| `msmtp` | `mail()` — PHP's `sendmail_path` (`php/sendmail/enabled`), and madock-pro's `mail:setup` |
| `procps` | madock-pro's SSH access: the sidecar is built on the application image and runs `pgrep` |
| `libmcrypt-dev` | the pecl mcrypt build for PHP 7.2–8.3 |
| extension `cli` | composer's installer and every `madock cli php …` |
| extension `fpm` | the container's own process — `php-fpm<version>` is its command |
| extension `common` | every other extension |
| extension `dev` | `pecl` and `phpize`: the xdebug and mcrypt builds |

### application image (`none`, `python`, `ruby`)

| Name | What depends on it |
|---|---|
| `locales` | `locale-gen` in the same layer — the build fails without it |
| `ca-certificates`, `curl` | NodeSource, when `nodejs/embedded/enabled` is on |
| `procps` | madock-pro's SSH access, as above |
| `build-essential` | native modules of `pip` and `gem` installs |
