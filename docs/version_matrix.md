# Version matrix

Which tool versions a new project gets by default — PHP, database, search
engine, Redis, RabbitMQ, Node and the rest — is decided by the version matrix.
It is data, one XML file per platform, compiled into the binary:

```
src/model/versions/matrix/<platform>.xml
src/model/versions/matrix/common.xml      what every platform shares: Xdebug, the menus
```

`madock setup` reads it twice: for the version you type or that madock finds in
`composer.json`, and for the presets it offers in the menu. Both are the same
answer — a preset is a platform version with a name, not a separate list of
versions. Existing projects are not affected by a change to the matrix: their
stack is in their own configuration.

## Format

The files are shaped like `config.xml`: elements only, **no attributes**.

```xml
<config>
    <name>magento2</name>
    <tools>
        <php>
            <rule><from>2.4.9</from><value>8.5</value></rule>
            <rule><from>2.4.8</from><value>8.4</value></rule>
            <rule><from>2.4.6</from><value>8.2</value></rule>
        </php>
        <opensearch>
            <rule><from>2.4.3-p2</from><value>1.2.0</value></rule>
            <rule><version>2.3.7-p4</version><value>1.2.0</value></rule>
            <default>NotCompatible</default>
        </opensearch>
    </tools>
    <presets>
        <preset>
            <version>2.4.9</version>
            <name>Magento 2.4.9 (Latest)</name>
            <description>Latest stable with PHP {php}, OpenSearch {opensearch}</description>
        </preset>
    </presets>
</config>
```

- **A tool is an element named after it; its rules are read top to bottom and
  the first that matches wins.**
- `<from>X</from>` matches X and every later version. Versions are compared as
  numbers, segment by segment — `2.4.10` is later than `2.4.9` — and a Magento
  `-pN` is a patch on top of its release: `2.4.6-p11` is later than `2.4.6` and
  earlier than `2.4.7`. Put a patch line below the next minor's rules.
- `<version>X</version>` matches exactly that string. It is how named variants
  are written: Shopify's `hydrogen`, BigCommerce's `catalyst`.
- `<value/>` — empty — answers nothing: that stack does not run the service.
  Every rule has a `<value>`; a missing one fails the tests.
- No match gives `<default>`, or nothing.
- `<by>php</by>` asks a tool about another tool's answer instead of the platform
  version. Xdebug follows PHP; a stack with no PHP gets no Xdebug.
- `NotCompatible` means the tool cannot run with that platform version.
  `setup` refuses it rather than writing it into the config as a version.

Tool names: `language`, `php`, `dbtype`, `db`, `searchengine`, `elasticsearch`,
`opensearch`, `composer`, `redis`, `valkey`, `rabbitmq`, `xdebug`, `nodejs`,
`yarn`, `python`, `golang`, `ruby`. An unknown name, a tool defined twice, a
rule with both or neither of `<from>` and `<version>`, or an attribute anywhere
fails the tests.

### Budgets: what a program takes inside its container

A platform file may set its own budgets where they should differ from madock's
default for every platform. They nest the way `config.xml` does:

```xml
<budgets>
    <search>
        <opensearch>
            <heap>2g</heap>
        </opensearch>
    </search>
</budgets>
```

| Key | What it is |
|---|---|
| `db/memory` | the database engine's budget, split into its buffer pool and caches (MariaDB/MySQL, PostgreSQL, MongoDB). The container itself has no limit |
| `search/elasticsearch/heap`, `search/opensearch/heap` | Java's heap (`-Xmx`) |
| `search/elasticsearch/memory_limit`, `search/opensearch/memory_limit` | the search engine's **container** limit — the one container limit in open madock, because the JVM lives beyond its heap and cannot run without one |
| `memcached/memory` | Memcached's cache size, in megabytes with no unit |

These are budgets, not container limits. Every other container limit — CPU and
memory for any service — is madock Pro's `resources:*`.

