**Clone project**

To clone the existing project execute the command:
```
madock project:clone --name={project_name} --domain-suffix=-copy
```

`--domain-suffix` is inserted before the last dot of every host of the clone, so
`shop.test` becomes `shop-copy.test` and the two projects do not collide. The
source project is stopped for the length of the copy and started again.

A refusal — a name that already exists, a name or suffix with a period or a
space, a project managed by deployer — exits 1.

`--json` (`-j`) answers `{"name": "…", "path": "…", "hosts": ["…"]}` on stdout,
with the progress on stderr. See [JSON output](json_output.md).