# Workflow

The following guide shows you the normal development workflow using madock.

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

**IMPORTANT:** For the Chrome browser, you can download the LiveReload plugin specifically for madock from the link [Google Chrome plugin](https://chrome.google.com/webstore/detail/livereload-for-madock/cmablbpbnbbgmakinefjgmgpolfahdbo). Then install it and enable it for the site you need.

**NOTE:** You might also need to disable your browser cache. For example in Chrome:

* `Open inspector > Settings > Network > Disable cache (while DevTools is open)`

#### 5. xdebug

* Enable xdebug

  ```
  madock debug on
  ```

* Configure xdebug in PHPStorm (Only first time)

    * [PHPStorm + Xdebug Setup](./xdebug_phpstorm.md)

* Disable xdebug when finish

  ```
  madock debug off
  ```

#### 6. SSL certificates

If you want to manually add an ssl certificate to the browser, you can find it at [path to madock folder]/aruntime/ctx/madockCA.pem

#### 7. help
```
  madock help
 ```

This command shows you the following items:

* `bash`    Connect into container using bash

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]` Name of container. Optional. Default container: php. For example: php, node, db, nginx

* `c:f`  Cleaning up static and generated files


* `cloud`  Executing commands to work with Magento Cloud. Also, can be used the long command: magento-cloud)


* `composer`  Execute composer inside php container
            
            
* `compress`  Compress a project to archive
            
            
* `config:list`  List all project environment settings


* `config:set`  Set a new value for parameter

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name`     Parameter name

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--value`     Parameter value
               
         
* `cron:enable`    Enable cron


* `cron:disable`    Disable cron
              
          
* `db:import`      Import database

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`-f`  Forse mode


* `db:export`      Export database

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name`  Name of the DB export file


* `db:info`      Information about credentials and remote host and port
                     
   
* `debug:enable`   Enable xdebug


* `debug:disable`   Disable xdebug
                     
   
* `info`   Show information about third-parties modules (name, current version, latest version, status)             
    
    
* `help`    Displays help for commands
                      
  
* `logs`    View logs of the container

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]`     Container name. Optional. Default container: php. Example: php
                        

* `magento` or `m` Execute Magento command inside php container.
                        

* `node`    Execute NodeJs command inside php container
                        

* `patch:create`   Create patch. The patch can be used with the composer plugin cweagans/composer-patches

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--file`     Path of changed file. For example: vendor/magento/module-analytics/Cron/CollectData.php

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name`     Name of the patch file

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--title`     Title of the patch

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--force`     Replace patch if it already exists


* `proxy:start`   Start a proxy server


* `proxy:stop`   Stop a proxy server


* `proxy:restart`   Restart a proxy server


* `proxy:rebuild`   Rebuild a proxy server


* `proxy:prune`   Prune a proxy server
                        

* `prune`   Stop and delete running project containers
                        

* `rebuild` Recreation of all containers in the project. All containers are re-created and the images from the Dockerfile are rebuilt
                        

* `remote:sync:media`  Synchronization media files from remote host

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--images-only`   Synchronization images only

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--compress`      Apply lossy compression. Images will have weight equals 30% of original

* `remote:sync:db`  Create and download dump of DB from remote host

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--name`  Name of the DB export file


* `remote:sync:file`  Create and download dump of DB from remote host 

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--path`   Path to file on server (from Magento root)
                        

* `restart` Restarting all containers and services. Stop all containers and start them again
                        

* `service:list`   Show all services


* `service:enable`   Enable the service

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name]`  Service name


* `service:disable`   Disable the service  

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name]`  Service name
                        

* `setup`   Initial the project setup

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--download`   Download the specific Magento version from Composer to the container

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--install`   Install Magento from the source code
                        

* `setup:env`   Generate app/etc/env.php

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`-f`   Force re-create the file
                        

* `ssl:rebuild`   Rebuild SSL Certificates  
                        

* `start`   Starting all containers and services
                        

* `status`   Display the status of the project
                        

* `stop`    Stopping all containers and services
                        

* `uncompress`  Uncompress the project from archive