# PHPStorm + Xdebug Setup

## Enable Xdebug

Xdebug needs to be enabled inside the `phpfpm` container.

```
madock debug:enable
```

## PHPStorm configuration

1. `PHPStorm > Preferences > PHP > Debug`
    * Debug Ports: 9001 and 9003

2. `PHPStorm > Preferences > PHP > Servers`

    * Name: `[your-domain].[xxx]` (for example world.test)
    * Port: 80
    * Mapping: `/Users/<username>/Sites/<project> -> /var/www/html`

3. Start Listening for PHP Debug connections

   **NOTE**: Be sure to activate that only after setting the right debug port. Changes in Debug port are ignored once the listener has started.

4. If you are using profiling, you can find the profile files in the <project>/var folder
	
	
	