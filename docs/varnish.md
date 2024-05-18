**Varnish cache**

To enable the varnish cache execute the command:
```
madock service:enable varnish
```
To disable the varnish cache execute the command:
```
madock service:disable varnish
```
Configuration
* Host: nginx
* Port: 80

To apply changes in the varnish configuration file default.vcl you should execute the command below
```
madock rebuild
```

To set up or change the varnish repository and version execute the following commands
```
madock config:set --name varnish/repository --value varnish
```
```
madock config:set --name varnish/version --value 7.5.0
```

To set up or change the path to default.vcl file use command (path should be from the root of the project without the first slash)
```
madock config:set --name varnish/config_file --value default.vcl
```
After any configuration changes, a project rebuild is required
```
madock rebuild
```