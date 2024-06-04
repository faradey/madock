# VSCODE + Xdebug Setup

## Enable Xdebug

Xdebug needs to be enabled inside the `phpfpm` container.

```
madock debug:enable
```

## VSCODE configuration

1. `Create the file within your project .vscode/launch.json`
    * Debug Ports: 9001 and 9003

2. `Add the following configs`

```
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Listen for XDebug",
      "type": "php",
      "request": "launch",
      "port": 9003,
      "pathMappings": {
        "/var/www/html": "${workspaceFolder}"
      },
      "log": true,
      "xdebugSettings": {
        "max_children": 128,
        "max_data": 512,
        "max_depth": 4
      }
    }
  ]
}
```

3. Start Listening for PHP Debug connections

   **NOTE**: Be sure to activate that only after setting the right debug port. Changes in Debug port are ignored once the listener has started.

4. If you are using profiling, you can find the profile files in the <project>/var folder
