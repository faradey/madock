**Custom commands**

To use custom commands, you need to add a similar code section in the file
`madock/projects/config.xml`:

```xml
<custom_commands>
    <ci>
        <alias>c:i</alias>
        <origin>madock composer install _args_</origin>
    </ci>
    <ls>
        <alias>ls</alias>
        <origin>madock cli ls</origin>
    </ls>
</custom_commands>
```

Where:
- `alias` is the command you type in the command line.
- `origin` is the command that will be executed.
- `_args_` is a tag that will be replaced with the command-line arguments provided after the main command.

You can also add unique custom commands for each project separately. 
To do this, add the commands to the project's config file located at `madock/projects/{project_name}/config.xml`.