# Workflow

The following guide shows you the normal development workflow using Madock.

IMPORTANT: After changing any option in the following files, you should run `madock rebuild`
madock/projects/config.xml
madock/projects/{project name}/env.xml

#### 1. Start containers

```
madock start
```

#### 2. Composer commands

```
madock composer <command>
```

#### 3. Magento commands

```
madock magento <command>
```

#### 4. Working on frontend

```
madock node <command>
madock node grunt exec:<theme>
madock node grunt watch
```

**IMPORTANT:** For the Chrome browser, you can download the LiveReload plugin specifically for Madock from the link [Google Chrome plugin](https://chrome.google.com/webstore/detail/livereload-for-madock/cmablbpbnbbgmakinefjgmgpolfahdbo). Then install it and enable it for the site you need.

**NOTE:** You might also need to disable your browser cache. For example in Chrome:

* `Open inspector > Settings > Network > Disable cache (while DevTools is open)`

#### 5. xdebug

* Enable xdebug

  ```
  madock debug:enable
  ```

* Configure xdebug in PHPStorm (Only first time)

    * [PHPStorm + Xdebug Setup](./xdebug_phpstorm.md)

* Disable xdebug when finish

  ```
  madock debug:disable
  ```

#### 6. SSL certificates

If you want to manually add an ssl certificate to the browser, you can find it at [path to Madock folder]/aruntime/ctx/madockCA.pem
If the SSL certificates do not work, run the `madock ssl:rebuild` command and restart your browser.

#### 7. auth.json

If your project does not have an auth.json file, then when executing `composer` commands, the global auth.json file will be used.

#### 8. Multistores and website codes

Magento uses "base" as the store code by default.
But if you are using multistore, then you need to specify the code of each website along with the website host in the madock configuration. For example: `madock config:set --name=HOSTS --value="website1.test:base website2.test:websitecode"`. You can see site codes in the database table store_website. Or by querying the database `SELECT * FROM store_website`.

#### 9. help
```
  madock help
 ```

This command shows you the following items:

* `bash`    Connect into container using bash

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]` Name of container. Optional. Default container: php. For example: php, node, db, nginx

* `c:f`  Cleaning up static and generated files


* `cli`  Execute any commands inside php container. If you want to run several commands you can cover them in the quotes. For example: `madock cli "php bin/magento setup:upgrade && php bin/magento setup:di:compile"`


* `cloud`  Executing commands to work with Magento Cloud. Also, can be used the long command: magento-cloud)


* `composer`  Execute composer inside php container. For example: `madock composer install`
            
            
* `compress`  Compress a project to archive
 

* `config:cache:clean`  Clearing internal Madock cache
* `c:c:c`  The short alias of `config:cache:clean` command


* `config:list`  List all project environment settings


* `config:set`  Set a new value for parameter. For example: `madock config:set --name=HOSTS --value="website1.test:base website2.test:websitecode"`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name, -n`     Parameter name

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--value, -v`     Parameter value
               
         
* `cron:enable`    Enable cron


* `cron:disable`    Disable cron
              
          
* `db:import`      Import database

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`-f`  Force mode

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--service-name, -s`  DB container name. Optional. Default container: db. Example: db2


* `db:export`      Export database. For example: `madock db:export --name=fromdevsite`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name, -n`  Name of the DB export file

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--service-name, -s`  DB container name. Optional. Default container: db. Example: db2

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--ignore-table, -i`  Ignore the table when exporting. The specified table will not be included in the backup file. To specify multiple tables, specify this option multiple times.


* `db:info`      Information about credentials and remote host and port
 
   
* `debug:enable`   Enable xdebug


* `debug:disable`   Disable xdebug
                     

* `debug:profile:enable`   Enable xdebug profiling


* `debug:profile:disable`   Disable xdebug profiling
                     
   
* `info`   Show information about third-parties modules (name, current version, latest version, status)             
    
    
* `install`   Install Magento. It is a synonym for `madock magento setup:install` with additional actions.            
    
    
* `help`    Displays help for commands
                      
  
* `logs`    View logs of the container. For example: `madock logs php`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]`     Container name. Optional. Default container: php. Example: php
                        

* `magento` or `m`   Execute Magento command inside php container. For example: `madock m setup:upgrade`
        
                
* `mftf`   Execute MFTF command inside php container. For example: `madock mftf generate:tests`
                        

* `mftf:init`   Init MFTF configuration. For example: `madock mftf:init`
                        

* `n98`   Execute n98 command inside php container. For example: `madock n98 sys:info`
                        
                        

* `node`    Execute NodeJs command inside php container. For example: `madock node grunt exec:<theme>`
                        

