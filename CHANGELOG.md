**v3.0.0**
- Introduced a generic diff command: `madock diff --platform <code> --old <ver> --new <ver> [--path <publicDirFromSiteRoot>]`
- Added store scopes documentation split into a dedicated file `docs/store_scopes.md` and linked from README
- Added Valkey key-value DB
- Minor fixes and refactors in diff scripts (path handling and directory creation)

**v2.9.1**
- Added Magento 2.4.8 support
- Fixed the restart policy for aruntime containers

**v2.9.0**
- Added the env variable MADOCK_TTY_ENABLED (0/1). MADOCK_TTY_ENABLED is enabled by default
- Fixed SSH volume
- Fixed "install" command for prestashop platform
- Fixed docs
- Added logo
- Fixed GetRunDirPath function for outside executors
- Added php8.4 support
- Fixed incorrect version comparison for MariaDB
- Fixed arguments for the Setup command
- Fixed Magento2 install subcommands
- Fixed livereload
- Fixed apt-get to apt and added --allow-releaseinfo-change
- Added php-redis library to php installation
- Fixed RabbitMQ recommended version for Magento 2.4.7-p5 and later
- Added the restart policy

**v2.8.0**
- Added **PrestaShop** as a separate service
- Fixed "composer" command for Shopify service
- Improved custom commands and documentation

**v2.7.0**
- Fixed the creation of patches
- Fixed the cron for Shopify platform
- Fixed TODO comments
- Fixed NodeJs major version for php.Docker file
- Added http2 in the nginx configuration

**v2.6.0**
- Added Grafana as a service
- Added Grafana dashboards for Loki, Mysql and Redis
- Support for snippets in configuration files has been added. This has allowed us to eliminate repetitive code and settings.
- Added the new option `--shell` for `madock bash` command. It can be used `bash` or `sh` as a shell.

**v2.5.0**
- Added supporting of Shopware
- Fixed mailcatcher configuration with MP_SMTP_AUTH_ACCEPT_ANY and MP_SMTP_AUTH_ALLOW_INSECURE
- Fixed documentation
- Fixed the media synchronization public path
- Added --db-host, --db-port, --db-name, --db-user, --db-password as options for the remote:sync:db command

**v2.4.4**
- Fixed opensearch-dashboards
- Added new command `madock project:clone` [more](docs/project_clone.md)
- Added php/nodejs service to the php container
- Fixed documentation
- Fixed bug with the `madock cli` command
- Added custom commands [more](docs/custom_commands.md)

**v2.4.3**
- Added interactive options for the `madock setup` command
- Added an isolation mode [more](docs/isolation.md)
- Added Varnish cache [more](docs/varnish.md)
- Refactoring code


**v2.4.2**
- Support Magento 2.4.7 and Adobe Commerce 2.4.7
- Updated docker-compose version to 3.8
- Fixed DB host for the service db2
- Fixed GetActiveProjects method
- Fixed start/stop project
- Fixed db:export
- Fixed node grunt exec:<theme>
- Fixed documentation
- Added "RUN npm install -g grunt-cli" to docker file
- Fixed bug with "cache" folder
- Fixed if/else in config files
- Fixed project configuration
- Fixed Snapshot container
- Added snapshots functionality for the project
- Fixed .madock/config.xml
- Update PHP mcrypt version
- Fixed OpenSearch env variables



**v2.4.1**
- Added command scope:add to add a new scope and activate it
- Added the ability to store the madock configuration within a project in the .madock folder. To do this, you need to manually create a .madock folder and transfer configuration files and database backups to it, if necessary
- Added full support for creating patches for cweagans/composer-patches
- Added full support for creating patches for vaimo/composer-patches
- Added logger with stack trace
- Fixed the config cache
- Fixed the bug with the enable/disable services
- Fixed compatible version magerun n98 and PHP
- Fixed Adobe Cloud commands
- Fixed project path
- Fixed db:import
- Fixed bug with config.xml and the setup of a new project
- Fixed missing dir aruntime/projects
- Fixed working commands Start, Stop, Restart without internet
- Fixed madock info
- Fixed xdebug profile for PHP 7.1 or less


