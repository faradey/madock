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


* `composer`  Execute composer inside php container
            
            
* `compress`  Compress a project to archive
            
            
* `config`  Viewing and changing the project configuration

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`show`    List all project environment settings

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`set`     Set parameters

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--hosts` Domains and code of project websites. Separated by commas. For example: one.example.com:base two.example.com:two_code. Optional
               
         
* `cron:enable`    Enable cron


* `cron:disable`    Disable cron
              
          
* `db:import`      Import database

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`-f`  Forse mode


* `db:export`      Export database


* `db:info`      Information about credentials and remote host and port
                     
   
* `debug:enable`   Enable xdebug


* `debug:disable`   Disable xdebug
                     
   
* `info`   Show information about third-parties modules (name, current version, latest version, status)             
    
    
* `help`    Displays help for commands
                      
  
* `logs`    View logs of the container

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
                        

* `remote:sync:media`  Synchronization media files from remote host

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--images-only`   Synchronization images only

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--compress`      Apply lossy compression. Images will have weight equals 30% of original

* `remote:sync:db`  Create and download dump of DB from remote host


* `remote:sync:file`  Create and download dump of DB from remote host 

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--path`   Path to file on server (from Magento root)
                        

* `restart` Restarting all containers and services. Stop all containers and start them again
                        

* `service`   Services

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`list`   Show all services

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name] on`   Enable the service

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`[service name] off`  Disable the service
                        

* `setup`   Initial the project setup

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--download`   Download the specific Magento version from Composer to the container

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;`--install`   Install Magento from the source code
                        

* `ssl:rebuild`   Rebuild SSL Certificates  
                        

* `start`   Starting all containers and services
                        

* `status`   Display the status of the project
                        

* `stop`    Stopping all containers and services
                        

* `uncompress`  Uncompress the project from archive