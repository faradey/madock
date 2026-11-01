# Working in this container

- This container holds the project's files and nothing else. php, composer,
  go, python, ruby and the project's own scripts run in the project's main
  container through commands of the same name, in the same directory.
  `madock-exec <command> [args...]` runs any other command there.
- Work only inside the project directory.
- After every edit, check it: a syntax check of the file you changed, and the
  project's tests if it has them. Fix what fails before saying the task is done.
- Say plainly when something failed or was not checked.
- Do not commit or push unless asked.
