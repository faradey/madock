**Clone project**

To clone the existing project execute the command:
```
madock project:clone --name={project_name} --domain-suffix=-copy
```

`--domain-suffix` is inserted before the last dot of every host of the clone, so
`shop.test` becomes `shop-copy.test` and the two projects do not collide. The
source project is stopped for the length of the copy and started again.

**From a snapshot.** `--from=<snapshot>` makes the clone from a saved state
instead of from the project as it is now:
```
madock project:clone --name=shop-demo --domain-suffix=-demo --from=golden
```
The source is not stopped and not read; the clone gets the snapshot's
databases, named volumes and files. The snapshot is found as `snapshot:restore
--name` finds it. Its archives are hard-linked into the clone where the
filesystem allows, so they are not copied twice. A snapshot taken by an engine
(`snapshot/engine`) is cloned by that engine if it can clone from a snapshot,
and refused before anything is created if it cannot. The clone's configuration
is the source's current one.

**Muted.** `--muted` makes a copy that cannot act as its source. It sets
`clone/muted` on the clone, which means:

- cron is off whatever `cron/enabled` says, and `cron:enable` refuses;
- PHP's `mail()` goes to mailpit whatever `php/sendmail` says;
- with `rabbitmq/persist`, every queue of the clone's broker is emptied at clone
  time — the queues stay, the source's messages do not.

What madock cannot see is not covered: an SMTP server, a payment callback or a
webhook configured inside the application's own database. Give the clone its
own addresses and credentials for those.

**No start.** `--no-start` writes the clone's files and restores its data, and
creates no container of the clone. Run `madock start` in the clone to create and
start its containers:
```
madock project:clone --name=shop-demo --domain-suffix=-demo --from=golden --no-start
```
Without the flag the copy creates the clone's containers and removes them again
before it returns; `madock start` creates them anew. With `--muted` and a
persisted RabbitMQ the broker is started for the purge; with `--no-start` its
container is then removed, so `madock start` creates the whole stack.

A refusal — a name that already exists, a name or suffix with a period or a
space, a project managed by deployer, a `--from` snapshot that does not exist —
exits 1.

`--json` (`-j`) answers
`{"success": true, "data": {"name": "…", "path": "…", "hosts": ["…"]}}` on
stdout, with the progress on stderr; a failure is `{"success": false, "error":
"…"}` carrying docker's message, and the half-made clone is removed. See
[JSON output](json_output.md).