A budget is applied when a project's configuration is read, **never written
into it**, so a change reaches every project of that platform at its next
rebuild. The order is: madock's defaults, then the platform's budget, then
`projects/config.xml` (what `config:set` writes for the installation), then the
project — anything set by hand wins. A value equal to madock's default fails the
tests: it would be a copy that stops following the default the day the default
changes.

No platform sets one yet; the defaults in `config_defaults.xml` apply to all.

### What the menus offer

`common.xml` also holds what `setup`'s menus list for each service, newest
first:

```xml
<offers>
    <opensearch><v>3.0.0</v><v>2.19.1</v><v>2.12.0</v></opensearch>
</offers>
```

Names: `php`, `dbtype`, `mariadb`, `mysql`, `postgresql`, `mongodb`,
`composer`, `searchengine`, `elasticsearch`, `opensearch`, `redis`, `valkey`,
`memcached`, `artemis`, `xdebug`, `rabbitmq`, `nodejs`, `yarn`, `language`,
`python`, `golang`, `ruby`. The menu adds "Custom" where a free value makes
sense, and **always shows the default the matrix gave**, highlighted, whether
or not the list carries it — a default can never be out of reach. The engine,
search-engine and language menus are closed lists and show only what they list.

A preset's `<description>` may name tools in braces — `{php}` — and is filled in
from the versions the preset gets, so it cannot name a version the preset does
not install.

## Reading it from outside madock

```bash
madock matrix                       # every platform, then common.xml, as text
madock matrix --json                # the same as JSON
madock matrix --json --platform magento2
```

The JSON is the files as madock reads them, not a table of answers: per
platform its tools in file order, each with `by`, its `rules` (exactly one of
`from` or `version`, and `value`) and `default`; its presets with the
description already filled in; its `budgets` by config key. `common` holds the
shared tools and the menu `offers`, and is printed with `--platform` too, because
a platform's answers are not complete without it. `default` is `null` when a
tool has none — no answer at all — and `""` when it answers nothing; the two are
different. An unknown `--platform` exits 1 with stdout empty and names the known
ones on stderr.

To answer a version the way madock does: take the platform's tools in order, then
the common tools it does not define; for each, the first rule that matches wins,
otherwise `default`; a tool with `by` is asked about the answer of the tool it
names instead of the platform version, and answers nothing when that is empty.
`TestTheExportAnswersWhatTheMatrixAnswers` holds the export to exactly that: every
version any rule names, and every preset, gets from the export what `setup` gets.

## Where the values come from

Each file says in a comment what it was checked against and when. The rule is:
the version the vendor states — on its requirements page, or in the `engines`
or `require` of its own package — and the newest of those it supports. Where
the vendor states nothing, the value chosen earlier stays, and the comment says
so.

## The warning at start and rebuild

Since 4.5.16 `madock start` and `madock rebuild` hold the project's configured
versions against the matrix and warn, on stderr, about each one past its line:

```
Outside the version matrix: php/version 8.4 is newer than magento2 2.4.7 supports (8.3 by the matrix)
A warning only: madock starts it as configured. `madock matrix --platform magento2` shows the lines.
```

- The platform version comes from the project's code, the way `setup` detects
  it. Code whose version cannot be read, or code of another platform than the
  config names, is not compared.
- Versions are compared by major.minor, because a line names a release line:
  OpenSearch 2.12.1 is within Adobe's 2.12.0.
- Only newer is named. A line is the newest supported version and the matrix
  has no floor, so an older version is neither confirmed nor flagged.
- A tool the matrix answers `NotCompatible` for is named when it is enabled.
- Only what runs is compared: a service with `enabled` false, PHP and Composer
  when the language is not PHP, a database of another engine than the
  matrix's (MariaDB when the matrix names none, as in `setup`).
- An answer that is no version — medusa's `postgres:17` — is skipped.

It never stops anything. Whoever pinned a version past the line may know
something the matrix does not.

## Changing it

Edit the XML, then:

```bash
go test ./src/model/versions/golden/ -update
git diff src/model/versions/golden/testdata/matrix.golden.json
```

The golden file is every answer the matrix and the presets give for every
boundary version of every platform. Each changed line in that diff is a new
project that will be set up differently — review it before committing.
