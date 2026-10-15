# Version matrix

Which tool versions a new project gets by default — PHP, database, search
engine, Redis, RabbitMQ, Node and the rest — is decided by the version matrix.
It is data, one XML file per platform, compiled into the binary:

```
src/model/versions/matrix/<platform>.xml
src/model/versions/matrix/common.xml      tools every platform shares (Xdebug)
```

`madock setup` reads it twice: for the version you type or that madock finds in
`composer.json`, and for the presets it offers in the menu. Both are the same
answer — a preset is a platform version with a name, not a separate list of
versions. Existing projects are not affected by a change to the matrix: their
stack is in their own configuration.

## Format

```xml
<platform name="magento2">
    <tool name="php">
        <when from="2.4.9">8.5</when>
        <when from="2.4.8">8.4</when>
        <when from="2.4.6">8.2</when>
    </tool>
    <tool name="opensearch">
        <when from="2.4.3-p2">1.2.0</when>
        <when version="2.3.7-p4">1.2.0</when>
        <default>NotCompatible</default>
    </tool>
    <preset version="2.4.9" name="Magento 2.4.9 (Latest)"
            description="Latest stable with PHP {php}, OpenSearch {opensearch}"/>
</platform>
```

- **Rules are read top to bottom; the first that matches wins.**
- `from="X"` matches X and every later version. Versions are compared as
  numbers, segment by segment — `2.4.10` is later than `2.4.9` — and a Magento
  `-pN` is a patch on top of its release: `2.4.6-p11` is later than `2.4.6` and
  earlier than `2.4.7`. Put a patch line below the next minor's rules.
- `version="X"` matches exactly that string. It is how named variants are
  written: Shopify's `hydrogen`, BigCommerce's `catalyst`.
- An empty rule — `<when version="hydrogen"></when>` — answers nothing: that
  stack does not run the service.
- No match gives `<default>`, or nothing.
- `by="php"` asks a tool about another tool's answer instead of the platform
  version. Xdebug follows PHP; a stack with no PHP gets no Xdebug.
- `NotCompatible` means the tool cannot run with that platform version.
  `setup` refuses it rather than writing it into the config as a version.

Tool names: `language`, `php`, `dbtype`, `db`, `searchengine`, `elastic`,
`opensearch`, `composer`, `redis`, `valkey`, `rabbitmq`, `xdebug`, `nodejs`,
`yarn`, `python`, `golang`, `ruby`. An unknown name, a tool defined twice, or a
rule with both or neither of `from` and `version` fails the tests.

### What the menus offer

`common.xml` also holds what `setup`'s menus list for each service, newest
first:

```xml
<offer name="opensearch"><v>3.0.0</v><v>2.19.1</v><v>2.12.0</v></offer>
```

Names: `php`, `dbengine`, `mariadb`, `mysql`, `postgresql`, `mongodb`,
`composer`, `searchengine`, `elastic`, `opensearch`, `redis`, `valkey`,
`memcached`, `artemis`, `xdebug`, `rabbitmq`, `nodejs`, `yarn`, `language`,
`python`, `golang`, `ruby`. The menu adds "Custom" where a free value makes
sense, and **always shows the default the matrix gave**, highlighted, whether
or not the list carries it — a default can never be out of reach. The engine,
search-engine and language menus are closed lists and show only what they list.

A preset's `description` may name tools in braces — `{php}` — and is filled in
from the versions the preset gets, so it cannot name a version the preset does
not install.

## Where the values come from

Each file says in a comment what it was checked against and when. The rule is:
the version the vendor states — on its requirements page, or in the `engines`
or `require` of its own package — and the newest of those it supports. Where
the vendor states nothing, the value chosen earlier stays, and the comment says
so.

## Changing it

Edit the XML, then:

```bash
go test ./src/model/versions/golden/ -update
git diff src/model/versions/golden/testdata/matrix.golden.json
```

The golden file is every answer the matrix and the presets give for every
boundary version of every platform. Each changed line in that diff is a new
project that will be set up differently — review it before committing.