**v2.4.0**
- Added the new option PUBLIC_DIR in the project configuration. Each platform can have a different path of public folder therefore this option will be specified as a public folder in the container.
- Fixed host for phpmyadmin2
- Fixed mcrypt extension for PHP
- Fixed mail for CLI
- Improve command "madock c:f"
- Added --force option for the command "madock rebuild". Removes running containers without waiting for them to complete correctly and creates new containers.
- Added new library for CLI commands
- Replaced Mailhog to Mailpit
- The configuration file format would be changed from .txt to .xml. The project configuration file env.txt has been renamed to config.xml. The old configuration files have been preserved so that if you have problems with the new version of Madock, you can roll back to the old version.
- Configuration scopes for the project have been added. Now switching between configurations has become convenient and there is no need to create a copy of the project in a neighboring folder. The database is also separate for each scope.
- Added the new command "madock scope:list" for listing all scopes of the project.
- Added the new command "madock scope:set" for switching between scopes of the project.
- The commands "remote:sync:media", "remote:sync:db" and "remote:sync:file" have received an additional option "--ssh-type" which specifies the prefix of the name of the ssh settings in the project configuration. This way you can specify which ssh settings to use when executing the command.
- Added aruntime configuration caching. Now Madock will parse files less when starting and rebuilding a project.
- Added the new command "madock config:cache:clean" for cleaning Madock aruntime cache.
- Added the new command "madock open" for opening the project in the browser.
- Improve documentation of Madock

**2.2.0**
- Shopify support
- Custom PHP project support
- Relocated setup option "Specify Magento version" to top
- Added CONTAINER_NAME_PREFIX option in config. This option will allow you to run a madock project independently of other docker builds in the space with the default madock_ prefix. For already configured projects, the space will have an empty prefix to prevent projects from breaking.
- Added --ignore-table for "db:export" and "remote:sync:db" commands. Ignore the table when exporting. The specified table will not be included in the backup file. To specify multiple tables, specify this option multiple times.
- Updated OS Ubuntu for containers from 20.04 to 22.04. This will only affect those projects that will be installed after updating this build.
- Improve documentation for new commands
- Fixed some problems with NodeJs
- Fixed issue #9

Thanks @artmouse @serhii-chernenko

**2.1.0**
- Support the Magento Functional Testing Framework (MFTF)
- Fixed multiline commands

**2.0.1**
- Fixed the setup with Hosts
- Fixed the setup with the version Redis and rabbitMQ
- Fixed "madock status" command
- Fixed the DB host description

**2.0.0**
- PWA Studio as a separate service.
- Backward incompatible changes were made to the code. Code changes allow new platforms to be added in the future.
- At the moment, PWA Studio has been added as a separate service.
- There are plans to add Shopify and Shopware in the future.

**1.9.1**
- Fixed command project:remove
- Removed "restart: on-failure:3" from Elasticsearch service of docker-compose
- Installed libssh2-1-dev libssh2-1 php-ssh2 for PHP
- Removed the restart_if_failure option for the DB service of docker-compose
- Improved removing project. Now deletion is more transparent. Before execution, you will see the items that will be deleted and only after your confirmation will they be deleted.
- Fixed files permission with --with-chmod

**1.9.0**
- Added
  - Support Magento 2.4.6
  - Support sample data with the setup command
  - OpenSearch
  - Support PHP 8.2 and xdebug
  - Improved patcher for creating patches from the whole folder
  - Updated phpmyadmin version from 5.2.0 to 5.2.1
  - Increased UPLOAD_LIMIT for phpmyadmin. Now it is 2GB
  - Custom DB repository in the config
  - PHP 8.2 to the setup process
  - Xdebug profile
  - Increased PHP Max Input Vars Limit by default
  - Enabled log_bin_trust_function_creators for DB
  - New option for DB commands "--service-name DB container name. Optional. Default container: db. Example: db2"
  - Support overriding /docker/nginx/conf/default-proxy.conf
  - Command "install"
  - Support n98-magerun
  - Support the second DB
  - Support proxy as a service

