# Testing

Three layers, each answering a different question.

| Layer | Where | Question | Cost |
|---|---|---|---|
| Unit and integration | beside the code | does this function behave | milliseconds |
| Golden | `src/helper/configs/aruntime/project` | does a config render into the files we expect | seconds |
| End-to-end | `test/e2e` | does any of it actually work | ~2 minutes |

```bash
go test -count=1 ./...        # the first two layers
./test/e2e/e2e.sh up          # build the VM, once
./test/e2e/e2e.sh run         # the third
```

## Always pass `-count=1`

Go caches a package's test result until one of its `.go` files changes. Both the
golden and the end-to-end tests are driven by things the cache cannot see —
templates under `docker/`, a compiled binary, a Docker daemon. Without the flag
they report `ok (cached)` after the thing they test has been broken.

This is measured, not theoretical: a deliberately broken entrypoint template
passed a cached run and failed immediately with the flag. The pre-push hook
passes it for this reason, and so should you.

## Golden tests

They render a project's whole docker configuration and compare it, file by
file, against a committed copy. Seven project shapes, about 150 files.

Update them when the output changes on purpose:

```bash
go test -count=1 ./src/helper/configs/aruntime/project/... -run Golden -update
```

Then read the diff like any other change. A golden file updated without being
read is worse than no test — it records whatever the code does today and calls
it correct. That is not hypothetical either: the first version of the
`custom`/`none` case was generated from a config missing `app/enabled`, so the
"expected" output had no main service in it at all.

The point of a golden file over an assertion is that it fails on changes nobody
thought to assert. An assertion answers the question you asked; a golden file
answers the one you did not.

## End-to-end tests

They run the real binary against a real Docker daemon: create a project, start
it, ask whether it is running, talk to its database, stop it, remove it.

**They do not run on your machine, and cannot be made to.** madock's proxy is a
single compose stack named `aruntime` on a network named `madock-proxy`, and
both names are written into the templates rather than derived from anything. A
test that starts a project therefore does not take a port from your work — it
operates on the very same containers. `MADOCK_EXEC_DIR` does not help: it moves
the files, not the container names.

So they run in a [Lima](https://lima-vm.org) VM with a Docker daemon nobody else
is using. Ubuntu, because that is what the servers we deploy to run.

```bash
brew install lima

./test/e2e/e2e.sh up      # build the VM (a few minutes, once)
./test/e2e/e2e.sh run     # build madock for linux, run the suite inside
./test/e2e/e2e.sh run -run TestDatabaseIsReachable   # narrow it
./test/e2e/e2e.sh shell   # a shell in the VM
./test/e2e/e2e.sh down    # shut it down, keep the disk
./test/e2e/e2e.sh reset   # delete it and build it again
```

The suite sits behind the `e2e` build tag, so `go test ./...` does not compile
it and the pre-push hook stays fast. The hook does run `go vet -tags=e2e
./test/...`, so a test file that no longer compiles is caught without a VM.

### What they are for

Not coverage. They exist to answer the question the other layers structurally
cannot: whether the generated files start anything. The golden tests prove a
compose file is written; only running it proves the container comes up, the
database answers, and the command reports honestly about both.

The first run of the very first test found two defects that had been in the code
for a while:

- `setup -y` was not actually non-interactive. It reached the SSL step, found no
  `certutil`, and stopped at a prompt — which in a provisioning script is a hang
  nobody is there to answer.
- the debug log was written next to the binary, ignoring `MADOCK_EXEC_DIR`, and
  a failure to write it ended the command. An installation directory without
  write permission turned every error into `open debug.log: read-only file
  system`.

Both are the kind of thing that is invisible to a unit test and obvious the
first time a machine runs the binary the way a customer does.

### Writing one

Take a project from `newProject`, drive it with `run`, assert on what comes
back. Cleanup is registered before anything starts, so a test that fails halfway
still takes its containers down.

```go
func TestSomething(t *testing.T) {
	p := newProject(t, "e2ething")

	p.run(3*time.Minute, "setup", "-y",
		"--platform=custom", "--language=none", "--hosts=e2ething.test")
	p.run(20*time.Minute, "start")

	out := p.run(time.Minute, "status")
	requireContains(t, out, "db running", "after start")
}
```

Two habits worth keeping:

- **`custom` with `--language=none` unless the test is about a platform.** No PHP
  image to build, no Magento to install: the whole lifecycle in twenty seconds.
- **Give each test its own project name.** `MADOCK_EXEC_DIR` isolates the files,
  not the container names, so two tests sharing a name share containers.
