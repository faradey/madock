The functionality of scopes will assist you in utilizing multiple environments for a single project.  
For instance, suppose you have two branches, 'dev' and 'dev-upgrade'.  
The 'dev' branch is intended for delivering updates and fixes to the master branch, while the 'dev-upgrade' branch is used for upgrading your project to a new major version of your platform.
Consequently, these branches may have different versions of PHP, MySQL, Redis, Elasticsearch, etc.  
By employing two distinct scopes, you can configure two separate environments for one project.  
When switching branches, you can simply change the scope and execute the `madock rebuild` command to apply the new settings.  
The database for each scope will also be different.  
This functionality will be particularly useful in the case of a major platform update, such as upgrading Magento from version 2.3 to version 2.4

The default scope is "default"

To add a new scope or switch to another scope, execute the command `madock scope:set {scope_name}`

To display a list of all scopes for the project, use the command `madock scope:list`


Scope settings are inherited in this order  
madock/projects/{project_name}/config.xml {current scope} - the current scope you created  
madock/projects/{project_name}/config.xml {default scope}  
madock/projects/{project_name}/config.xml  
madock/projects/config.xml  
madock/config.xml