- Fixed
  - Default_server for the nginx proxy configuration
  - Remove --single-transaction option from the mysqldump command
  - Remove the innodb_log_file_size option for MySQL 8.x
  - Improved elasticsearch plugins installation
  - Cron
  - Bug with the start/stop command of the proxy server
  - FOREIGN_KEY_CHECKS for the import DB
  - Project setup with Redis and rabbitMQ versions
  - Bug with the media synchronization
  - Proxy port and the starting script
  - Livereload location in nginx proxy
  - DEFINER for the DB import/export
  - Issue with permissions of .ssh folder #8

**1.8.2**
- Fixed generation env.php file with rabbitmq password

**1.8.1**
- Fixed starting the Nginx proxy containers

**1.8.0**
- Added a new command "patch:create"
- Added a new param "--name" for "db:export" and "remote:sync:db" 
- Added a new command setup:env for generating env.php file
- Changed domain .loc to .test by default
- Optimization for MariaDB 10.4
- Prune the volumes with option --with-volumes. For example Madock prune --with-volumes
- Added the ability to specify a custom repository and version of docker images when you set up the project
- Added "--with-chown" option for some commands. Reset permissions for files and folders
- Improved "db:import" command. Now, the Madock can read DB files from any folder of the Magento project. The name of the DB file must contain ".sql" in any part of the name
- Fixed the problem with the same project folder names from different locations
- Added a new command project:remove
- Added stopping proxy containers if there are no active projects
- Refactoring code

**1.7.4**
- Additional changing external IPs for containers from 0.0.0.0 to 127.0.0.1

**1.7.3**
- Changed external IPs for containers from 0.0.0.0 to 127.0.0.1
- Fixed bug with CLI options and arguments

**1.7.2**
- Fixed bug with the docker compose

**1.7.1**
- The internal command "docker-compose" was replaced by "docker compose"

**1.7.0**
- All commands are brought to uniformity. Now they match the Magento approach
- Added the support of Magento cloud
- Added the support of automatically creating composer patches
- Added the new command "cli"
- Fixed some bugs
- Some code improvements

**1.6.0**
- Added the LiveReload plugin and NodeJs  
- Added automatic start of containers after project setup 
- Added the ability to download a specific file from a remote server (for example: madock remote sync file --path app/etc/config.php)    
- Now changed project configuration is applied only after setup or rebuild commands   
- Fixed some bugs and added some improvements 

**1.5.0**
- Added new options for the setup command:    
  - --download - Download the specific Magento version from Composer to the container
  - --install - Install Magento, Shopware, etc. from the source code
- Added new command madock db info. This command prints data for connecting to the database. The output contains a port (permanent) for connecting such database programs as HeidiSQL, MySQL Workbench, and others
- Support Windows OS

**1.4.0**
- Added
  - Kibana  
  - CHANGELOG.md    
  - MADOCK_VERSION in global config.txt 
  - new functionality with services. For example: madock service phpmyadmin on  
- Fixed   
  - text of warning with DB import selecting

**1.3.0**
- For media, js, css requests it was added a new container without Xdebug. This improvement decreases load when you debug your code

**v1.2.0**
- Added a new command for displaying the status of the project   
  - madock status

**v1.1.0**
- Added support for PHP 8.1
- Added support for SSL certificates. Now you can use HTTPS in local development

**v1.0.3**
- Fixed remote sync DB

**v1.0.2**
- Added  
  - Additional logging for sync
  - Validation of project folder name  
- Fixed  
  - Mapping for the general config  
  - Remove compression for an image in png format   
  - Improve sync media files    

**v1.0.1**
- Remove the unison container for macOS

**v1.0.0**
- change docs