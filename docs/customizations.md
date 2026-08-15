# Customizations

## Custom properties

There are some properties that can be customised if your project setup differs from default one. You can do so by adding a file in one of the following paths and setting your custom values.

### Default properties

* `<MADOCK_ROOT>/config.xml` - Origin Properties. This file cannot be changed
* `<MADOCK_ROOT>/projects/config.xml` - Overridden properties. This file must be created manually if you want to override the default values

Default list of properties that can be customised:

* See [config.xml](../config.xml)


### Custom properties paths

Docker configuration files are resolved using a fallback chain (first found wins):

1. `<PROJECT_ROOT>/.madock/docker/...` - In-project overrides (highest priority)
2. `<MADOCK_ROOT>/projects/<PROJECT_NAME>/docker/...` - Per-project overrides
3. `<MADOCK_ROOT>/docker/<PLATFORM>/...` - Platform-specific defaults (e.g., `magento2`, `shopware`)
4. `<MADOCK_ROOT>/docker/languages/<LANGUAGE>/...` - Language-specific defaults (e.g., `php`, `python`, `golang`)
5. `<MADOCK_ROOT>/docker/general/service/...` - General service defaults (lowest priority)

To customize a Docker file, copy it from its origin to one of the override paths while keeping the relative path. Changes will be applied when you run `madock rebuild`.

Default list of properties that can be customised:

* [docker/docker-compose.yml](../docker/magento2/docker-compose.yml)
* etc.

## Template syntax

Every file under `docker/` is a Go [text/template](https://pkg.go.dev/text/template)
with the delimiters changed to `{{{` and `}}}`. The delimiters are three braces
rather than two because the templates are full of shell and nginx, which use
`{{ }}` and `${ }` of their own — and they are not `<<< >>>` because a Dockerfile
line like `read major minor patch <<< "$x"` is a bash here-string.

A setting is read by its path, with the slashes written as dots:

```
FROM php:{{{.php.version}}}-fpm
WORKDIR {{{.workdir}}}
```

The names are the ones `madock config:set` uses — `php/xdebug/enabled` is
`.php.xdebug.enabled`. A setting the project does not have reads as empty, so a
snippet may ask about a service the platform has never heard of.

Conditionals, with `and`, `or`, `not`, `eq` and `ne`:

```
{{{- if .php.xdebug.enabled}}}
RUN pecl install xdebug-{{{.php.xdebug.version}}}
{{{- end}}}

{{{- if and (eq .db.type "mysql") (versionLt .db.version "8.4")}}}
--default-authentication-plugin=mysql_native_password
{{{- end}}}
```

The `-` inside a delimiter deletes the whitespace on that side, which is what
keeps a switched-off block from leaving a blank line behind. Write conditionals
that own a line with `{{{-` on both the opening and the closing tag.

Loops, over the project's hosts:

```
    extra_hosts:
      {{{- range $host := .nginx.hosts}}}
      - "{{{$host.name}}}:host-gateway"
      {{{- end}}}
```

`.nginx.hosts` is ordered and never empty — a project with no host configured
gets `loc.<project>.com` — so `{{{(index .nginx.hosts 0).name}}}` is the first
host.

Another file is included by name, relative to `docker/`. The override chain
above applies to the snippet as well, so a project can replace one snippet and
keep the rest:

```
{{{template "snippets/docker-compose/php.yml" .}}}
```

Beyond the [built-in functions](https://pkg.go.dev/text/template#hdr-Functions),
madock adds:

| Function | What it does |
|---|---|
| `port "livereload"` | The host port published for a service. **Calling it allocates the port**, which is why it is a function and not a setting |
| `versionGte`, `versionGt`, `versionLte`, `versionLt` | Compare two dotted versions: `versionLt .db.version "8.4"` |
| `join`, `lower`, `upper` | As they read |

A template that does not parse stops the run and names the file and the line.

### The old `<<<if>>>` syntax

Before v3.9.1 madock read these files with an engine of its own:
`<<<if{{{php/enabled}}}>>>…<<<endif>>>`, `{{{include snippets/…}}}` and
`{{{php/version}}}` without the leading dot.

An override still written that way keeps working — it is converted as it is read
— but madock prints a warning naming the file, and the conversion will be removed
in a later release. To convert the files for good:

```bash
madock template:convert                 # the current project's .madock/docker/
madock template:convert --dry-run       # say what would change, write nothing
madock template:convert /some/other/dir # a copy kept somewhere else
```

It rewrites in place, reports every file it touched, and running it twice is a
no-op. It is the same conversion the renderer applies as it reads, so the result
cannot differ from what the warning was already producing.

To convert by hand instead, the table is:

| Old | New |
|---|---|
| `{{{php/version}}}` | `{{{.php.version}}}` |
| `{{{include snippets/x}}}` | `{{{template "snippets/x" .}}}` |
| `{{{port/nginx}}}` | `{{{port "nginx"}}}` |
| `<<<if{{{a}}}{{{b}}}>>>` | `{{{- if and .a .b}}}` |
| `<<<else>>>` / `<<<endif>>>` | `{{{- else}}}` / `{{{- end}}}` |
| `{{{nginx/host_gateways}}}` | a `range` over `.nginx.hosts` |
| `{{{db/type_is_mysql}}}` | `(eq .db.type "mysql")` |
| `{{{db/use_default_auth_plugin}}}` | `(not (and (eq (lower .db.repository) "mysql") (versionGte .db.version "8.4")))` |