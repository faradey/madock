# Customizations

## Custom properties

There are some properties that can be customised if your project setup differs from default one. You can do so by adding a file in one of the following paths and setting your custom values.

### Default properties

* `<MADOCK_ROOT>/config.txt` - Origin Properties. This file cannot be changed
* `<MADOCK_ROOT>/projects/config.txt` - Overridden properties. This file must be created manually if you want to override the default values

Default list of properties that can be customised:

* See [config.txt](../config.txt)


### Custom properties paths

* `<MADOCK_ROOT>/docker/...` - Origin Properties. This files cannot be changed
* `<MADOCK_ROOT>/projects/<PROJECT_NAME>/docker/...` - You can copy the original file to the project folder while keeping the original path. Changes to this file will be applied when you run the command `madock rebuild`

Default list of properties that can be customised:

* [docker/docker-compose.yml](../docker/magento2/docker-compose.yml)
* etc.