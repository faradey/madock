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

#### 6. help
```
  madock help
 ```

This command shows you the following items:

* `bash`    Connect into container using bash

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]` Name of container. Optional. Default container: php. For example: php, node, db, nginx

* `c:f`  Cleaning up static and generated files


* `composer`  Execute composer inside php container
            
            
* `compress`  Compress a project to archive
            
            
* `config`  Viewing and changing the project configuration

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`show`    List all project environment settings

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`set`     Set parameters

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--hosts` Domains and code of project websites. Separated by commas. For example: one.example.com:base two.example.com:two_code. Optional
               
         
* `cron`    Enable / disable cron

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`on`  Enable cron

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`off`   Disable cron
              
          
* `db`      Database import / export

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`import`  Database import

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`export`  Database export

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`soft-clean`      Soft cleanup of the database from unnecessary garbage.
                     
   
* `debug`   Enable / disable xdebug

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`on`      Enable xdebug

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`off`     Disable xdebug
                    
    
* `help`    Displays help for commands
                      
  
* `logs`    View logs of a container

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[name of container]`     Container name. Optional. Default container: php. Example: php
                        

* `magento` Execute Magento command inside php container
                        

* `node`    Execute NodeJs command inside php container
                        

* `proxy`   Actions on the proxy server
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`start`   Start a proxy server

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`stop`    Stop a proxy server

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`restart` Restart a proxy server 

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`rebuild` Rebuild a proxy server

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`prune`   Prune a proxy server
                        

* `prune`   Stop and delete running project containers
                        

* `rebuild` Recreation of all containers in the project. All containers are re-created and the images from the Dockerfile are rebuilt
                        

* `remote`  Performing actions on a remote server

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`sync`    Synchronization media, DB, etc.

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`media`   Synchronization media files from remote host

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--images-only`   Synchronization images only

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--compress`      Apply lossy compression. Images will have weight equals 30% of original

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`db`      Create and download dump of DB from remote host
                        

* `restart` Restarting all containers and services. Stop all containers and start them again
                        

* `setup`   Initial project setup
                        

* `start`   Starting all containers and services
                        

* `stop`    Stopping all containers and services
                        

* `uncompress`  Uncompress a project from archive