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