# Workflow

The following guide shows you the normal development workflow using madock.

#### 1. Start containers

```
madock start
```

#### 2. Install/update dependecies with composer

```
madock composer <install/update>
```

#### 3. Develop code normally inside `magento/app`

While developing you might need to execute magento commands like `cache:flush` for example

```
madock magento <command>
```

#### 4. Working on frontend

```
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