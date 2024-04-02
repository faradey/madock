With `madock` you can import the database from your dev site.

First, add ssh data in the project file `madock/projects/{projects_name}/config.xml`

List of ssh options
* SSH_HOST
* SSH_PORT
* SSH_USERNAME
* SSH_KEY_PATH
* SSH_SITE_ROOT_PATH
* SSH_AUTH_TYPE (key or password)
* SSH_PASSWORD (enter only if you do not use a ssh key)
* See [examples](./ssh_example.md)

Run the following commands
```
madock remote:sync:db
```
```
madock db:import
```
After the successful execution of these commands, the database will be imported.

You can access the database through phpmyadmin by going to http://{you_domain_name}/phpmyadmin

For DB exporting run command
```
madock db:export
```
The files of DB dumps keep in madock/projects/{projects_name}/backup/db