* `open`    Open project in browser

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--service, -s`     Open a specific project service in the browser. For example: phpmyadmin                   


* `patch:create`   Create patch. The patch can be used with the composer plugin cweagans/composer-patches. For example: `madock patch:create --file=vendor/magento/module-analytics/Cron/CollectData.php --name=collect-data-cron.patch --title="Collect data cron patch" --force`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--file`     Path of changed file. For example: vendor/magento/module-analytics/Cron/CollectData.php

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name, -n`     Name of the patch file

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--title, -t`     Title of the patch

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--force, -f`     Replace patch if it already exists


* `project:remove`   Remove project (project folder, madock project configuration, volumes, images, containers)

* `proxy:start`   Start a proxy server


* `proxy:stop`   Stop a proxy server


* `proxy:restart`   Restart a proxy server


* `proxy:rebuild`   Rebuild a proxy server


* `proxy:prune`   Prune a proxy server
                        

* `prune`   Stop and delete running project containers. For example: `madock prune --with-volumes`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--with-volumes, -v`   Remove volumes, too

* `pwa`    Execute PWA command inside node container. For example: `madock pwa yarn watch`


* `rebuild` Recreation of all containers in the project. All containers are re-created and the images from the Dockerfile are rebuilt
                        

* `remote:sync:media`  Synchronization media files from remote host. For example: `madock remote:sync:media --images-only --compress`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--images-only, -i`   Synchronization images only

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--compress, -c`      Apply lossy compression. Images will have weight equals 30% of original

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--ssh-type, -s`   SSH type (dev, stage, prod)

* `remote:sync:db`  Create and download dump of DB from remote host. For example: `madock remote:sync:db --name=local`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name, -n`  Name of the DB export file

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--ignore-table, -i`  Ignore the table when exporting. The specified table will not be included in the backup file. To specify multiple tables, specify this option multiple times.

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--ssh-type, -s`   SSH type (dev, stage, prod)


* `remote:sync:file`  Create and download dump of DB from remote host

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--path`   Path to file on server (from Magento root)

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--ssh-type, -s`   SSH type (dev, stage, prod)
                        

* `restart` Restarting all containers and services. Stop all containers and start them again


* `scope:add`   Add and activate a new config scope

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[scope name]`  Scope name


* `scope:list`   Show all config scopes


* `scope:set`   Set config scope

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[scope name]`  Scope name


* `service:list`   Show all services


* `service:enable`   Enable the service. For example: `madock service:enable phpmyadmin`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name]`  Service name
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--global, -g`  Enable the service globally


* `service:disable`   Disable the service. For example: `madock service:disable phpmyadmin` 

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name]`  Service name
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--global, -g`  Disable the service globally
                        

* `setup`   Initial the project setup

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--download, -d`   Download the specific Magento version from Composer to the container

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--install, -i`   Install Magento from the source code

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--sample-data, -s`   Install Magento Sample Data          

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--platform`   Platform (magento2, shopify, pwa, custom, etc.)                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--platform-edition`   Platform edition (community or enterprise for Magento 2)                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--platform-version`   Platform version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--php`   PHP version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--db`   DB version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--composer`   Composer version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--search-engine`   Search Engine                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--elastic`   Elasticsearch version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--opensearch`   OpenSearch version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--redis`   Redis version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--rabbitmq`   RabbitMQ version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--hosts`   Hosts                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--nodejs`   Node.js version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--yarn`   Yarn version                            

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--pwa-backend-url`   PWA backend url                         

* `setup:env`   Generate app/etc/env.php

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`-f`   Force re-create the file

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--host, -h`   Default host


* `shopify` or `sy`   Execute the Shopify command inside the php container. For example: `madock shopify yarn create @shopify/app --template php`


* `shopify:web` or `sy:w`   Execute the Shopify command inside the php container in 'web' folder. For example: `madock shopify:web composer install`


* `shopify:web:frontend` or `sy:w:f`   Execute the Shopify command inside the php container in 'web/frontend' folder. For example: `madock shopify:web:frontend SHOPIFY_API_KEY=REPLACE_ME yarn build`


* `snapshot:create`   To create a snapshot of the project. The snapshot will include databases and project files

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name, -n`  Name of the snapshot


* `snapshot:restore`   To restore a project of the snapshot. The databases and project files will be restored


* `ssl:rebuild`   Rebuild SSL Certificates  
                        

* `start`   Starting all containers and services
                        

* `status`   Display the status of the project
                        

* `stop`    Stopping all containers and services
                        

* `uncompress`  Uncompress the project from archive