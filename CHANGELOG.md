**v4.1.19**

Fixed:
- **The ban on destructive commands refused the one removal that destroys nothing.** madock-pro ships `allow_destructive_commands` false on servers, and it covered `project:remove --registry-only` wholesale — so the three entries 4.1.18 taught `project:list` to name, sitting on a production host and pointing into live releases, still could not be removed. The machine that most needs the ban is the machine where such entries accumulate, and the only way out was to turn the ban off on production for the length of the cleanup
- **The exemption is by state, not by flag.** An entry whose source is gone, whose link resolves to nothing, or whose path is inside another registered project owns nothing that could be lost — a port reservation and a block in the shared proxy, both on behalf of something that does not exist. A healthy project is refused as before: its record is its madock configuration, passwords and ports and stack, and nothing recreates it. `no-path` is refused too, being a legacy entry of a project that may well still exist
- **And the exemption cannot destroy data even if an entry turns out to have some**: registry-only removal no longer passes `-v --rmi all`. Containers of that name go and come back from the configuration; volumes and images stay, findable by `prune` and the orphans command

**v4.1.18**

Fixed:
- **`project:remove` said what it would delete, and the sentence was true of neither case.** The directory it deletes is `os.Getwd()`, which keeps whatever symlinks the shell walked through, so on a Deployer layout one line typed two ways destroys two different things — measured on a `releases/N` + `current` fixture: a plain `cd .../current` unlinks `current` and leaves the releases, `cd -P .../current` takes the whole release directory. Both printed the same warning naming `.../current`. The scale of the deletion was a property of the shell, not of the command, and nothing said so. The confirmation now names the path the recursive delete will actually be given, and says when only a symlink goes
- **A registry entry could be registered inside another project, and every check written for a bad entry called it healthy.** The project name is the name of the working directory, so madock run inside a release registers `current`; three such entries were found on one production host — `current`, `current-2` and `current-3`, pointing into three different applications — holding ports and blocks in the shared proxy on behalf of projects that already had entries of their own. `project:list --stale` answered "Every registry entry resolves" about all three, because the path existed. There is a state for it now, `project:list` names the owning project, and `--stale` reports it
- **Removing such an entry was impossible, and the advice for it was dangerous.** `project:remove --name` refuses while the source directory exists and told the caller to `cd` there and run it with `--force` — which for these entries is a live release of somebody else's application, and the command ends with a recursive delete of the working directory. `--registry-only` removes what the installation holds — entry, runtime, proxy block, ports, containers — and cannot reach a source directory at all. The refusal now offers it, and for a nested entry the `cd` advice is not printed
- **The nesting is refused at the moment it would be created.** A directory that resolves inside an already-registered project, or that is itself a symlink, no longer becomes a project: the first has an entry already, and the second records a path that the next deploy repoints while every command that reads it keeps reporting success. Only where a *new* entry would be written — an installation that already carries one keeps working
- **Removing an entry by name no longer chowns the caller's working directory.** The reclaim step that hands root-written files back mounts `os.Getwd()` into a container and runs `chown -R` when the project's own container is gone. That is the project being deleted when the command runs in it, and the caller's unrelated directory when it does not

**v4.1.17**

Changed:
-  says the marker phrase itself rather than pointing at its neighbour. A cross-reference is not what a search finds, and being findable is the whole purpose of the phrase

**v4.1.16**

Changed:
- **Fifteen functions now say they exist to be called from madock-pro.** A reachability analysis over this module alone reports 61 unreachable functions, and fifteen of them are what pro is built on — the hook registry, the licence gate, the template trees pro installs over these, the password generator that replaces this module weak defaults on every server, the guard on destructive commands. Every one is genuinely unreachable from here, because being called from the other side is the point; read without that context they are dead code with tidy comments, and an audit proposed deleting some

**v4.1.15**

Fixed:
- **A node preset defaults, and no longer overrules.** Three lines of the same `if/else` disagreed: the php branch reads `redis/enabled` from the project and says in a comment that it respects an explicit disable, while the node branch replaced whatever the project had asked for, in silence, along with `db/type`. Catalyst and Stencil need neither, so `false` and empty are the right defaults — but a project that wrote something there wrote it on purpose
- It stopped being theoretical when the BigCommerce line was decided: its Core is Node with Prisma and BullMQ, so it needs both a database and Redis, and this branch forbade exactly what that stack requires. The workaround was `platform custom` with the stack spelled out by hand

Changed:
- **A failed TLS handshake in the end-to-end suite now asks the proxy what it was doing.** Two tests assert that handshake and both have failed on a runner with the same sentence — nothing answered on 443 — which is three situations wearing one message. `status` and `proxy:logs` run after the failure, so neither can turn a pass into a failure and neither is an assertion

**v4.1.14**

Fixed:
- **The cron entrypoint reaches the Node image, which is the side that actually failed.** 4.1.13 put it in the php image, and a Shopify or Medusa project has no php container at all — its application runs in the node one, and the production machine that came back from a reboot with no scheduler was running two Node apps. The php test passed throughout, on an image the failure never touched; what found it was setting up a project, installing a job, restarting the container and looking
- Two conditions the node entrypoint imposes and php does not: it runs under `set -e`, so the guard cannot be allowed to end the container, and cron is opt-in per platform there, so `crontab` may be absent from the image entirely

**v4.1.13**

Fixed:
- **Cron did not come back when a container did.** Nothing in an application image starts it — the CMD is php-fpm — so the daemon existed only for as long as the container it was started in, and anything that recreates that container takes it away and **leaves the crontab**: the jobs are there, nothing reads them, nothing fails, every check stays green. Measured on a production host rebooted for a plan change on 2026-08-30: containers up, applications answering, cron down in both projects that had it, carrying billing periods, carrier charges, storage counts and every Shopify sync
- **The image starts cron now, not madock.** Re-arming after `service:restart` was already there and could never cover this: on a host reboot Docker brings the containers back on its own and madock is not running at all
- **The signal is the crontab, not the config.** A config value is baked in when the image is built, so enabling cron and rebooting without a rebuild would produce the same silence again. Jobs in the crontab are true at the moment the container starts, and `cron:disable` removes them, so a project told to stop has none. A Debian crontab is never empty — it ships an explanatory header — so the check is for a line that is neither blank nor a comment, and there is a test for exactly that case
- Failing to start cron never blocks the application: a container that refuses to serve because its scheduler is unhappy is worse than the silence being fixed

Changed:
- **Built on Go 1.27.** Go supports a line until two newer ones exist, so 1.25 stopped receiving fixes on 2026-08-19, the day 1.27 shipped — eleven days before this was noticed, because the check meant to catch it ended its calendar a release early and an empty end-of-life date reads as supported. Nothing needs installing: `GOTOOLCHAIN=auto` fetches what `go.mod` requires and CI reads the same file
- The php image's CMD is exec form. With an entrypoint in place the shell form would have been wrapped in another `/bin/sh`, and the direct exec is what keeps php-fpm as the process that receives signals

**v4.1.12**

Changed:
- **Six more images are named in the config**, which 4.1.11 said were already there and were not. The `grafana` block named one image while its stack is six containers: loki, promtail (in two snippets), prometheus, the mysqld exporter and the rabbitmq exporter lived in their compose files, so a reader checking the config would have concluded grafana was a single image. **Three of them carried no tag at all** — `latest` spelled invisibly, and worse than writing it, because nothing in the file suggests a version was ever decided
- The project's own nginx was a literal `FROM` in its Dockerfile. It is a different container from the shared proxy, whose version moved in 4.1.11, and the two read alike enough that moving one was reported as moving both
- **Pinned at what runs, not at what is newest.** loki and promtail stay on 2.9.10 — 3.x is a major move and belongs in a change that says so — and the three untagged ones are pinned at whatever `latest` resolved to on 2026-08-29, so nothing changes today and the next move becomes a decision

**v4.1.11**

Fixed:
- **`docker compose pull` was asking the registry about images the machine already had.** It asks about every image whether or not it is present, and madock called it on the first start of every installation. Measured in CI on 2026-08-29: the image had just been loaded from a local cache, the log said `Loaded image:`, and compose still reported `Pulling` and then failed on Docker Hub's rate limit
- **Told apart by what the run is for, which is what the 2021 commit that added the pull meant.** Its message says so: "add pulling images from docker hub with rebuild command run". A rebuild does mean go and look for a newer image; creating containers, recreating them after a config change and recovering from a failed `compose start` do not — there the image only has to be present. Rebuild and clone keep the old behaviour, everything else asks only for what is missing
- The flag that makes this possible arrived in compose v2.22.0, and madock declares no minimum compose version anywhere, so the version is asked for rather than assumed. An unknown flag is not a degradation — compose exits non-zero and `madock start` dies — so where the version cannot be read, the old behaviour stands

Changed:
- **The mailpit image is named in the config like every other service** (`proxy/mailpit/repository` and `version`), so it can be raised in one place, per machine or per project, instead of being editable only by changing madock. It was `axllent/mailpit:latest` and is now pinned: the proxy is shared by every project on a machine, and a floating tag means it changes underneath all of them on whatever day upstream publishes

**v4.1.10**

Fixed:
- **The cron check cried wolf on every healthy deploy.** `cron:remove && cron:install && cron:run` exits 1 in the ordinary case on every deploy after the first: `cron:remove` leaves the last block in the crontab whatever it reports, so `cron:install` finds one already there, prints "Crontab has already been generated and saved" and stops the chain. That exit code was read as failure, and "Magento cron setup failed — scheduled jobs may NOT run" went out on deploys where cron was fine. Measured on extmag.com, release 174: the crontab held exactly one `cron:run` naming that release, `cron` was running by name, a job had completed successfully 23 seconds earlier, and the alarm printed four times in one day
- **The setup is judged by what is installed now, not by how the sequence exited.** The crontab is read back and matched against the base path the project resolves to in the container; a workdir that cannot be resolved says so rather than being rounded to "fine"
- **The two messages are separated, and that is most of the fix.** "The old entry could not be removed" is Magento behaving as it always has and belongs in the log; "cron is not installed" is an outage and belongs on the screen. One text for both is what made the alarm meaningless
- This is the second wrong answer from this probe, in the opposite direction. The first — `service cron status` reporting a daemon that was gone — let production run six hours with nothing scheduled. Crying wolf is the more expensive of the two: it teaches people to look away from the one occasion the check is right

**v4.1.9**

Fixed:
- **A project with `db/enabled=false` was still given a database image.** `MakeDBDockerfile` is called from twelve places and only two of them checked whether the project has a database; with the database off there is no version to render — the platform default supplies the repository and nothing supplies the tag — so the file came out as `FROM mariadb:`, which is not a valid reference and would not build
- It broke nothing, which is why it lasted: compose renders no `db` service for a disabled database, so nothing ever built the file. What it cost was a reader's time. Found on the BigCommerce cluster VM while chasing something else, where two cluster consumers — projects whose database comes from the provider — each carried one, and it reads exactly like a broken image until you establish that neither has a database at all
- The gate is in `MakeDBDockerfile` rather than at the call sites, because the call sites are what got it wrong. The `mariadb` in it was never the defect: that is the BigCommerce platform default, not a global one

**v4.1.8**

Fixed:
- **The node container set `NODE_ENV=development`, so `next build` built a production bundle in development mode.** Next obeys the variable instead of setting `production` itself, and React's internals then disagree with themselves: prerendering `/_global-error` dies on `useContext` with a null dispatcher. No Next application in madock could be built for production
- The cost was in the finding. Next prints `You are using a non-standard "NODE_ENV" value` as the first line of every build, and that line went past about ten times — a warning printed always is indistinguishable from noise until it is the cause. React 19.1 against 19.2, Node 22 against 24, Next 16.2 against 16.3, styled-components, `transpilePackages`, the root layout and a custom error page were all ruled out first; an application of two files with none of our dependencies failed the same way, which is what finally pointed at the environment
- **madock needs the value, but not the name.** It decides one thing — whether the entrypoint starts `dev` or `start` when `nodejs/script` is not configured — and under the standard name that private choice was published to every tool in the container. It travels as `MADOCK_NODE_ENV` now; a project's own `NODE_ENV` is still honoured, so anything that set it deliberately keeps working
- **The same line was hardcoded in the Medusa and Spree storefronts**, both of which run Next, so two shipped platforms had the defect as well. Removed there: `next dev` and `next build` each decide for themselves. `sylius-encore` is left alone — Encore takes its mode from its command, and there is no measurement for it
- The test is in the container rather than in the rendered file, because the question is what the container ends up with. It checks both directions: `NODE_ENV` is unset, and `MADOCK_NODE_ENV` is there — without the second, the entrypoint could no longer tell `dev` from `start` and the fix would have broken starting instead

**v4.1.7**

Added:
- **`project:orphans`** — what is left on this machine of projects the registry no longer knows. A project's volume outlives its directory, and after that nothing named it: it is not in the registry, `project:list` reads the registry, and `project:remove` needs a registry entry there is none of. On a server, where destructive commands are switched off as shipped, the only cleanup available was to delete the directories and leave the volumes — turning litter that can be seen into litter that cannot. Found a real one on the machine it was written on the first time it ran: three volumes belonging to a project retired long ago
- **It removes nothing and prints the command rather than running it.** Deleting these stays with `project:remove`, behind `allow_destructive_commands`. A `--remove` flag here would be a second route past that switch, and the switch is what stops a habit from becoming an accident
- Only what madock made: the compose label is on every stack on the machine, and reporting somebody else's would be calling a stranger's disk our mess. A name the registry still holds is left alone even when madock cannot read its entry — that is `project:list --stale`'s business, and two commands arguing about one thing is how both stop being believed
- `--json` for tooling, and a non-zero exit when something is found, the way `project:list --stale` answers

**v4.1.6**

Fixed:
- **A registry entry that resolves to nothing was invisible to the command written to find it.** `projects/<name>` is a symlink wherever a project was set up from a temporary checkout, and when the target goes — a `/tmp` directory a reboot cleared — the entry was skipped, on the reasoning that an entry whose own directory is gone is not an entry. True, and the wrong conclusion: the name is still in the registry and nothing said so. Measured on the BigCommerce cluster VM on 2026-08-27, where four such entries sat: `project:list` answered `No projects are registered in this installation` and `project:list --stale`, which exists for exactly "the source is gone", answered `Every registered project still has its source directory`. A confident wrong answer from the check written for the case, and the one wrong answer that reads as good news
- Such an entry is now reported as `broken-link`, and the listing names what it points at — usually a temporary directory, which tells the reader what happened without a search
- **A directory with no `config.xml` is still skipped, deliberately.** It was tempting to report those too, since a madock run outside a project leaves one named after the current directory. But `projects/` legitimately holds support directories beside the projects — `bin`, `docker` and `assets` on the installation this was checked on — so reporting them would put three lines of noise on every machine, and the existing test that pins this was right

**v4.1.5**

Fixed:
- **`db:execute --json` was accepted, did nothing, and the result was archived.** The flag is declared once for every command, on the argument struct they all embed, and `db:execute` never read it — `grep -ci json` over its file answered 0. So `madock db:execute --json "SELECT * FROM extmag_shipper_account"` printed the mysql client's ordinary TSV, and that output is in the recovery archive as `client/extmag_shipper_account.json`: not JSON, with one value in a `signature` column already corrupted by the client's escaping. The command behaved exactly as a working one does, which is the whole difficulty — a flag that is accepted reads as a flag that works
- **The obvious repair is the one that had to be avoided.** Post-processing the client's batch output, or asking the server for `JSON_ARRAYAGG`, both break on the same client: in batch mode it escapes backslashes, so the `\"` JSON uses to escape a quote arrives as `\\"` and the document will not parse. `TO_BASE64` gets the bytes past that, but MariaDB breaks base64 with a newline every 76 characters, and the aggregate is cut off at `group_concat_max_len` — 1024 bytes against a dump of 8634 on the stand where this was measured — **silently**, which is the worst of the three: the file looks like JSON, parses as JSON, and is missing most of the rows
- So nothing parses the client's text any more. MySQL and MariaDB are asked for `--xml`, a structured format the client produces itself: it needs no knowledge of the columns, so an arbitrary `SELECT *` works; NULL arrives as an attribute rather than as a word a string could also spell; and the escaping is XML's, which a parser undoes. PostgreSQL is asked for `json_agg(row_to_json(t))` and encodes the rows server-side. Values stay strings, because the client carries no types and guessing has a wrong answer that matters — a zip code, a version and an account number are all "numbers" that must not lose a leading zero
- **MongoDB refuses the flag instead of approximating it.** The argument there is JavaScript evaluated by mongosh, so producing JSON would mean wrapping it — and the wrap is a guess: a cursor needs `.toArray()` and a statement returns nothing, so it would change what some queries mean and silently empty others
- **And the flag itself is no longer accepted where nothing implements it.** It was declared globally and honoured by ten commands out of the thirty-six that take it; the rest ignored it in silence. The dispatcher now refuses `--json` on a command that does not answer in JSON, and names the ones that do. Pass-through commands are exempt — there the flag belongs to composer, npm or `bin/magento`. The default is deliberately the strict one: forgetting the mark on a command that formats JSON breaks it loudly the first time anyone runs it, while the other way round is the silence above

**v4.1.4**

Fixed:
- **One file it could not write ended the whole extraction, in silence.** `extractFS` returned `os.WriteFile`'s result straight out of the `fs.WalkDir` callback, and `fs.WalkDir` reads any non-nil answer as "stop the walk" — while `extractFS` discarded the walk's own result, so nothing was left to report. Measured on the `shopify-e2e` machine on 2026-08-27: exactly one file in the tree, `docker/snippets/docker-compose/worker.yml`, had been left owned by root by a single run under `MADOCK_USER=root` five days earlier, and every extraction since stopped there — 202 files of 309, with everything lexically after `w` in that directory missing, which took out the whole of `snippets/dockerfile/**`, `snippets/nginx/**` and three platforms. The walk now records what it could not write and finishes the tree
- **Nothing about that installation looked wrong, which is why it cost an hour.** The directories exist, because the pass before the failing write creates them; `.embedded_version` carries the running version, because it is stamped after extraction returns and extraction returned normally; `status` and `project:list` work, because they need no templates. The only symptom is a build printing the three paths it looked for a snippet in — not "the file is missing", and certainly not "the extraction was incomplete". A partial extraction that says nothing is worse than one that stops: the one that stops is fixed the same minute
- **And it did not merely leave templates unwritten — on an upgraded installation it deleted working ones.** `removeWithdrawn` deletes what the previous manifest names and this run did not write, so a truncated run reads its own gap as "the release withdrew these". Proven by test rather than argued: against the old code a template that is still shipped and was on disk is gone after one blocked write. An incomplete run now draws no conclusion from what it managed to write — nothing is removed, the manifest is left as it was, and the version is not stamped, so the next command tries again instead of declaring the tree current
- **A file that cannot be written in place is replaced outright.** Unlink permission on Unix belongs to the containing directory rather than to the file, so a root-owned template in a directory that is ours is removed and written afresh — the mines a `MADOCK_USER=root` run leaves behind are cleared by the next ordinary command instead of needing a `chown` nobody knows to run. Only on a permission error: a write that failed for any other reason has not said that replacing the file would help, and removing it first would turn "could not update this template" into "this template is gone"
- What could not be written is now named on stderr with the reason, once per command until it is fixed. stderr, because `--json` output has to stay parseable

**v4.1.3**

Fixed:
- **The stale-job check called six working jobs broken the first time it met a real crontab.** It tested every absolute-looking word in the command, and `pricer-shopify` schedules `.../poke.sh /api/cron/apply-due` — where the second word is a URL route, not a path. Only the program is checked now, and the script after it when the program is an interpreter; arguments are left alone. A check that cries wolf on a healthy project is worse than no check, because it is the one people learn to skip
- **And before that it reported nothing at all, for a reason worth writing down.** A crontab line begins with five fields that are usually `*`, and splitting it without `set -f` expands them into the names of whatever files are in the working directory — so the fields shifted and the program was never examined. Both faults were found by running the check against the machines rather than against the tests, which passed throughout

**v4.1.2**

Fixed:
- **Every deploy left the previous release's Magento cron block installed, and Magento cannot remove it.** `CrontabManager` marks its block `#~ MAGENTO START <sha256(BP)>` and finds it again by recomputing that hash from wherever it runs — so on a deployer layout, where `BP` is `…/releases/<n>`, `cron:remove` run from the new release cannot see the old release's block and reports success without touching it. Measured on extmag.com on 2026-08-27: after one deploy the crontab carried blocks for `releases/159` and `releases/160`, and `cron:run` started twice a minute out of two trees. The delayed half is worse: `deploy:cleanup` removes the old release and the entry stays, running a php that fails every minute into a shared log. madock removes those blocks itself now, after installing the current one, and prints what it removed — a schedule that quietly stops being double is worth a line
- The block to keep is the one naming the base path the container resolves `workdir` to, asked of the container rather than taken from the config: `workdir` is `…/current` on a deployed project while the block names the release the symlink pointed at when it was written. An unterminated block, or a base path that cannot be resolved, removes nothing — damage of another kind is not something to tidy away
- **`cron:status` counted a job that cannot run as a working one.** An entry naming a directory that no longer exists is counted among the installed jobs, which is exactly how the state above looked healthy while half of it was failing every minute. It is reported separately now, as `stale_jobs` in `--json` and as a problem in the exit code, checking the command rather than the whole line — a job writing into a log file that does not exist yet is healthy, and flagging it would make the check unusable on the day it matters
- **`cron:disable` said the Magento block was removed and left it installed.** It ran `bin/magento cron:remove` and trusted the answer, and that command cannot finish the job: `cleanMagentoSection` matches the END marker followed by a newline, while the crontab it matches against comes back from `Shell::execute` as `implode(PHP_EOL, $output)` — so the string ends at the marker and the **last** block in the file is never matched. Traced by the session that reported the duplicate, after ruling out both likelier explanations by measurement: the trailing newline is in the file (`od -c`), and the marker is sha256 of the right base path (compared byte for byte against the four candidates). The radius is wider than madock — `cron:remove` silently no-ops on the last block of any installation — so nothing here relies on its result any more; madock takes the blocks out itself
- Which also settles the duplicate case: `cron:install` clears the previous block through that same function, so installing over an install appends instead of replacing. A second block naming the same base path is now dropped along with the stale ones
- **"See debug.log for details" named a file the reader could not find.** The log is in the installation directory; on a live server the person looked in the project and in their home, found nothing, and worked the failure out from the state of the crontab instead. The message prints the path now, from `logger.Path()`

**v4.1.1**

Fixed:
- **The warning about an overridden `cron/enabled` sent the reader to a file a deploy replaces.** 4.1.0 started saying which copy of the config wins and pointed at `<path>/.madock/config.xml` with "edit it there". On a checkout that is the right file. Where deployer manages the project it is a symlink to `current/.madock` and resolves into `releases/<n>/`, so the edit works and is undone by the next release — worse than not working, because the value is right for a week and wrong afterwards with nothing said. The warning now names the repository, which is where that file is edited in either case, and says the path is a release when it is one. Measured on `extmag` on 2026-08-27, where turning cron off for one project was done in three files
- Which of those three decides is settled rather than assumed: `ConfigMapping` fills only keys that are **missing**, and the project's own `.madock/config.xml` is read before the installation's copy — so the release wins, and editing the runtime copy under `/opt/madock/projects/<name>/` changes nothing while the project's file names the key

**v4.1.0**

Fixed:
- **`status` reported a cron daemon that was not there, and that is what made everything below invisible.** It asked the container `service cron status`, which reaches `pidofproc` in `/lib/lsb/init-functions`; given a pidfile, that function does no more than `kill -0` on the number it holds and never checks the process is cron. `/var/run/crond.pid` lives in the container's filesystem and survives a restart, so afterwards it names a pid from the previous boot — and on a busy container that number belongs to something else. Measured on a live Node project: `ps` had no cron, `madock status` answered `Cron is running (6 jobs)`, and the six jobs were real, because the crontab survives the restart too and the count agrees with the lie. Reproduced deterministically by writing `1` — the container's own init — into the pidfile
- The probe now reads `/proc/*/comm` and matches `cron` or `crond` by name. Not `ps` or `pgrep`: neither is guaranteed to be installed in an application image, and a missing binary is indistinguishable from a dead daemon
- **Cron did not survive a restart of the container it runs in, so a deploy stopped the scheduler.** No application image starts cron — the php container's CMD is php-fpm and the Node one's is the dev server — so the daemon exists only because `start`, `rebuild` or `cron:enable` ran a command inside a container that was already up. `service:restart`, which is what a finished deploy runs, took it away and left the crontab behind: jobs installed, nothing running them, and a status that agreed. Three occurrences on one machine on 2026-08-26, the worst of them six hours old on production and found by accident. `RestartServices` now restarts cron when the service it restarted is the one the application runs in, and leaves it alone otherwise — restarting nginx has no cron in it to lose
- **`cron:enable` reported success on a project whose own config says otherwise.** It writes `cron/enabled` into the installation's copy, and a project's committed `.madock/config.xml` is merged over that — so where the file says false, cron started, "Cron was started" was printed, the setting read false immediately afterwards, and the next `start` turned it off again. The daemon really was running, which is what made it convincing. It now says which value is in effect and names the file that wins; that file is not edited, because it belongs to whoever committed it. Measured on `extmag-core-bigcommerce`
- **`cron:disable` skipped the removal when the daemon happened to be down.** The guard asked whether cron was running, but what has to be true for the removal to make sense is that the container answers: the jobs are in its crontab either way, and a project told to stop kept every job installed, ready to run the moment anything started cron again

Added:
- **`cron:status`** — the read-only question, which did not exist. `cron:enable` and `cron:disable` both act, so the only way to ask was `status`, which reports the whole project. It answers in the exit code as well as in words: `0` the configuration and the container agree, `1` a problem — enabled and not running, or running with an empty crontab — and `2` could not tell, which is never rounded up to healthy. `--json` carries `enabled`, `running`, `jobs`, `jobs_known`, `state` and `reason`

**v4.0.5**

Fixed:
- **The template-drift warning spoke on every command, in every project.** It ran from the dispatcher, so on a machine where somebody is editing templates every `db:execute`, `cli` and `setup:upgrade` in every *other* project carried a warning about a binary its user was not going to rebuild. Accurate and unusable: a warning that is always there is one that stops being read, and this one exists for a failure that took out every environment on the machine at once. Reported from a session working in another project — from inside the one doing the editing it looks like a warning about work in progress
- The hazard is rendering, so it moved to where rendering happens: `MakeConf`, which everything that generates a stack goes through and nothing that merely talks to a container does. No list of commands to keep current, which was the objection to putting it in the handlers. Once per run, because a command renders several times and the answer does not change in between. `rebuild` keeps a call of its own, in the pre-flight beside `ReportBrokenIncludes`, because it destroys the containers and generates the build context afterwards — waiting for the renderer there would say it with the environment already down
- **An empty pool value walked past the check written for it.** `ValidateFpmPool` treated "present and empty" as "absent", on the reasoning that the embedded defaults would answer. They do not: `ConfigMapping` fills only keys that are *missing*, whatever the value, so an empty one survives into the template as `pm.max_children = `, php-fpm refuses to start from inside the container after a full image build, and the other three numbers were never compared either
- **`debug:enable` promised a port nothing would listen on.** It finished by pointing at `info:ports` whenever the project had a Node container; with anything but `nodejs/script_type=command` the container starts through its package manager, the entrypoint declines to hand that the inspector, and the IDE gets "connection refused" while the explanation sits in the container log. It now says so where the switch is thrown, and names the setting that changes it
- The entrypoint's own explanation called the package manager `npm run` — `$pm` carries the subcommand — in a message whose whole job is to be understood

Changed:
- **The platform fixtures render what `setup` produces.** `writePlatformDefaults` called each platform's config writer directly and skipped the two keys the shared path writes first, `platform` and `language`. `language` decides which template tree answers when a platform ships no file of its own, so with it unset everything fell through to `docker/general/service/` — and Medusa, a Node platform, had its application image pinned as **ubuntu with `CMD php-fpm8.4`**. A real Medusa project has always got the Node image; the fixture was blessing something else, which is worse than having none: had the real resolution broken, this would have stayed green
- The version providers are blank-imported there because they register in `init()` and in the real binary arrive with the platform controllers, which cannot be imported from that package. A missing one now fails the fixture by name — the first attempt at this fix set the keys without them, `GetVersionsForPlatform` answered "no such platform", and the change did nothing at all while the suite stayed green

**v4.0.4**

Fixed:
- **Node debugging attached to the package manager, not to the application.** 4.0.3 put `NODE_OPTIONS=--inspect` on the container, on the reasoning that the dev server is a child of `yarn dev` and would inherit it. It does inherit it — and so does yarn, which is a Node program itself, as are npm and pnpm. The wrapper opened the inspector first and kept it; the dev server it spawned failed to bind the port and ran with no debugger at all, while the IDE attached happily to the package manager. The reasoning named the exact mechanism that defeats it
- The port and the intent are passed instead (`MADOCK_DEBUG_PORT`, `MADOCK_DEBUG_BREAK`), and the **entrypoint** decides, because it is the only thing that knows what it is about to start. A configured command or the bare `node` fallback gets the inspector. A package script does not: it says so, names `nodejs/script_type=command` and the alternative of putting `--inspect` in the script, and **starts the project anyway** — somebody asked for a debugger, not for the container to stop
- **`nodejs/debug/break` hung the container with nothing said.** `--inspect-brk` in the environment was inherited by the entrypoint's own `node -e` helper, which asks package.json whether a script exists; it stopped before its first line and waited for a debugger that was never coming, with its stderr going to /dev/null. The helper now runs with `NODE_OPTIONS` cleared, which also covers a `NODE_OPTIONS` somebody set themselves
- Both are covered by tests that **run the entrypoint** against fake `node` and `npm` and assert which process ended up with the inspector. A rendering test could not have caught either: the file looked right and the container did the wrong thing
- **The template-drift warning went to stdout and corrupted `--json`.** Everything in `fmtc` prints there, and on a source checkout drift is the ordinary state of a session spent editing templates — so `project:list --json`, `status --json` and `db:export --json` came back with five lines in front of the JSON, and every consumer stopped parsing at the first character. It is on stderr now
- **The php-fpm pool values are checked against each other before anything is built.** They are four settings and not four independent numbers: lower `php/fpm/max_children` to 2 and leave the spare bounds at their defaults, and php-fpm exits at start-up with `pm.max_spare_servers(3) must not be greater than pm.max_children(2)` — a good message in a bad place, arriving from inside the container after a full image rebuild. The same disagreement is visible when the file is read, so it is said there. Nothing is clamped into validity: a pool quietly adjusted to what madock thinks you meant is a configuration that says one thing and runs another

**v4.0.3**

Added:
- **`debug:enable` now debugs Node as well as PHP**, and on a project with both it turns on both rather than picking one. It was a PHP command wearing a general name: every one of its handlers wrote `php/xdebug/*`, so on a Node project it set a value nothing reads, rebuilt, and reported success — debugging simply absent, arranged by a command that said it had
- **The two debuggers work in opposite directions, and that is the whole design.** Xdebug *connects out* to the IDE, which is why PHP debugging has never needed a compose change: nothing is published and no port is allocated. `node --inspect` *listens*, so the IDE connects in — which costs a published port, and it is taken from the registry like every other one. Never a fixed 9229: that is a number the second project to start debugging would fight over
- **`madock info:ports` shows it with no change to that command**, as `nodejs_debug`. It prints every pair the registry holds, so allocating through the registry rather than writing a number into the template is what puts the port in front of the person who needs it
- `NODE_OPTIONS` rather than a flag on the command, because the entrypoint execs `yarn dev` or `npm run dev` and the dev server is a child of that — a flag on the outer command dies with the package manager, an environment variable is inherited. It binds `0.0.0.0`, not node's default loopback, which inside a container is reachable by nothing
- `nodejs/debug/break` stops the process before the first line and waits for the IDE. A separate switch and off by default: right for debugging a startup problem, and as a default it would make every debugged container look hung
- **Node inside the application container (`nodejs/embedded/enabled`) is refused, by name.** It has no container of its own and therefore nothing to publish a port from, which is the entire mechanism its debugger needs — so the command says that and points at the setting that gives it one, instead of turning on a switch that renders nothing
- The profiler stays PHP-only. `--cpu-prof` writes a file to open afterwards, which is a different command with a different answer, not this one with a second branch

Fixed:
- **The php-fpm pool had five workers and no way to change them.** `pm.max_children = 5` came from the distribution, no template touched it, and `config:list` showed nothing about fpm — so the only way to raise it was by hand inside the container, until the next rebuild took it away. It is `php/fpm/max_children` now, with `start_servers`, `min_spare_servers` and `max_spare_servers` beside it
- **Five is plenty for a CLI and short of what any modern admin panel opens at once.** Measured on the Shopware 6.7 stand: a cold administration load asks for several hundred assets and dozens of API calls together, and everything past the fifth came back 503. It does not read as slow — the admin draws its own "An unexpected error has occurred", and the order grid is empty while the same query answers 200 on its own. Half an hour went into telling that apart from a plugin defect
- **Only the cap is raised by default**, to 40 — and that half was measured rather than argued. Same 11-test suite, same stand, cap at 40 throughout, only the spare values moved: `2/1/3` gave 10 passed / 1 flaky and then 1 failed / 9 passed, `8/4/16` gave 1 failed / 9 passed. Indistinguishable, with the same test falling over on both sides, so the pre-warmed pool bought nothing while eight idle workers per project is a real memory tax on a machine running several stands. It follows from `pm = dynamic`: a burst spawns up to the cap as it arrives, and the spare bounds only decide how many wait around beforehand. Raise them to pre-warm a stand hit hard from cold
- Worth keeping for the method rather than the numbers: the first case for raising all four moved all four at once and credited the suite's result to the change. That is a coincidence wearing a measurement's clothes, and it took a second run with one variable to tell them apart
- **`madock logs php` works.** It used to answer `too many positional arguments at 'php'` — an error about the argument parser, on a command whose help says it shows container logs, and the only way to find `-s` was to ask for help you did not know you needed. The flag still works; naming the same service both ways is fine, and naming two different ones is refused rather than guessed, because a guess here comes back looking like logs and the reader concludes the other service is quiet

**v4.0.2**

Added:
- **A binary older than the templates it renders now says so, before the command runs.** In a source checkout the two arrive separately — git delivers `docker/`, a build delivers the binary — and nothing married them, so a `git pull` silently left the installation half-updated. Measured 2026-08-23 on this machine: a 3.9.17 binary built on 19 August against a 4.0.1 tree merged on 21 August, with the php snippets reorganised in between. Every rebuild ended on `failed to read dockerfile: open php.Dockerfile: no such file or directory` — **after the containers were already down**, because `rebuild` destroys them and generates the build context afterwards. Any project on the machine would have hit it on its first rebuild, not only the one that did
- Nothing could have shown it earlier: `--version` answers for the binary alone, and there is no second version beside it to disagree with. The check compares the templates compiled into the running binary against the ones on disk, so it answers the exact question — *was this binary built from these templates* — rather than asking a clock. Timestamps were the obvious implementation and the wrong one: `git checkout` rewrites mtimes on files it restored and leaves them on files it did not change
- It warns and does not stop. From source the renderer reads templates off disk, so editing one and running a command is the development loop working — refusing there would break the workflow the drift is normal in. It says how many differ, names three, and gives the `go build` line
- Silent wherever extraction owns the tree: a downloaded binary unpacks its own snapshot and stamps it, and madock-pro's installation gets `docker/` the same way because the assets belong to the imported module. Neither has a compiler to act on the warning, and in both the two cannot disagree

**v4.0.1**

Added:
- **`madock sw:cli` runs shopware-cli**, the vendor's extension tooling — `extension validate`, `extension build`, `extension zip`. It is a different program from `bin/console` and a compiled binary rather than a composer package, so it cannot arrive with the project: madock downloads it into the php image, the same way `n98-magerun` arrives in the magento2 image. Not a container of its own — the commands read and write the project's source, which is already mounted there, and nothing about the tool is a service
- Registered as a service, so `madock service:enable shopware-cli` is the short form. madock's idea of a service is wider than a compose container — `n98magerun`, `mftf`, `ioncube` and `xdebug` are in the same registry and none of them is one either
- Off by default behind `shopware/cli/enabled`, with the release pinned by `shopware/cli/version`, so every machine building the project gets the same tool. Running it without enabling it first refuses and names the key and the rebuild, rather than failing with "command not found" — which reads as a broken installation rather than as a setting nobody turned on
- The archive name is the vendor's own spelling and does not match `uname -m`: x86_64 is `Linux_x86_64` but aarch64 is `Linux_arm64`. Mapped rather than interpolated, so an unknown architecture fails the build with a name instead of fetching a 404 and unpacking nothing
- **`project:list --running` answers which projects are up**, and the plain list gains the column. There was no way to ask: the JSON carried `name`, `path` and `state`, where state answers a different question (`ok` / `missing-source`), so finding out who was eating memory meant walking the registry and running `status` in each project. Found while a Shopware administration build kept dying on OOM — Vite wants about 4 GB in an 8 GB machine — where the first question, "what else is running", had no command behind it
- One `docker ps` for the whole registry rather than a status call per project, reusing what `stop` already does to decide whether the shared proxy is still needed
- **Docker unavailable is `null`, not `false`.** "No projects are running" and "nobody could find out" are different facts, and `--running` refuses rather than printing an empty list when the answer is unknown

Fixed:
- **`setup -y` asked a question into a closed stdin and died on EOF.** `--yes` means "do not ask", and every other question honours it — the selectors take the configured default, or the first real option. The platform-version prompts are asked only when nothing supplied a version (no preset, nothing detected in the project, no default in the configuration), and there `--yes` had nothing to fall back on: it asked anyway and ended the command with `logger.go:82: EOF`, which names neither the question nor the flag that answers it. A scripted `madock setup -y --platform=shopware` died there every time
- It refuses and names `--platform-version`, before the handler downloads anything, so nothing is left to clean up. `Waiter` refuses too, so a prompt added later without going through `RequireAnswer` gets a sentence rather than an EOF
- Not enforced centrally as "`--platform-version` is required with `--yes`": it is required for the platforms with no default and not for the rest — custom, medusa, saleor, spree, sylius and bigcommerce set up under `--yes` without it, and shopify takes `--preset` instead. A central rule would mean keeping a list of platforms, and the list is what would rot
- **`service:enable storefront` picked a platform at random.** Two platforms claim that short name — medusa and spree — and the resolver ranged over a map and took the first match. Go randomises map iteration, so the answer was not merely ambiguous: measured over 200 calls, it returned `medusa/storefront` 195 times and `spree/storefront` 5. In a Spree project the command therefore set medusa's key almost always and its own occasionally, reported success either way, and changed nothing in the stack
- Candidates are collected, sorted, and the project's platform decides between them. A name claimed by several platforms and belonging to none of this project's is refused with both candidates named — guessing is what produced the bug, and the guess was silent
- **`shopware/messenger` was missing from the registry entirely**, so `service:enable messenger` in a Shopware project reached for sylius's key. It is registered now, which is a second contested name and the reason the resolver had to be fixed first
- A test pins the property rather than the cases: any short name claimed twice within one platform, or by an entry with no platform prefix, is unresolvable by anything and fails the suite — because that would put the random answer straight back
- **A failing command now exits with the code of the program it ran.** `madock cli bash -c "exit 137"` exited 1, and so did `exit 3`, and so did a failing test suite — a script could tell that something went wrong and nothing about what. 137 is the OOM killer and means "give it more memory"; 1 from a test runner means "fix the code". madock had the number the whole time: `exec.Cmd.Run` returns it, the pass-through commands hand it straight to the logger, and the debug log even printed it — it just never reached the caller, because `log.Fatal` exits 1 unconditionally
- Only where madock is a transport for somebody else's program — `cli`, `bash`, `composer`, `node`, `magento`, `n98`, `mftf`, `magento-cloud`, `db:execute`, and the platform wrappers. Where madock is doing its own work, a child failing is madock failing and the code stays 1; two meanings for one number is what this is fixing, not something to reintroduce
- **`madock shopware` said "Execute Shopware CLI" and runs `php bin/console`.** Shopware CLI is a different program — a separate Go binary that validates an extension against the store's requirements and builds its zip — and madock does not wrap it at all, so the wording sent somebody looking for a tool this command is not. It says `bin/console` now
- `shopware:bin` was described the same way and is the wider of the two: it runs anything in the project's `bin/`, not only console. The narrower name held the wider meaning and the wider name held the narrower one
- **Two commands in the Shopware documentation did not exist.** `madock swbin` and `madock sw:consume` appear in [docs/shopware.md](docs/shopware.md) — in the examples and in the command table — and neither is an alias: they are `sw:b` and `sw:c`. Every one of those lines answered "command not found" for anybody who copied it
- **`status` named a container the project no longer has as one of its services.** `docker compose ps` lists by project rather than by file — the project name comes from the directory the compose file sits in — so a container created from an earlier version of that file keeps being reported after its service is gone from the configuration. Turning `db/enabled` off, or moving a project onto a shared database, left the old container running and `status` calling it a service
- **It hid a real defect for a day, which is how it was found.** A project whose config said `db/type: MariaDB` generated a compose file with no `db` service at all, and `status --json` went on listing `db` as running — so the first test written for that bug passed against the broken build, and only reading the generated file showed the truth. A status that invents a service is worse than one that says nothing, because it is believed
- Named, not removed, and that is the decision rather than an omission. The compose file is generated from a config, so a rendering bug decides what counts as an orphan: `--remove-orphans` on `up` would have pointed the deletion at a running database container in exactly the case above. `orphan` is a field in `--json`, absent for ordinary services, and a phrase on the human line
- The list of services is asked of compose (`config --services`) rather than parsed out of the YAML, so the answer is the one `up` acts on — override files and interpolation included. When compose cannot be asked, or answers with nothing, no container is flagged: an unanswered check must not become "everything here is a leftover"

**v4.0.0**

Changed:
- **The module path is now `github.com/faradey/madock/v4`.** Go puts the major version in the import path from v2 on, so a 4.x release cannot keep the old one. Anything importing madock as a library updates its imports; nothing about the binary, the commands or a project's configuration changes with it
- **`db:info` and `madock info` print passwords again by default in this edition.** The change that described them instead was right for a server and wrong here: madock manages a developer's own laptop, `db:info` is run to copy the value into a database client, and the config file it comes from is two directories away — withholding it added a flag to every such use and protected nothing. The paid edition, which runs on servers where the same command prints a shared database's **root** password into a ticket or a screen share, still withholds it and still takes `--show-secrets`
- The switch lives in the library and the edition sets it, the same way the help renderer and the command scope resolver already work. Not a configuration option: a setting can be wrong on a server and nothing would say so, while the edition cannot be

Fixed:
- **`madock install --help` ran a real installation.** On an installed project it printed the assembled `bin/magento setup:install --base-url=… --admin-password=…` line — the password with it — and ran it, over the live database, reaching "Enabling Maintenance Mode" before it stopped on the existing `env.php`. Somebody typing `--help` has asked for the exact opposite of that. **Found on a live stand, 2026-08-20**
- **It was never `install`'s bug alone, and that is the part worth reading.** Answering `--help` used to be each command's own job, done as a side effect of calling the argument parser; a command that had no arguments to parse never called one, and so ran instead of explaining itself. Counted across the registry that day: eight in madock — `install`, `stop`, `ssl:rebuild`, `mcp`, `mftf:init`, `compress`, `uncompress`, `config:cache:clean` — and over fifty in madock-pro, `backup:create`, `firewall:setup`, `server:init` and `shared-db:unshare` among them
- The check moved to the dispatcher, which every command passes through, and the opt-out is a flag on the definition: `PassThrough` marks a command that hands its arguments to another program, where `--help` is composer's or `bin/magento`'s to answer. The default is the safe one on purpose — forgetting the flag on a pass-through command prints madock's help instead of composer's, which is visible and costs nothing, while forgetting to parse was silent and ran the command. A pinned test makes marking one a deliberate edit in two places
- Help is answered **before** the project check, because asking what a command does is not a use of a project. `--help` only, never `-h`: the short form is `--host` for `setup:env`, and a dispatcher cannot tell them apart without knowing each command's own flags — go-arg still answers `-h` from inside a command that has an argument struct
- `install`'s own help line said "Install Magento", which is neither what it does — the platform comes from the project's configuration — nor a warning that it is destructive. It says both now
- **The end-to-end suite covered `--help` for exactly one command, and it was the one that worked.** `restart --help` has a test because `restart` parses its arguments before stopping anything; none of the eight that ran instead of answering had one. They do now, and against a pre-fix binary the run reads like the bug report: `ssl:rebuild --help` generated a key pair and asked for the password to add a certificate to the system trust store, `install --help` answered "This command is not supported for custom" rather than printing help, and `stop --help` took ten seconds, stopped the project, and *then* printed the help — so for that one only the container state proves anything, which is a second test
- **A project with `db/type` = `MariaDB` came up with no database container, and said nothing.** Every compose snippet gates on one of three engine families — `db.yml` on `mysql`, `db-postgresql.yml` on `postgresql`, `db-mongodb.yml` on `mongodb` — and "mariadb" is a repository wearing a family's name, so it matched none of them. The generated `docker-compose.yml` simply had no `db` service, while the `dbdata` volume was still declared, so nothing about the output looked truncated
- What it looked like from outside is the reason this cost a day: `madock start` reported success, `madock status` listed php, nginx and opensearch without mentioning a database, and `madock info` printed "Database: type MARIADB, host db". The failure surfaced somewhere else entirely — on Magento as `bin/magento` answering "There are no commands defined in the … namespace", which points nowhere near the cause
- The normalization already existed and was reached only when `db/type` was **empty** (`DbTypeFromRepository`, where mariadb → mysql is pinned by a test). An explicitly written value went straight to `strings.ToLower` and past the one function written for it. It is normalized at read time now, which covers the historical configs and the hand-edited ones alike without a madock command writing into a file that belongs to the project's repository
- An unreadable value falls back to the repository rather than defaulting blindly, so `db/type=postgress` beside `db/repository=postgres:16` still gets postgres. What cannot happen again is an answer no snippet gates on, because that one is silent
- **Found on a project whose config madock itself wrote in March 2024**, so the radius is every configuration old enough to predate the current writer
- **`madock status` printed a JSON parse error on every run of some projects.** `Could not read the container status: invalid character 'i' in literal true (expecting 'r')` — a JSON parser complaining about English. The status still printed underneath it
- The cause is `CombinedOutput`: compose writes its data to stdout and its warnings to stderr, and folding them together handed "the attribute `version` is obsolete, it will be ignored" to the decoder as if it were data. Combining was a deliberate choice — the failure message needs docker's own words, and the exit code alone had been useless — so stderr is captured separately instead, which answers both. The same pattern is fixed in the two proxy checks that read `compose ps --format json`
- The line reader also ignores anything that is not an object now. Docker has put a warning on stdout before, and ignoring a line of prose costs a line of prose while parsing one costs the whole status

**v3.9.22**

Added:
- **`rebuild` now removes settings the project has deleted.** madock keeps two copies of a project's configuration — the one in the repository and the one written into the installation at setup — and reads prefer the first, so an added or changed setting arrives on its own while a **deleted** one does not: the read falls through to the installed copy, which still holds it. **Measured on Pricesmith, 2026-08-17**: a `custom_commands` block was removed from the repository, committed and rolled out, and `madock pr` went on working on every machine that had ever run setup. Nothing failed, and nobody was told
- **What made this unfixable was telling two identical keys apart** — one copied from the project at setup, one typed here with `config:set` — and there was no such distinction anywhere. There is now: madock keeps a snapshot of the project's configuration as it last saw it, and uses it the way a merge base is used. A key that was in the snapshot and is gone from the project was the project's to remove; a key whose installed value no longer matches the snapshot was changed on this machine, and is reported and kept
- Only what the snapshot recorded is ever a candidate, so `path`, generated passwords, allocated ports — everything madock writes into the installed copy itself — cannot be touched by a deletion in the project. The first rebuild after upgrading records the baseline and removes nothing, and says so
- Nothing is silent: every key removed is printed with the reason, and so is every key kept. Swapping a setting that refuses to leave for one that vanishes without a word would be a poor trade

Decided:
- **`status` keeps exiting zero when nothing is running**, and that is now written down and pinned by a test. Zero means the question was answered — "nothing is running" is a true answer to "what is running", and a script that reads it as failure cannot tell a stopped project from a broken one. A non-zero exit is reserved for a question that could not be asked, which is what `status` already does when docker does not answer. Exit codes are documented in [docs/json_output.md](docs/json_output.md)

**v3.9.21**

Fixed:
- **The `db/type` migration wrote an engine into projects that had never named one.** `GetDbType` answers "mysql" for anything it cannot read as postgres or mongo — nothing at all included — so V340 put `db/type=mysql` into the config of a project with `db/enabled=false` and no repository. A migration may carry a key across; it may not invent one, and the invention lands in the project's own committed file, where every later reader believes it
- Two guards, answering different questions: `db/enabled=false` is the project saying it has no database (absent still means enabled, as everywhere else in the codebase), and no `db/repository` is the project having said nothing to carry across. Neither costs anything — `GetDbType` falls back the same way at read time whether or not the key was ever written

**v3.9.20**

Added:
- **`service:restart <name>` restarts one service and leaves the rest of the project running.** `madock service:restart php`, `madock service:restart php worker-queue`. Names are the ones compose uses — a name the project does not have is refused with the list of the ones it does — and a config key resolves too, so `search/opensearch` finds `opensearch`
- **This is the command a deploy recipe can call, and `restart` is not.** Not for being blunt: `restart` stops every container including `deployer`, which on a deployer host is the container running the deploy, so a recipe calling it dies at the moment it succeeds. Restarting after a deploy therefore stayed a second step for a person to remember — and **on 2026-08-19, on one machine, three of four projects were serving a release older than the one `current` pointed at**, twice in the same session, with two of the three who forgot knowing about the trap. One of them was found only because a new column stayed NULL where the new code always fills it
- The same precision removes a cost on a shared-database machine, where a project-wide restart takes down the database container every other application on the host is connected to
- **It reports the state after the restart, not the fact that one was ordered.** `docker compose restart` exits zero once the signals are sent, and a worker with a broken command is gone a moment later; a service that is not running when the command returns is a failure naming the service. A docker that cannot be asked afterwards says so — that is neither success nor failure, and rounding it to either is the silence this command was written against
- Every name is resolved before anything is restarted, so a typo in the second of two arguments does not leave the first restarted and the caller guessing

Changed:
- **`madock restart php` now names the command that does what was asked.** It was already refused rather than acted on — that is the 3.9.16 fix and it still holds, containers stay up — but the message was go-arg's "too many positional arguments at 'php'", which reads as a parser complaint and names neither the intent nor the command that serves it

**v3.9.19**

Fixed:
- **Where the application lives is derived now, not stored.** With deployer managing releases the root is `<mount>/current`, and that was a value written into the config once, by `deploy:enable` — a copy of a derivation, kept in one file. It is computed on every read instead, in the same place `nodejs/major_version` and `db/type` are computed, so no caller has to remember and none can read a stale one
- **Three failures in one day, 2026-08-19, all the same shape**: a cron job with the path spelled out, correct until the project moved to deployer; a systemd unit on the host naming `current/artisan`, pointing at a deploy root that has no artisan in it; and a deploy that failed after the pre-rebuild, which removes the file the copy lived in — the rebuild in that window read an older config and installed a scheduler that ran the wrong command for seventeen minutes on a live application. Derived, the answer survives that file being absent, which is exactly the window that broke
- Idempotent, so an installation that already stored `/var/www/html/current` reads back unchanged, and a custom root keeps its shape: `/var/www/html/storefront` becomes `/var/www/html/storefront/current`
- **One key, and `php/workdir` is not a second one.** `deploy:enable` used to write it alongside `workdir`, and nothing has ever read it: no config declares it, no template renders it, no command consults it. It is not derived here either — a second spelling of one fact is how the two answers get to disagree, which is the defect this release is about. The write is gone from madock-pro
- **`snapshot:restore` and `project:clone` refuse on a deployer-managed project.** Both write a file tree into `/var/www/html` after `rm -rf`. On an ordinary project that path is the application and the mount at once, which is why the literal was never wrong; here the mount holds `releases/`, `shared/`, `current` and deployer's state, so the same line would delete every release **and `shared/`** — where the environment file lives, and which no snapshot contains. The database half of a restore would have worked, which is worse: it would have done the safe part first. They stop and name what to do instead — a deploy, `deploy:rollback`, or `db:import` for the database alone

**v3.9.18**

Added:
- **A long-running process can be a service of the environment, on any platform.** `<worker><programs><queue>…</queue></programs></worker>` in the project config, one compose service per named entry, called `worker-<name>`. It gets the main service's image — so a PHP project's worker runs on the PHP image, a Node project's on the Node one, with no second Dockerfile to drift — the project's `workdir` as its working directory, and `entrypoint: []`, because the language images start the application and a worker replaces that rather than running after it
- The form already existed three times and was locked away each time: `shopware-messenger.yml`, `sylius-messenger.yml` and `saleor-worker.yml` are each included only from their own platform's compose and gated on a key only that platform has. A Laravel queue or a Node job runner had nowhere to go, so people put a systemd unit on the host running `madock cli <command>`. Those three snippets are unchanged and keep their keys; this is the one anybody can use
- **Measured on a production machine on 2026-08-19, which is what the feature is arguing against.** The unit knew the path `/var/www/html/current/artisan` and pointed at a directory without it after the move to a deployer layout. Every rebuild killed the `docker exec` underneath it and systemd restarted the chain — the counter had reached 74. And `systemctl is-active` answered `active` whether or not the container was there, so the liveness check was reading the wrapper rather than the work. A service dies and returns with the container, and `madock status` counts it
- It also removes a cost nobody would predict: a host-side unit keeps the madock binary open, and replacing an open binary with `cp` fails with `Text file busy` — discovered while updating a server, which is a poor moment to learn it
- Documented in [docs/workers.md](docs/workers.md), with a golden fixture rendering two programs on a platform that has no worker snippet of its own

**v3.9.17**

Fixed:
- **A `<jobs>` block with more than one `<job>` in it parsed to no jobs at all.** The XML reader hands a repeated text element over as a list, and the flattener knew three shapes — a string, a branch, a list of branches — so a list of strings matched nothing and the key was dropped without a word. One `<job>` parsed and worked; two or more silently became zero. Nothing downstream could tell that apart from a config with no jobs in it, so cron was started, the crontab was left empty, and `status` reported a scheduler that was running. **Measured on Pricesmith on 2026-08-19**, live and demo: seven jobs in `.madock/config.xml`, `Cron is running`, and `no crontab for www-data` in the container — no reconcile, no bulk-sync polling, no usage submission, for as long as the project had been on that config. The same silence applied to any repeated text tag anywhere in a config, and `<job>` is the only one the shipped defaults document
- The writer had to learn the shape too, and this is the half that would have been worse than the bug: an index cannot be written as an element, because `<job><0>…</0></job>` is not legal XML and the next read of that file fails with `invalid XML name: 0` — on a file madock wrote itself, from a parser that exits the process on a bad read. A list is written the way it was read, as the tag repeated
- **Deleting every job from the config left the old ones running in the container.** The install step returned early on an empty list without clearing what the previous one had put there, so a job removed from the config went on firing with nothing naming it. The config owns its jobs; an empty list is an instruction
- **Config jobs are installed into a `#~ MADOCK START … #~ MADOCK END` block, the way Magento marks its own.** The crontab of `www-data` is shared — Magento writes a block through `cron:install`, Shopware appends a `scheduled-task:run` line, and somebody may have added something by hand in the container — and installing config jobs used to wipe the whole file and write them over it. That survived only because the platform branch runs a moment later and puts its own block back, which `cron:install` does **not** do while Magento's DI is still warming up: the code already waits six times for the `cron` namespace to appear and gives up with a warning. In that window the wipe was permanent. Every read-modify-write now touches our block and nothing else
- Jobs already installed by an earlier version are unmarked, and are adopted into the block rather than duplicated by it
- **A job containing a quote broke the install.** The crontab was written with `echo '<jobs>' | crontab -`, so a single quote anywhere in a command — `php -r 'echo …'` is an ordinary thing to schedule — ended the quoting and handed the rest to the shell. It is written through a quoted heredoc now
- Repeated jobs are ordered by index as a number rather than as text, so the tenth job is no longer installed second

Added:
- **`{{workdir}}` in a cron job, so one committed config is right on every machine.** The application root is `/var/www/html` on a plain checkout and `/var/www/html/current` where deployer manages releases, so a job that writes the path out is correct on one kind of machine and wrong on the other — silently, because cron sends its output nowhere. madock's own jobs never had this problem: the magento2 and shopware branches build their command from `workdir`. This gives the config's jobs the same thing. Expansion happens when the crontab is written, so `crontab -l` shows the real path rather than an indirection to read at three in the morning
- `{{workdir}}` is the only placeholder, and secret keys are refused outright — a crontab is a file, and `{{db/password}}` would put a password in it. A job naming anything else is **not installed**, with a warning saying which placeholder and why: a line still carrying `{{…}}` would run every minute and fail every minute, into /dev/null

Changed:
- **`Cron is running` now says how many jobs are installed** — `Cron is running (7 jobs)`, or a warning when the daemon is up with an empty crontab, which is the state the bug above produced and the one the old line read as healthy. The count is read from the container's crontab, not from the config: the config is what was asked for. When the container cannot answer, the line says `installed jobs: unknown` rather than reporting none
- `status --json` gains `cron_jobs` and `cron_jobs_known`. `cron_jobs` is `-1` when the question could not be asked
- Starting a project whose `cron/enabled` is true with no jobs defined now says so once, in the start output, on platforms that install no jobs of their own. It was only ever said under `cron:enable`, run by hand — the path nobody takes on a server

Worth saying out loud, because it is the shape rather than the incident: **this is the second time a status line answered a question nobody asks.** `status` printing `exit status 1` was the first. Here it named the daemon while the only reason anyone looks is the jobs — and unlike the first, this one read as reassurance and did so for four days on a live product. A status line has to answer the question its reader has, and where it cannot, say that it cannot: `installed jobs: unknown` is an answer, `Cron is running` was not.

**v3.9.16**

Fixed:
- **`rebuild` destroyed the containers before it found out that the build context cannot be generated.** It stops first and renders second, so a template whose includes no longer resolve ended the run with the environment down and a message naming a path — `The file snippets/dockerfile/php/nodejs does not exist` — rather than a cause. **Measured on 2026-08-18**: a production machine went down that way inside a maintenance window, and the demo machine after it, on an ordinary `restart`. The drift behind it is structural: a project's own templates under `.madock/docker/` survive every madock upgrade, while the snippets they include ship inside the binary and move between releases — `php/nodejs` became `common/nodejs` — and nothing reconciles the two until a build
- `rebuild` and `restart` now resolve the includes of the project's own templates before touching anything, and refuse with the file, the missing include, the three places looked in, and what to do about it. Only missing includes stop the run: a template that fails to parse is left to the render, because a preflight that can fail for unrelated reasons is one people learn to skip
- **A consumer of a shared database could not export its own data.** `db:export` already dumped in a single transaction there, but mysqldump still issues a `FLUSH TABLES`, which needs RELOAD or FLUSH_TABLES — privileges a consumer's account does not have and should not. Measured across a live cluster: every consumer failed with error 1227, so the only way to take a dump was to run it against the provider as root, which reaches every other project's tables. `--skip-lock-tables` is the other half of the fix
- **`config:set nodejs/version` left the old `major_version` sitting in the file.** The value is computed on every read, so a stored copy can only go stale — and a stale copy is read by people even when nothing reads it. Measured on a live server: after `nodejs/version 22.22.0` the file still said `major_version 20`, and the next person to open it concluded the environment would build Node 20. It would not, since the render derives 22 from the version — but nothing in the file said so, and the conclusion cost an evening. Writing a source key now removes the stored derivative it governs, and says so; if the project's own committed `.madock/config.xml` also carries it, the command names that file instead of quietly leaving it
- **`restart` stopped the containers before it looked at its arguments.** It was stop-then-start with no parsing of its own, and the parsing lived inside `start` — which is to say it happened after everything was already down. Anything the command does not understand therefore reached the argument parser with the environment already stopped, and the parser ends the process. **Two cases measured on production machines on 2026-08-18**: `madock restart php`, meant as "restart just the php service", stopped nginx, php, db, redisdb and deployer and left them stopped — the message, `too many positional arguments at 'php'`, reads as though nothing had happened. And `madock restart --help` did the same to a shared-database provider, taking the cluster's database down with it: `--help` is typed exactly when somebody is unsure of the syntax, so asking how to use a command was a way of finding out
- Arguments are read first now, so the same typo costs a refusal and nothing else. `--with-chown` is passed through to `start` rather than re-parsed, because the parser answers only the first call in a process and a second one would have silently dropped the flag; `restart` also declares its arguments, so `madock restart --help` describes them
- **`db:info` printed the database passwords, root included, and it is the wrong command to print them from.** The values are in a config file on the same machine either way — the difference is radius. A file is read on purpose; the output of a command goes wherever output goes: terminal scrollback, a CI log, a screenshot, the issue somebody pastes it into. This one is run to find a host and a port. Worse on a project that borrows a shared database, where the root password it printed is the **provider's** — a key to every other project's schema on that server, which the consumer's own account cannot reach
- Both commands describe the value instead: `password: set (24)`, `not set` when there is none. `--show-secrets` prints them in full, which is the one case that was ever the point — copying a password into a database client. `madock info` used to mask by showing the first and last character, which gives away two characters of a password in exchange for nothing; it uses the same description now
- `db:info --json` masks too, and gains `password_set` / `root_password_set` so a script can still tell "no password" from "not printed" — the only thing it could honestly do with a masked string. JSON is not the safer half: it is what a CI log is made of
- Installing a platform still prints the admin password once, and that stays: it is the moment the account is created and the only time the value is told to anyone

Upgrading:
- **`db:info` and `madock info` no longer print passwords by default.** A script that read a password out of either — including out of `db:info --json`, where the `password` and `root_password` fields are now absent unless asked for — needs `--show-secrets` added to the command.

**v3.9.15**

Fixed:
- **Extraction removes what a release withdrew.** It only ever added and overwrote, so a template dropped from the shipped set stayed on the machine for good. Measured on a live server: twelve `docker-compose.{darwin,linux,windows}.yml` files that neither madock nor madock-pro ships, and `snippets/dockerfile/php/nodejs` still carrying the `.php.nodejs.enabled` syntax removed in 3.9.8. Harmless there by accident — the first are empty, and nothing includes the second any more
- Worth fixing before that accident runs out: the resolver reaches `{execDir}/docker/{platform}/docker-compose.<GOOS>.yml` and applies what it finds as `docker-compose.override.yml`, so a non-empty template withdrawn in a release would go on applying on upgraded machines and not on fresh ones — **at the same version**. The result would depend on the history of an installation rather than on what it says it is
- **Only files this mechanism itself wrote are removed**, from a manifest it keeps, never from a walk of the directory. madock-pro extracts its own platform templates into the same tree, and a sweep of everything-not-in-the-embed would delete them. An installation with no manifest yet — every installation before this version — has nothing removed, so orphans predating it need one clean by hand
- **The error a template raises when the binary is older than it now says so.** `function "memShareGB" not defined` reads as a broken template and sends somebody to edit a file that is correct: template functions are compiled into the binary while templates are files on disk. The message keeps the engine's own words and adds the cause and the cure. Measured on this machine, where the radius is every project at once, because the installation directory and the source checkout are the same directory

**v3.9.14**

Added:
- **`config:unset`, because nothing could remove a setting.** madock keeps the project's own `.madock/config.xml` — the copy in git — and a machine-side copy under the installation, seeded from it the first time the project is seen. Reads merge both with the project's copy winning, so *adding* or *changing* a setting in git reaches every machine. **Deleting one does not**: the machine-side copy still has it, `config:set` can only assign, and clearing the cache does not touch the file. The only way to drop a single key was to remove the project and set it up again
- Measured on a live project: a `custom_commands` block was deleted from the repository, committed and rolled out, and `madock pr` went on working on every machine that had ever run setup. Nothing broke and nobody was told, which is the whole difficulty — a setting retired months ago is still in force and nothing says so
- It reports by reading back rather than by trusting the write. A key can survive an unset in a way that looks exactly like success, because the project's own config may also set it and that file wins — so the command says which key is still set and where to look, instead of printing "removed" over a value that did not move

Still open, and stated rather than quietly skipped: which copy should win in general. Making `rebuild` reduce the machine-side copy to the project's would fix the class rather than the case, and would also discard everything set with `config:set` on that machine. Telling those two apart needs provenance the config does not record, and that is a decision, not a patch.

**v3.9.13**

Fixed:
- **A project's own `.madock/config.xml` is edited now, not re-rendered.** It is the one file madock writes that a person wrote first and committed to their repository, and its comments are usually the record of *why* a setting is what it is — "the database is off because this app talks to the shared cluster" is not something the values can say. Parsing it into a map and rendering the map back lost every comment, sorted the keys alphabetically and turned a one-line migration into a diff nobody reads. The new writer edits the document text by byte offsets from the decoder: the element that changes has the text between its tags replaced, and every other byte is copied through. Measured on the reproduction that opened this: a run that used to change 63 lines now changes **one**, and that one is a setting genuinely being added
- Scoped to that file on purpose. The registry copies and the installation's own config are machine-owned, have no comments, and go on using the renderer — putting every config write on a new path for the benefit of one file is how a formatting fix becomes an outage

Added:
- **Settings can be removed, which was never possible.** `config:set` can only assign, there is no unset, and clearing the cache does not touch the file — so a key taken out of a project's config stayed in the installed copy for good, and the only way to drop one was to remove the project and set it up again. A team that deletes a setting, commits and rolls out therefore left every machine that had ever run setup living by the old value, with nothing failing and nobody told. `RemoveKeepingComments` is the primitive that ends that; removing a branch takes its children, which is what an explicit unset of a branch means

**v3.9.12**

Fixed:
- **A migration rewrote a project as a language it is not.** `V320` backfills `language` into configs that lack one, and its guard asked `rawConf["language"]` — but the parser returns keys as they sit in the file, so the real key is `scopes/default/language`. The lookup therefore never found one, the guard always passed, and a **nodejs** project was written back as **php**. Measured: it sent `madock cli` into a php container that does not exist, and cost half an hour looking for a defect in the service resolver instead
- It hid because it only fires when the migrations run at all — an installation with a current recorded version never reaches it, and a fresh one reaches it every time. Which means it fired precisely when somebody tried a new build against a clean `MADOCK_EXEC_DIR`, the one safe way to try a new build
- **Not a regression.** Measured against 3.9.5, which produces a byte-identical rewrite: this has been there since the migration was written
- Config files are written with a trailing newline and without the formatter's leading blank line. These files live in somebody's repository, so every write used to show up as two spurious line changes on top of the real one

Known, and not fixed here: a migration that writes a project's `.madock/config.xml` still loses the XML comments in it and reorders the keys, because the file is parsed and rendered rather than edited. The values all survive. Comments in that file are often the record of *why* a setting is what it is, so this is worth closing — it needs the writer to edit the text rather than re-render it.

**v3.9.11**

Fixed:
- **Migrations were gated by comparing versions as strings, and the first two-digit patch release breaks that.** `"3.9.10" < "3.9.8"` is **true** as a string, because `'1'` sorts before `'8'` — so an installation on 3.9.10 would have re-run the 3.9.8 migration on every command, forever. On 3.10.0 it is three migrations, not one. It would also have stayed quiet: migrations here are written to be harmless when there is nothing to do, which is exactly what would have kept anybody from noticing. `configs.CompareVersions` was already in the tree; the gate uses it now, and a test pins both the case that broke and the ordinary ones the string compare happened to get right

**v3.9.10**

Changed:
- **The new key is `nodejs/embedded/enabled`, not a bare `nodejs/embedded`.** The short form was the switch itself, and every other service in madock is an `<x>/enabled` pair — so it needed a special case in the resolver, in `service:enable`, in `service:disable` and in `service:list`. That last one is what settles the question: the listing walked the config splitting on `/enabled`, so the setting was enableable and **invisible**, which is worse than not being settable at all. It was caught by somebody asking, not by a test, and the next config walker would have missed it the same way. Shaping it like everything else deleted the special case entirely — about forty lines, and the class of bug with them
- Nothing was released with the short form: 3.9.8 and 3.9.9 are `-norelease` tags, no server carries them and no customer has the key. `service:enable nodejs/embedded` still reads exactly the same, because `nodejs/embedded` is now an ordinary service prefix rather than a special one

**v3.9.9**

Fixed:
- **The old service names still work, and say what replaced them.** `service:enable php/nodejs` is documented in four places and worked for every project, because `php/nodejs/enabled` was in the shipped defaults — so 3.9.8 would have met people with `The service "php/nodejs" doesn't exist.`, which is true and useless: it says neither that the thing moved nor where to. It is accepted now, resolves to `nodejs/embedded`, and prints the new name once. That line is the point — somebody learns the rename from the command they already type, rather than from a changelog they never opened
- `php/yarn` is aliased on the same terms, with one honest difference: it was **never** a registered service and never documented as one. It was in no service map, in no defaults, and worked only where a platform configurator happened to have written `php/yarn/enabled` — shopify, bigcommerce and sylius, three platforms out of eleven. The alias is there for whoever scripted it on one of those, not because it was a supported name
- Neither alias is permanent. They are cheap while the old name is still in people's fingers and in their scripts, and they come out when it is not
- **`service:list` could not see a setting that is itself the switch.** It walked the config splitting on `/enabled`, so `nodejs/embedded` was enableable and invisible — which is worse than not being enableable at all. It is listed now, and the scope filter moved ahead of the check so an override does not list the same service twice

**v3.9.8**

Changed:
- **Node stopped being PHP's business: `php/nodejs/enabled` is now `nodejs/embedded`.** The old name said a runtime belongs to the language beside it, and it does not — a Python service with a JavaScript front end, or a Go one with an admin panel, needs exactly the same thing, and until now there was no way to ask for it. Half the design was already general: the version has always lived at `nodejs/version` and the php snippet read it, so only the switch was wrong. The snippet moved from `snippets/dockerfile/php/nodejs` to `snippets/dockerfile/common/nodejs` and is now included by the python, ruby and golang images as well — every one of those base images is Debian or Ubuntu, so the same nodesource install works in each
- **`php/yarn/enabled` is now `nodejs/yarn/enabled`**, and that key was never declared at all: no default carried it, three platform configurators set it, and one template read it, while a real `nodejs/yarn/enabled` sat unused in the defaults
- `service:enable php/nodejs` was documented and worked by accident — it was in no service map, and the lookup found it only because `php/nodejs/enabled` happened to exist. `service:enable nodejs/embedded` (or `embedded-node`) is the deliberate replacement: a service map entry for a setting that is itself the switch rather than an `<x>/enabled` pair
- **The rename carries itself.** The migration moves the key in the installation's config, in the global project defaults, in every project's registry config, in each project's own `.madock/config.xml` — the one file no command may write, where a migration is the single exception — and in any template a project copied under `.madock/docker/`. Missing one of those is silent in the worst way: the key no longer exists, the renderer answers it as false, node stops being installed, and it surfaces much later as a build with no npm. Every scope in a file is migrated, not only `default`, and a file with nothing to migrate is not rewritten at all
- Two golden cases were added, because none existed with node turned on: the whole half of the php image that installs node was rendered by nothing and compared against nothing. One covers node in the php image, the other node in a python image — the case the rename exists for

**v3.9.7**

Fixed:
- **The guard added in 3.9.6 covered `prune`, and `prune` is not destructive.** It is `docker compose down` for the current project: containers and the network go, the volumes stay, the images stay, the project directory and its registry entry are untouched, and `madock start` puts it all back. Guarding it obstructed without protecting anything — and inconsistently, since `stop` takes the same site down with nothing in front of it. Only `prune --with-volumes` is covered now, which is `down -v --rmi all`: the data volumes and the images, with nothing to bring the database back. `project:remove` is unchanged

**v3.9.6**

Added:
- **`allow_destructive_commands`, the one setting a project cannot reach.** `project:remove` ends in a recursive delete of the project directory and `prune` — which sounds like tidying — is `docker compose down` for the current project, with `--with-volumes` its data volumes and images as well. Neither had anything in front of it but a `--force` flag. The setting sits at the **top level** of `~/.madock/config.xml`, outside `<scopes>`, and that placement is the feature: everything under `<scopes>` is overridable by a project's own `.madock/config.xml` and writable by `config:set`, so a guard put there would be switchable by the very thing it protects against. A copy of the key in a project's config, or inside `<scopes>`, is ignored — an e2e test makes the attempt through the real binary rather than trusting the resolver
- The default stays `true`, because a laptop creates and removes projects all day and a guard that gets in the way is one people learn to switch off. madock-pro ships it `false`: that edition is the one that runs on servers. The installation's own file has the last word in both directions, so a machine that needs the command back does not need a different binary
- It is a guard rail and not a security control, and the refusal says so: the file is writable by the same user who runs the command. It stops a mistake, not a person

Fixed:
- **A comment in `config.xml` could make the key test report nonsense.** `TestEveryKeyIsAKeyMadockHas` reads the config with a regexp over tag names, and it had no idea what a comment was — a tag mentioned inside one was pushed onto its stack and, having no closing half, never left it. Every key after that point was then computed one level too deep, so the test said madock has no setting called `restart_policy` and listed forty more, none of them the comment that caused it. Latent until now only because the two commented-out blocks in the file (`<hosts>`, `<jobs>`) happen to be balanced

**v3.9.5**

Upgrading:
- **The shared proxy's request limits become limits.** `proxy/rate_limit` defaulted to 1000 requests a second per address with a burst of 2000, which is not a limit but a permission — it was written to catch a request loop, and the comment in the code said so. The default is now 50 with a burst of 200: an asset-heavy page still loads in one go, while one address can no longer occupy the machine. It remains per address and therefore does nothing against a distributed flood; nothing on this side of the wire does
- **`client_max_body_size` was 2G, hard-coded in every server block.** That is an upload nobody makes through a browser and a cheap way to hold workers and disk. It is `proxy/max_body_size` now, defaulting to 128M, and a project that genuinely needs more sets its own

Added:
- **`proxy/conn_limit`, the half of resource exhaustion nothing answered.** A request that never finishes spends no rate at all, so a few hundred slow connections hold every worker the proxy has while staying under any per-second limit. 100 simultaneous connections per address by default, where a browser needs six to eight
- **`proxy/livereload/publish` and `proxy/vite/publish`, both on by default.** The shared proxy publishes LiveReload on 35729 and Vite on 5173 as fixed numbers, so they fall outside the 17000-19999 range the firewall guard closes — and they are published whether or not anything can serve them, since Vite needs nodejs a project may not run. On a laptop that is the point of them; on a server it is a development server answering the internet, which is where one was found. `publish` rather than `enabled`, because that is all it does: the project still runs the tool and still publishes its own allocated port, which the guard covers. What goes away is the proxy's fixed entry point, the one that routes by Host so the same number works for every project. The neighbouring `mailpit/enabled` means something else — there it decides whether a container exists at all

**v3.9.4**

Fixed:
- **An installation could stop receiving templates altogether, and look healthy doing it.** The guard added in 3.9.3 skips extraction when the installation directory holds a `go.mod`, on the reasoning that such an install is a clone and git delivers `docker/`. That is true of madock and false of an installation that imports it: madock-pro's install directory is a clone with `go.mod` at the root, but its `docker/` is in `.gitignore` on purpose, because the assets belong to the imported module and arrive by extraction. So extraction switched itself off in the one installation where nothing else brings the templates in, and the tree stopped moving — measured on the development install, `.embedded_version` read **3.6.7** against a **3.9.3** module, with 47 templates still written in the syntax the engine replaced in 3.9.1. Nothing announced it, because the renderer converts the old syntax on the fly and warns: every command worked, from templates two years old. A customer install, being a bare binary with no `go.mod`, was unaffected — so the breakage was confined to the installation the paid edition is developed and tested on, which is the worst place for it to hide
- The test is now a file in the tree it is a statement about: `docker/embed.go`, the embed declaration. It exists wherever the templates are source, and it appears in no `//go:embed` pattern — those name asset directories — so extraction cannot write it and cannot make an extracted tree pass for a checkout. A test pins the path, because a rename would turn the guard off silently and bring back the reverted-edits defect it was written for

**v3.9.3**

Added:
- **`db/memory`, one budget that sizes whichever database engine the project runs.** The numbers used to be written into `my.cnf` by hand — `innodb_buffer_pool_size = 512M` and `innodb_log_buffer_size = 256M`, 768 MB reserved by every MySQL before it holds a single row — and there was no setting for them at all. The only way to change one was to copy the shipped `my.cnf` into a project's `.madock/docker/db/`, which is a frozen copy that then drifts from the file it was taken from, silently and for as long as nobody compares them. Measured on a demo server: two `mysqld` holding 732 and 409 MB of 3819, with the container limits unable to reach them
- **The engines are not the same problem, which is why one number is divided rather than passed through.** MySQL takes two thirds as its buffer pool and one third as its log buffer, so the default of `768M` renders exactly the two lines that were there before — the golden `my.cnf` does not move. PostgreSQL had never been given anything and sat on its stock 128MB `shared_buffers`; it now takes a quarter of the budget there and three quarters as `effective_cache_size`, which buys no memory but tells the planner what to expect. MongoDB is the one that had to be told: left alone, WiredTiger sizes its cache from the RAM it can see, which is the **host's** and not the container's limit, so one `mongod` on a large machine quietly reserves gigabytes. It takes half the budget, with MongoDB's own 0.25 GB floor
- Templates can do the arithmetic themselves — `memShare` for a fraction of the budget in a given unit, `memShareGB` for MongoDB's bare decimal. Fractions rather than percentages so the numbers land where a person would have written them: two thirds of 768M is exactly 512M, where 67% is 514M and every generated file would differ from the one before it for no reason
- A golden fixture for MongoDB, which had none. Rendering `--wiredTigerCacheSizeGB` correctly is exactly the kind of thing that is wrong once and then wrong forever
- **`--db-type`, so a scripted setup can choose an engine.** `--db` names a version — or a repository and a version, when it carries a colon — and the engine was asked for interactively, which under `--yes` meant every scripted project got MariaDB. It also made an end-to-end test of PostgreSQL or MongoDB impossible: changing `db/type` afterwards leaves the first engine's data directory in the volume and the second refuses to start on it. Three e2e tests come with it, one per engine, each asking the running server what it actually got
- The engine flag works alongside an explicit version, which took two fixes of its own. The platform setups asked for the engine only when no version was given, on the reasoning that `--db` answers the whole question — it does not, a version says nothing about an engine. And a version carried over from the general configuration belongs to the engine that was there before: asking for MongoDB produced `FROM mongo:10.6`, the MariaDB default and a tag that does not exist, so a default from another engine's list is now dropped rather than reused

Fixed:
- **A PostgreSQL 18 project could not start at all.** The image keeps a major-version directory under `/var/lib/postgresql`, so that `pg_ctlcluster` and `pg_upgrade` can see two versions at once; madock mounted the volume at the older `/var/lib/postgresql/data`, and finding a database in what it now treats as an unused mount makes the image refuse outright. The mount follows the version now. It had been hiding behind another defect: under `--yes` the version silently stayed at the MariaDB default, and `postgres:10.6` exists, so a scripted postgres project quietly came up on a seven-year-old release rather than failing — and once the version selection was fixed, 18 is what a new project gets. A golden fixture holds the mount and an end-to-end test starts a real 18 container and asks it a query
- **A development binary reverted edited templates in the source tree.** The embedded `docker/` is extracted over the installation whenever the binary's version differs from the `.embedded_version` beside it — and for an installation made by `install.sh`, the installation *is* the clone, so the extraction writes a build-time snapshot over the working copy. A binary built before an edit therefore undoes it, silently, and `version` is enough to trigger it because the check runs before any command. Measured the expensive way: three edited database templates disappeared mid-session, and an end-to-end test then passed against the reverted files while reporting the number the engine had chosen for itself. Extraction now skips a source checkout entirely — that install gets its templates from git, and only a binary installation has no other way to get them
- **`--db postgres:16` set the repository and left the engine at MySQL.** The colon form writes `db/repository` and `db/version`, but `db/type` came from the engine question, which nobody answered under `--yes` — so the generated compose rendered the MySQL service block around a postgres image and set `MYSQL_*` on it. An explicit repository now decides the engine, being the more specific thing the user said

Fixed:
- **An argument containing shell syntax never reached the program it was meant for.** Every passthrough command — `cli`, `magento`, `composer`, `node`, `n98`, `shopware`, `claude` and the rest, twenty-one of them — joins the arguments back into one string and hands it to `bash -c` in the container. Argv arrives already split by the caller's shell with the quoting removed, so the join makes the container's shell split it a second time, by its own rules rather than the ones that were typed:

  ```
  $ madock cli node -e "console.log(process.version)"
  bash: -c: line 1: syntax error near unexpected token `('
  ```

  The parentheses never reached node. An argument was quoted only when it contained a space or an `=`, which is neither brackets, nor `|`, `;`, `&`, `*`, `>`, `$`, backticks, `#`, `~`, nor a newline — and the quoting used was double quotes, which leave `$` and backticks live, so `madock cli echo '$(id -u)'` ran the substitution instead of printing it. Values also had quote characters stripped off both ends. Every argument is now single-quoted, which is the one form that carries anything.
- Two behaviours are kept, and both are load-bearing. A single argument is still passed through untouched, because `madock cli "ls -la | grep conf"` is how a pipeline has always been run and there is no way to both quote that string and leave it a pipeline — one argument means a script, several mean argv. And `NAME=value` keeps its shape with only the value quoted: bash reads an assignment only when the `=` is unquoted, so quoting the whole word would turn `madock cli APP_ENV=dev php bin/console` into a search for a command called `APP_ENV=dev php`.
- Arguments are no longer trimmed of surrounding whitespace on the way through. A function whose job is to deliver an argument unchanged has no business editing it.
- Measured on a real container before and after, not only in tests: the reported command errors on 3.9.2 and prints the string here, `echo 'a|b' 'c;d' '$HOME' "it's"` arrives whole, `madock cli echo --json hello` still passes its flags through rather than madock eating them, and the pipeline idiom still pipes.

**v3.9.2**

Added:
- **`madock template:convert`**, for anyone keeping their own docker templates under `.madock/docker/`. An override written in the old `<<<if>>>` syntax already keeps working — it is converted as it is read — but the warning that says so could only tell people to rewrite it by hand or to run `go run ./tools/tmplconvert`, which is an answer for somebody who installed from source and no answer at all for somebody holding a downloaded binary. The command rewrites in place, reports every file, has `--dry-run`, and running it twice is a no-op. It is the same conversion the renderer applies at read time, so the two cannot drift
- **The warning now names the file on disk and the command that fixes it.** It named the template — `nginx/conf/default.conf` — which says which template and not which file: the shipped copy and the project's override are both that, and only one of them is the reader's to edit

Upgrading:
- **The default timezone for a *new* project created on an installation from source becomes UTC**, where it was Europe/Kiev. Existing projects are untouched: `setup` writes `timezone` into the project's own configuration, so every project already created carries its own value. The reason it changed is below — the two files of defaults disagreed, and this is the one key where they disagreed on a value rather than on existence

Fixed:
- **madock had two files of defaults and they had drifted apart.** `config_defaults.xml` is compiled into the binary and is the only source an installation from a downloaded release has; the `config.xml` at the repository root is what `install.sh` leaves on disk, and it is read **over** the embedded defaults. So the copy that is easiest to forget is the one with the last word — and it had been forgotten: a strict subset missing seventy-three keys, among them all of `php/sendmail`, `proxy/mailpit`, `memcached`, `artemis`, `search/meilisearch`, `permissions/umask` and every storefront block. The visible consequence was two installations of the same version disagreeing about the timezone. The root copy is now identical, and a test compares them byte for byte so they cannot drift again — bytes rather than keys, because that file is what the documentation points a user at to read the available settings, so its comments have to be current too
- **A project's configured SSH key was never offered when an agent was running.** `remote:sync:*` built the agent and `ssh/key_path` as two entries in the client's auth list, and to the protocol both answer to the name `publickey` — which is what the client marks as tried, by name. So an agent that had no key the server would accept did not fall back to the key file: it closed that path too. Measured on a live sshd, the server log held exactly one attempt, with a key from the agent, and none with the key the project had been told to use. The visible symptom was `ssh host` working while `remote:sync:media` failed on the same host, from the same machine. They are now one method, agent keys first and the key file after, so every key is tried
- **Two things that had to come with it, or the fix would have been worse than the defect.** The key file is still not read while an agent key is in play: the signer built for it finds its public half without a passphrase — from the `.pub` beside it, from the key itself when it is not encrypted, or from the cleartext public key an encrypted OpenSSH key carries inside — and reads the private half only once the server has said it would accept that key. A key whose public half cannot be found that way is not offered at all, because the handshake dereferences it before anything can check it. And an RSA key file now signs `rsa-sha2-256`: a plain signer is assumed to manage only `ssh-rsa`, which is SHA-1 and refused by every OpenSSH since 8.8, so one silent refusal would have been traded for another on the commonest key type
- The same defect was fixed in madock-pro 0.23.7, which replaces the whole client configuration and so was never on this path. This is the same fix for everyone else

**v3.9.1**

Upgrading:
- **The `<<<if>>>` template syntax is deprecated.** Every file under `docker/` is now a Go `text/template` with `{{{ }}}` delimiters, documented in [Customizations](docs/customizations.md). A project's own override under `.madock/docker/` written in the old syntax keeps working — it is converted as it is read — but madock names the file in a warning on every generation, and the conversion goes away in a later release. `go run ./tools/tmplconvert -dir <path>` converts a tree in place
- **Generated files move in whitespace.** Blank lines that came from the line an `{{{include}}}` tag sat on are gone, eighteen in a row at the end of a default compose file among them. Nothing about a container changes; a diff of `aruntime/projects/<name>/` after an upgrade will not be empty

Fixed:
- **golang.org/x/crypto updated to 0.52.0**, clearing seventeen advisories, worst critical. Not theoretical here: it is a direct dependency and `remote:sync` imports `ssh`, `ssh/agent` and `ssh/terminal` to pull a production database down over SSH, while two of the CVEs are exactly that code path — agent constraints not dropped when keys are forwarded (CVE-2026-39832) and key constraints not enforced (CVE-2026-39833). `x/sys` and `x/term` come along as the versions it now requires; no source change was needed
- **A project's block in the shared proxy was rendered with another project's configuration.** `proxy.conf` is built by walking every registered project, and the substitution pass was handed the name of whichever project happened to be starting — so the mftf locations in one project's server block followed a different project's setting. It follows its own now
- **A snippet that includes itself no longer spins forever.** The include pass was a regex looped "while a match remains", with no cycle detection of any kind

Added:
- **`nginx/enabled`, for a project that answers no request.** A queue worker, a bus consumer, the service that owns a shared database schema while its neighbours take their own webhooks — madock gave all of them a web server. `false` leaves out the container, the vhost, the block in the shared proxy, the name in the shared certificate and the two reserved ports. Removing the project's `<hosts>` was the obvious way to ask for this and did none of it: the container still started, and a project with no hosts has `loc.<name>.com` invented for it, so its proxy block was renamed rather than removed and the ports stayed held. The cached block is deleted rather than skipped, or the next start of any other project would put it back
- **A template can compare two values, so four settings that were never settings are gone.** `db/type_is_mysql`, `db/type_is_postgresql`, `db/type_is_mongodb` and `db/use_default_auth_plugin` were booleans computed in Go and written into the config map as though a user had set them, because the old engine's only test was "does the substituted text contain the word false" — which also meant a path or an image name containing `false` flipped a branch. Conditions are expressions now: `{{{- if and (eq .db.type "mysql") (versionLt .db.version "8.4")}}}`
- **Loops, so a compose file's indentation is not decided in Go.** The hosts used to be joined into one string with the YAML indentation of the file they were going into baked into the separator; they are a list a template ranges over
- **A template that does not parse stops the run and names the file and the line.** The old engine located a closing tag by counting openings, so one unbalanced tag — including one inside a comment — made it abandon the whole file with every conditional unresolved and write the result out anyway. That produced an nginx configuration with six server blocks where one belonged, and nothing said a word
- **Two tests over the whole template tree**: every embedded template parses, and every setting a template reads is a setting madock has. The second is what replaces `missingkey=error`, which cannot be used — a shared snippet asks about `memcached/enabled` on platforms whose configuration has never heard of memcached, so an absent key has to stay falsy. It checks all eleven platforms in a second, where a render could only ever check the one somebody happened to run

**v3.9.0**

Upgrading:
- **The front door follows the language, not every enabled runtime.** A project with php switched on beside another runtime used to render one nginx server block per runtime, all on the same listen and server_name — nginx kept the php one and warned about the rest, so the other runtime's route silently answered from php's document root. Now there is one block and `language` chooses it. If a project runs php alongside node, python, ruby or go, check which one is meant to answer
- **Static files are served `no-cache` instead of a month of `immutable`.** Only files actually present in `public_dir`, which nginx serves itself. Nothing breaks; there are more conditional requests, answered `304` with no body. An installation that wants the long cache back can override the snippet per project from `.madock/docker/snippets/nginx/`
- **Commands refuse to run outside a project.** The project name comes from the directory name, so `stop`, `cli`, `status` and the rest used to act on whatever generated files carried that name. They now say so and exit non-zero. **Scripts are what this breaks**: anything invoking madock from a wrapper directory rather than the project's own. `setup`, `project:clone`, `project:list`, `proxy:logs`, `help`, `version` and `mcp` still run anywhere

Released:
- **The first published release since v3.8.6.** Everything between the two shipped as `-norelease` tags — usable by madock-pro, invisible to anyone who installs from a GitHub Release — so a user upgrading from v3.8.6 receives thirty-seven versions, a hundred and two commits, at once. The bump is minor rather than patch because that range is not only fixes: it adds `project:list`, `proxy:logs`, `cli --service`, the end-to-end suite that now gates every push, and it changes behaviour a project can notice — the front door follows the main service instead of every enabled runtime, static files are served `no-cache`, and commands refuse to run outside a project. The sections below stand unchanged; this entry exists because the version number is what a user sees, and a jump from 3.8.6 to 3.8.51 says nothing about the size of what arrived
- No code changes of its own beyond the version constant

**v3.8.51**

Fixed:
- **`project:list` was blind to exactly the projects it was written to find.** It listed the registry with `os.ReadDir` and kept the entries whose `IsDir()` was true — and that answers about the entry, not its target, so an entry that is a symlink to its project reported false and was dropped before anything read its configuration. On a cluster VM with four projects running, one release after the command shipped, it answered "No projects are registered in this installation": the single wrong answer that reads as good news. Symlinked entries are how an installation looks whenever a project was set up from a temporary checkout, which is also where forgotten entries come from — so `--stale`, whose whole purpose is finding what nobody remembers, could not see the likeliest cases. The predicate now stats the target, and a symlink pointing at nothing is skipped rather than reported, which is right for an entry whose directory is gone
- **Same predicate, same blindness, in `paths.GetDirs`** — and everything that walks `projects/` is built on it: the migrations from v1.4 to v2.4, `project:clone`'s name check, and the project list itself. A symlinked project silently missed a migration. Fixed in the one place, with a test for a real directory, a symlink to one, a plain file and a broken symlink

**v3.8.50**

Added:
- **`project:list`, and it says which entries are no longer real.** Three commands in madock-pro told the user to run it when a provider name was not found, and it did not exist. Worse, the registry drifts and nothing said so: an entry whose source directory has been deleted stays, keeps its port reservations, and keeps a server block in the shared proxy routing its hosts at containers that cannot exist. Measured on the machine this was written on: two of fifty-eight entries were in that state, both still in the generated proxy configuration, one of them holding four ports. `--stale` lists only those and exits non-zero so a script can ask; `--json` for anything that parses. The proxy generator now names them once while it writes the file, which is where the consequence is created. It reports and changes nothing: a stopped project keeps its routing on purpose, and a directory that is not there right now is also what an unmounted disk looks like — removing an entry stays `project:remove`, which refuses unless it is run from the directory the entry records

Fixed:
- **`status` printed `exit status 1` and threw away the sentence that said why.** `docker compose ps` runs through `CombinedOutput`, so docker's own error was already in hand — and `logger.Fatal(err)` reported only the exit code. That one line meant a missing compose file, a daemon that is not running, and a compose file docker refuses to parse, with nothing to tell them apart. It cost about an hour on a server and ended in looking at docker directly, which is the thing madock exists to avoid: the answer, once a diagnostic build printed it, was `open /opt/madock/aruntime/projects/shiplab-shopify-2/docker-compose.yml: no such file or directory` — a ghost project left in the registry after its source was removed, resolved and never mentioned. The path, the error and docker's output are now all in the message. The handler a few lines below, the one that parses the JSON, had always done this correctly, so it was one missed case rather than a habit
- Related, and already in 3.8.49: a command run in a directory that is not a project now says so instead of failing on generated files that happen to carry the directory's name. That covers the first of the three meanings above; this release covers the other two

**v3.8.49**

Fixed:
- **The project guard added in 3.8.48 refused madock-pro's machine commands.** That guard is a property of a command's definition, and a layer built on madock registers definitions madock never sees — around a hundred and ten of them in pro, whole families of which act on the host rather than on a project: `server:*`, `firewall:*`, `dns:*`, `disk:*`, the systemd `service:install|remove|status`, `webhook:*`, `mail:*`, `dashboard:tls:*`, `license:reset`, `init`, and the `:all` group. Measured on a pro binary built against 3.8.48: `firewall:status` and `version` both answered "This directory is not a madock project", and on a server nobody stands in a project directory. `command.AddScopeResolver` lets the layer that owns those commands answer for them, as a rule rather than a list of aliases; the flag stays the fallback, so an unmarked command of madock's own is still project-scoped and a forgotten check still fails loudly. A resolver may also pin a command as project-scoped, which matters where a family is shared: pro's `service:install` is the machine, madock's `service:enable` is a project's containers, and a prefix would have freed both

**v3.8.48**

Added:
- **`proxy:logs`.** The shared proxy is the only container with no way to read its log, and its log is the only place the reason for a 502 is written down — `madock logs` addresses a *project's* container, and the proxy is not a project. Diagnosing one meant reaching past madock to docker, which is the thing the tool exists to avoid. `--service` picks nginx or mailcatcher, `--follow` streams, `--tail` bounds it. An empty log and a healthy proxy print differently: silence is reported as "logged nothing yet" or "not running", because the two are not the same answer. It found the case it was written for within a minute — the reason a livereload port answered 502 was `connect() failed (111: Connection refused)` naming an upstream nothing was listening on
- **`cli --service <name>`.** `cli` always ran in the project's main service, and `bash --service` takes a service but no command (`-c` answers `unknown argument`), so a command in any other container was reachable only by setting `MADOCK_SERVICE_NAME`. The flag is parsed by hand and only in first position, because everything after it belongs to the container and has flags of its own — `madock cli php -v -d memory_limit=1G` still passes both through untouched. An explicit flag beats the environment variable

Fixed:
- **`project:remove` deleted the current directory on a name match alone.** The project name comes from the directory name, `--force` only checks that the caller repeated it, and the removal ends with `RemoveAll` on the directory it was run in. Nothing verified that a project by that name existed, or that this directory was its own: a leftover runtime directory with no configuration was enough to make any same-named directory look removable — and one such leftover was the madock installation itself, whose runtime `src` was a symlink back to the source tree. `project:remove --force --name madock` there would have deleted madock, its git history and every other project's configuration. Three refusals now come before anything is touched — the installation, a directory that is not the project's own (the recorded `path` disagrees), and a project with no configuration — and the list of what is about to go is printed even under `--force`
- **Commands acted on directories that are not projects.** `start` checked and refused; `stop`, `cli`, `bash`, `status`, `info`, `config`, `install`, `composer`, `snapshot:*`, `scope:*` and a dozen more did not, so in a directory whose name happened to match generated files under `aruntime/projects/`, they drove docker compose with those files. In the madock source tree — a directory called "madock" beside leftovers from a version long gone — `restart` failed with `'services[nodejs].extra_hosts' bad host name ''`, blaming a service in a project that does not exist. The dispatcher now refuses a project command outside a project, naming the leftover directory when there is one, and pointing at `setup` when there is not. Commands that belong to no project are marked `Global` (`setup`, `project:clone`, `help`, `version`, `mcp`, `proxy:logs`) — project-scoped is the default, because forgetting the flag on a global command fails loudly while forgetting a check inside a project command fails silently
- **Three platforms never shipped in a built binary.** The `go:embed` list in `docker/embed.go` is written by hand and was last touched when Saleor arrived; BigCommerce, Spree and Sylius came later. From source everything worked, because the templates are read off disk — but a released binary extracts what is embedded, so `madock start` on those platforms died with `docker config file not found: docker-compose.yml`, and Spree and Sylius are first-class entries in the defaults. A test now compares the directory against the embedded filesystem and fails when they disagree
- **One unbalanced conditional tag silently disabled every conditional in a template.** `processConditionals` finds the closing tag by counting openings, so a single unmatched `<<<if` — including one written inside a comment — made it abandon the file with nothing resolved and the result written out as-is. Measured: a generated nginx configuration with raw `<<<iffalse>>>` in it and six server blocks where one belonged. Generation now stops and names the line, and a test walks every embedded template checking the tags balance, so an author's mistake surfaces before any project renders anything
- **`Connection: upgrade` was sent on every request, not only on a WebSocket handshake.** Recreating those hop-by-hop headers is what makes a socket survive a proxy, but as a literal it also told the application that an ordinary GET was switching protocol — malformed, and refused outright by strict runtimes: workerd/Miniflare answers "invalid connection header", undici and Bun the same. The value now comes from a table, "upgrade" only when the client really asked, and nginx does not send a header whose value is empty. Same fix in the shared proxy's livereload and vite listeners, where the constant sat beside the correct map
- **The scheme was rewritten to http on every proxied request.** TLS terminates on the shared proxy; a project's nginx listens on plain 80 inside its container, so `X-Forwarded-Proto $scheme` overwrote the `https` the proxy had correctly reported. Applications then built http URLs for an https request: redirect loops, mixed content, and an OAuth `redirect_uri` Shopify refuses. The proxy's answer now wins, with `$scheme` as the fallback for a request that arrived at the published project port directly
- **The port disappeared from `Host`.** `$host` is the hostname without it, and both hops used `$host` — so an application reached on any port other than the default was told it was reached on the default one, and every absolute URL it generated pointed at port 80. Both now pass the header through, with `$host` kept as the fallback for a request that carries no `Host` at all
- **Static files were served with a month of cache and `immutable`.** The headers landed only on files actually present in `public_dir` — that is, precisely the ones somebody edits by hand — and `immutable` stops the browser revalidating even on reload, so a changed file kept serving its old copy. A long cache is right for names carrying a content hash, and that location matches every `.css` and `.js` by extension. It is now `no-cache`: the file is still stored, and an unchanged one costs a 304 with no body
- **The hidden-file rule answered 403 to ordinary URLs and missed the files it was for.** `location ~* (\.htaccess$|\.git)` is unanchored, so any URI merely containing ".git" was denied — `/media/catalog/logo.github.png` among them — while `.env` was not covered at all, and in one snippet the static-file block matched first, so `/.git/config.css` skipped the rule entirely. Replaced by one rule per template, first among the regex locations, matching a dot only after a slash and exempting `/.well-known/` so an ACME challenge stays reachable. Magento, Shopware and Sylius had no such rule at all; WooCommerce had one but no exemption, and its document root is the project root
- **One nginx server block was emitted per enabled runtime, all on the same listen and server_name.** The templates asked which services were switched on, while the project's own rule for who answers the front door is the language. So php enabled beside another runtime produced two identical listeners, nginx kept the first and warned about the rest, and the route to the application answered 404 from a document root with no index.php — silently. The front door now follows the main service, one include instead of six. `resolveMainServiceEnabled` was answering a blanket "true" for python, golang, ruby and app while every one of them is rendered into compose behind its own switch, which is how `depends_on` came to name a service the file did not contain
- **`@proxy` spoke HTTP/1.0 to the application** while the `location /` beside it spoke 1.1, so every asset not present on disk opened a new connection and a chunked response had to be delimited by closing it. It now speaks 1.1, and `X-Forwarded-Host` is sent beside the other forwarded headers
- **The php platforms told the application every request was secure.** `fastcgi_param HTTPS on` was a constant in WooCommerce, Shopware and Sylius, with `SERVER_PORT "443"` beside it in the last two — so a stand reached over plain http at its published port built https URLs, set secure cookies and redirected accordingly. HTTPS now follows the request (`off` rather than an empty value, which WordPress reads as secure), and the hardcoded port is gone: `fastcgi_params` sets it from the request
- **`worker_priority -10` never once applied, and said so twice per start.** Lowering a nice value needs `CAP_SYS_NICE`, which is not in a container's default capability set, so every proxy start logged `[alert] setpriority(-10) failed (13: Permission denied)` per worker — alert being the loudest level short of emerg, in the one log where a real fault has to be visible. The proxy ran at ordinary priority throughout. Removed rather than granted the capability: nginx here is epoll and I/O, what starves under load is php-fpm and the database, and the container-native way to express priority is a cgroup weight

**v3.8.47**

Fixed:
- **A named platform version installed the wrong Composer — on two platforms.** `setup --platform-version=2.4.8` left Composer at the default for an *unknown* version, which is the empty string: the table that knows ≥2.4.2 needs Composer 2 was read only when the version was detected from the project or came from a preset. Magento 2.4.8 under Composer 1 cannot resolve a single package, so twenty minutes of downloading ended with every package "could not be found". Shopware had the same hole one condition further along — its table was read only while PHP was still empty, so naming a version *and* a PHP version skipped it and left Composer, the database and the search engine unset. Every install driven by a script or by CI goes through exactly that flag
- **`madock setup -d` never downloaded a Medusa project.** The download runs after the containers are up, because git runs inside the node container — and starting them makes the daemon create the bind-mount sources that do not exist yet, `storefront/` among them, owned by root. The emptiness check then said "not empty" about a directory madock had just created, skipped the clone, and left the install to run against nothing: yarn had no package.json and the failure surfaced two commands later as "npm error could not determine executable to run". On a clean directory, which is the only way anybody starts. The storefront clone was skipped for the same reason and left a project with a backend and no shop. Those directories are now created by madock before the stack starts, so they belong to whoever ran it, and an empty directory is no longer mistaken for a project
- **Node dev servers ran as root, and the files they wrote could not be deleted.** The images remap their `node` user to the host uid — that is what makes files come out owned by the developer — but nothing ever switched to it, so `yarn dev` ran as root and everything it produced, `.medusa/client/` included, belonged to root inside the user's own project. `project:remove`, whose whole promise is to leave nothing behind, then stopped with "permission denied". The entrypoint now keeps root only for the part that needs it — waiting for code madock writes after the container starts — and drops to `node` for the long-running command, which is the arrangement php-fpm has always had here. Same change in the storefront image; `project:remove` reclaims ownership before deleting, for projects created by the old behaviour
- **`start --with-chown` failed on every project without PHP.** The directories to chown are declared per platform and chained into one command, and `custom` declares `/var/www/.composer` — which does not exist in an image with no PHP. The chain stopped there and took the command with it, so the flag exited 1 *after* the containers were already up. Each directory is now chowned only if it is there; a chown that genuinely cannot run still fails
- **`db:import --reset-gtid` could not work on any default project.** The flag issues `RESET MASTER`, and madock's database images run with binary logging off, so the server answers `ERROR 1186: Binlog closed` — whereupon the import stopped, having restored nothing, printing `exit status 1` with the server's own message discarded. It now says there is no GTID state to reset and carries on with the import, and the interactive "run RESET MASTER and retry" branch names the option that does work instead of dying the same way
- **`proxy:reload` reported success with the proxy stopped.** `nginx -s reload` execs inside a container that has to be running already; with it down the exec failed, the configuration on disk was not applied, and the command printed `Done` and exited 0 regardless
- **`db:info` allocated ports while describing a project.** It asked the registry for "the port, or a new one", so asking about a project that had never started reserved a port for every service it might one day run — and then printed one nothing was listening on. It now looks the port up and says "not published yet" when there is none, which is what `madock info` already did

**v3.8.46**

Fixed:
- **`db:execute` has never worked against PostgreSQL.** It ran `psql -U … -h … <db> -c <query>` and offered no password, which psql does not accept on the command line — so every query ended in `fe_sendauth: no password supplied`. `db:export` and `db:import` already passed `PGPASSWORD` and worked; this one call site did not, which is why the gap survived. Found on our own Packeton server, where the answer to any database question was to open a shell in the container by hand
- **A quote in a password, or in a query, broke the command it sat in.** The PostgreSQL paths build a shell line, and every value went into it raw: an apostrophe ended the string early and handed the rest to the shell. For `db:execute` the query is the sharper case — `WHERE name='x'` is ordinary SQL and would have been mangled. Quoting now goes through `src/helper/cli/shell`, one place with tests that hand the result to a real `/bin/sh`

**v3.8.45**

Fixed:
- **`debug:enable` reported success on projects it cannot debug.** Every debug command writes `php/xdebug/*`, so on a nodejs, python, golang or ruby project it set a value nothing reads, rebuilt the project, and finished with a tick — debugging absent, and the command that was supposed to arrange it saying otherwise. It now says the language has no debugger wired up and changes nothing. Wiring the others up is a piece of work rather than an oversight: xdebug connects out to the IDE, while Node, Python, Ruby and Go debuggers listen, so each needs a published port, an allocation from the registry, and its process started under the debugger — and Go additionally needs `SYS_PTRACE`

**v3.8.44**

Fixed:
- **`status` answered the cron question from the configuration.** It reported "Cron is running" whenever `cron/enabled` was set, which says what was asked for and not what happened — and starting cron is a command executed inside the container, which can fail. A project could have nothing on a schedule and a status that said otherwise, with the consequence surfacing whenever somebody noticed a mail or a reindex missing. It now asks the container. `--json` gains `cron_running` beside `cron_enabled`, because the two are different questions and their disagreement is the interesting case; the text says "Cron is enabled but not running" when they disagree

**v3.8.43**

Fixed:
- **`rebuild` reported success while a service it had just started was already dead.** `start` has said this for a while; rebuild did not, and rebuild is what runs when a service is enabled — the moment a new service first shows whether it can run at all. Found with an image published only for amd64: on an arm64 host the container was created, started, and killed on exec, and the rebuild finished with a tick. It now lists services that are not running, with their exit code, and points at `madock logs`. The check waits two seconds first: a container that cannot exec its entrypoint is reported as started and is gone a breath later, so asking immediately sees it running and gives exactly the answer the check exists to prevent

**v3.8.42**

Fixed:
- **`start` reported success while creating nothing.** It wakes existing containers when the configuration has not changed, and waking nothing succeeds: the command returned in a fraction of a second, said the project was started, and left the machine empty. The fallback only ran when the wake failed, which it never does. Any project whose containers were removed while its configuration stayed put was affected; a freshly cloned project always was, since `project:clone` removes them to load the copied data. `start` now creates containers when there are none to wake — and does not count the snapshot helper, which lives in the same compose project, is not a service of the project proper, and is exactly what clone leaves behind

**v3.8.41**

Fixed:
- **A removed project stayed in the proxy.** Its server block, pointing at a container that no longer existed, survived until something else happened to regenerate the file. Removal is the one moment routing should change: a stopped project keeps its block on purpose, because it is coming back and rewriting the configuration every other project is served through would be churn for nothing. The regeneration runs on behalf of a project that still exists — naming the removed one brings its registry entry and its port reservation straight back, which is how both of those were found
- **`project:clone` could not copy a running database.** It read the source's live containers deliberately, so as not to interrupt it, and the check added for snapshots then refused the torn result — so clone stopped working for any project with a database, which is every project worth cloning. The source now stands still for the copy and is read from the helper container, the same trade `snapshot:create` makes
- **`--domain-suffix` documented a form that cannot work.** The suffix usually starts with a dash, and `-s -update` is read as two flags: "missing value for -s". The help now shows `-s=-update`

**v3.8.40**

Fixed:
- **`status` reported a stack with exactly one service as empty.** Compose prints one JSON object per line when asked for a stack's status, so the count decides the shape of its output, and the parser only wrapped it into an array when it could see a `}{` boundary between two of them. With one service the bare object failed to decode into a list — and the error was discarded, so the answer was "No services found" rather than a complaint. Reported from a server where disabling mailpit left the proxy with nginx alone: `status` called the proxy empty while every site it was serving stayed up. The parser now handles none, one, many and an already-formed array, and a decode failure is said out loud instead of being turned into "nothing is running"

**v3.8.39**

Fixed:
- **Every `db:*` command authenticated as root, which a shared database refuses.** For a project's own server that is right — the client runs inside the container, root is reachable on localhost, and export needs to lock tables while import needs to create them. For a database belonging to another project the client connects over the network from a different container, and MySQL grants root to localhost only: the refusal arrives before any password is checked, as `Host '172.21.0.2' is not allowed to connect`, which reads like a network fault and is an account one. `db:execute`, `db:export` and `db:import` now use the account `shared-db:connect` created and granted. Export adds `--single-transaction` for a shared database, because a consumer is not granted LOCK TABLES and a snapshot read is the better choice on a server other projects are using anyway

**v3.8.38**

Added:
- **`php/sendmail/from`** — the envelope sender. msmtp refuses to send without one (`envelope-from address is missing`, exit 78) and there is no msmtprc in the image to hold a default, so a plain four-argument `mail()` call failed while Magento and Laravel were fine: a mail transport passes its own sender. Anyone testing their site reaches for `mail()` first, gets exit 78, and has no reason to suspect the sender is the missing part. Empty by default, because no address is right for every project and a wrong one costs more in deliverability than a clear failure does

**v3.8.37**

Added:
- **`configs.SetDefaultOverride`** — a seam for editions whose answer to a setting differs from the community one. It applies after the embedded defaults and before the user's `config.xml`, which is the only layering that works: the edition chooses the default and whoever edits the file still overrides it, so turning something back on never needs a different binary. Written for mailpit, which madock-pro disables by default — a mail interceptor with no authentication is what a developer wants and the opposite of what a server wants

**v3.8.36**

Fixed:
- **`sendmail_path` was written into php.ini whatever the configuration said**, hardcoded to mailpit's port. With mailpit disabled every `mail()` call handed the message to msmtp, which connected to a port nobody was listening on: mail did not arrive and nothing said so, because the setting looked correct and only the port was wrong. It is now written only when `php/sendmail/enabled` is on, and `php/sendmail/host` and `php/sendmail/port` point it somewhere real — port 25 on the host for a local postfix, for instance. Editing php.ini inside a running container was never an alternative: the image is rebuilt from a template and the edit goes with it, which is how a working mail configuration disappears at the next rebuild
- **Mailpit's web interface was published on every interface.** It has no authentication and shows every message every project has sent, so on a server that is everyone's mail readable by anyone who can reach the port. It now binds to loopback, and `proxy/mailpit/interface_ip` opens it deliberately for those who want it open. The SMTP port stays on every interface and has to: the php containers reach it through `host.docker.internal`, which resolves to the host gateway rather than to loopback. What arrives there is stored and never forwarded, so an open SMTP port means somebody can fill a developer's inbox, not relay through it

**v3.8.35**

Fixed:
- **The certificate fix in 3.8.34 covered half the problem.** The check sat in the branch taken when the proxy is already running, and the more common case is the other one: `restart` is a stop followed by a start, stopping the last project takes the proxy down with it, and the `conf-cache` marker survives — so nothing reissued the certificate. A project whose host was edited kept serving the old name. The host set is now compared before that branch, so both halves are covered
- **`./test/e2e/e2e.sh run -run 'A|B'`** passed the pattern into a shell unquoted, so anything after a pipe was run as a command

**v3.8.34**

Fixed:
- **A second project got a route but no certificate.** The TLS certificate covers every project at once, and it was only ever issued when the proxy started for the first time. `proxy.conf` is regenerated and reloaded on every start, so a newly added project was routed immediately — and served over HTTPS with a certificate that did not name it, which every browser refuses. It is now reissued whenever the set of hosts changes, and the proxy reloads to pick it up. Found by the end-to-end suite on the first run that started two projects
- **`setup -y` was not non-interactive.** On a machine without `certutil` the SSL step printed a question and waited on stdin, so `madock setup -y` in a provisioning script hung with nobody there to answer. The flag is read there now, and an unreadable stdin means "continue without SSL" rather than a fatal error — skipping SSL leaves a working project, installing packages nobody agreed to does not
- **A debug log that could not be written ended the command.** It was placed next to the binary, ignoring `MADOCK_EXEC_DIR`, and a failed write called `log.Fatal` — so an installation directory without write permission turned every error into `open debug.log: read-only file system` and hid the error being reported. It now goes to the installation directory, and a failure warns once on stderr

Added:
- **An end-to-end suite** in `test/e2e`: it creates a project with the real binary, starts it, asks whether it is running, talks to its database, checks the proxy serves the right certificate, and takes it down. It runs in a Lima VM rather than on the developer's machine, because the proxy stack is named in the templates rather than derived — a test that starts a project operates on the same containers your own work uses. `./test/e2e/e2e.sh up`, then `run`. See [docs/testing.md](docs/testing.md)

**v3.8.33**

Fixed:
- **On the `custom` platform the Node container's Dockerfile was generated only for PHP projects.** The `nodejs` service is rendered into docker-compose.yml whenever `nodejs/enabled` is set, whatever the language — so a Python, Go, Ruby or language-less project with Node enabled had a compose service pointing at a file nobody wrote: missing on a fresh project, and worse on an older one, where a copy left over from an earlier madock was silently built instead. It is now generated whenever the service exists, except when the language is `nodejs` — there the Node container *is* the main service, its Dockerfile already comes from the language template, and that template carries cron where the service one does not

**v3.8.32**

Added:
- **`nodejs/script` and `nodejs/script_type`** — the Node container can be told what to run instead of having it guessed. `script_type` is `auto` (a name `package.json` declares is a script, anything else is a command, and madock says which it chose), `package` (always a script — a missing one stops the container with an explanation instead of failing at exec) or `command` (always a shell command; a path to a file is a command). A script still goes through the package manager the lockfile names
- **`nodejs/browser_libs` and `php/browser_libs`** install the shared libraries a headless Chromium needs, off by default. The list is not written down: the image asks Playwright for it at build time, because the package names differ per distribution and move between releases — `libasound2` became `libasound2t64` in Debian trixie and Ubuntu 24.04, so a hand-pinned list breaks the build the day the base image is bumped. Installing them at runtime cannot work anyway: madock execs as a non-root user, so Playwright stops at a sudo password prompt, and apt's work is lost at the next rebuild

Fixed:
- **The Node entrypoint always preferred `dev`, which is the wrong process on a server.** For a Shopify app `dev` is `shopify app dev` — it prints a verification code and waits for a human. Nobody logs in, it gives up, and the container dies with it, because that command is its main process. Now `start` is preferred when `NODE_ENV` is `production`
- **`nodejs/env` replaces a hardcoded `NODE_ENV: "development"`** in the compose file. It was set that way for every project, laptop and server alike, so the entrypoint had nothing to read and libraries that branch on it were always in development mode
- **`start` no longer reports success when a container is already dead.** `docker compose up` returning zero only says the containers were created; a main process that is not a daemon is gone seconds later. Services that are not running are now listed with their exit code, and `madock logs -s <service>` is offered. Until now the only signal was the log, and nothing pointed at it — `status`, asked in between, honestly answered "running"

**v3.8.31**

Fixed:
- **The proxy.conf extension point was a single slot, and a second consumer would have silently disabled the first.** `SetProxyConfTransformer` was last-writer-wins, which is fine for one caller and a trap for two — and two is the normal case, since routing and TLS have nothing to do with each other and both need the generated file. `AddProxyConfTransformer` appends instead, transformers run in registration order each seeing the previous one's output, and a transformer returning an empty string no longer truncates the file for the ones after it. `SetProxyConfTransformer` is kept for callers that mean "this and nothing else"

**v3.8.30**

Changed:
- **`snapshot:create` stops the project's containers, copies, and starts them again.** It reads from the helper container that mounts the same volumes — the mirror of what `snapshot:restore` already did. A database's data directory copied out of a running server comes out torn: an archive that looks like a backup and may refuse to start. There is no way around that from inside the running container, so the project stands still for the length of the copy. A project that was already stopped is left stopped
- Containers are **stopped, not removed**, so coming back is a start and not a rebuild. `docker.Stop` and `docker.Start` are new for this; only `Down`, which removes them, existed before
- `project:clone` still reads the source project's live containers on purpose — cloning must not interrupt whatever the source is doing — so it accepts the risk `snapshot:create` refuses

Fixed:
- [docs/snapshot.md](docs/snapshot.md) claimed snapshots are stored in `~/.madock/projects/{name}/snapshots/`, that the database part is a dump, and that `vendor/` and generated files are excluded. The path is `{madock_dir}/projects/{name}/backup/snapshot/{snapshot_name}/`, the database part is a copy of the data directory, and nothing is filtered out of the project directory — a snapshot is the size of the project, which the old table hid

**v3.8.29**

Fixed:
- **`snapshot:create` failed on a project whose files changed while it ran.** tar exits 1 when a member was written while being read — the archive is complete, those members are not — and the command treated it as a fatal error. Worse, the shell chain was `tar -czf /tmp/x.tar.gz . && cat /tmp/x.tar.gz`, so exit 1 skipped the `cat` and produced an empty archive alongside the error. A project with its containers up rotates a log or writes a cache file, which was enough
- tar now streams to stdout instead of writing a second copy inside the container and cat-ing it, so its exit status reaches madock rather than being swallowed. Status 1 is interpreted; 2 and above stay errors, and `--ignore-failed-read` is deliberately not passed, so an unreadable file cannot arrive disguised as a changed one
- **Project files and the database data directory are treated differently, on purpose.** For the files, a changed member is expected and the snapshot is kept with a warning. For a data directory it is not survivable: those pages only mean anything together, so the archive is deleted, the run stops, and it points at `db:export` for a dump that can be relied on. Keeping it would have left something indistinguishable from a good snapshot for `snapshot:restore` to write back
- `cd` into the directory is checked separately, so a missing path cannot land in the exit-1 branch and pass off an empty archive as a complete one

**v3.8.28**

Fixed:
- **`--with-chown` and `snapshot:create` reached into a `php` container on projects that run none.** Both hardcoded the service name, so on a Node, Python, Go or Ruby project they died with `No such container` — `snapshot:create` unconditionally, `--with-chown` whenever containers had to be recreated. They now resolve the service that runs the application code. Verified on a `language: none` project, where the main service is `app`
- The rule that maps a language to its main service lived in two copies, in packages that could not import each other. It is now `configs.ResolveMainService`, and both copies delegate — the copy that did not get updated is how this bug existed at all
- **A `--db-service-name` that is not a database service is now named as such.** Only `db` and `db2` hold one; anything else used to be assumed to exist, and the user got Docker's `No such container` instead of the actual mistake. Resolvers are still asked first, so an installation that adds a database under another name keeps working
- **`db:export` from a shared database no longer calls the file `local_`.** The dump sits in this project's backup directory but holds another project's schema — and everything every other consumer keeps in it. The name now says `shared-<provider>_`, which matters before that file is restored or uploaded somewhere
- `snapshot:create` on a project whose database is shared now names the provider and says why the snapshot stops at the files

**v3.8.27**

Fixed:
- `Fingerprint` answered with the hash of the empty set for a project whose stack had never been generated. That is a stable, plausible-looking value, and `RecordApplied` would happily store it as "what the containers were built from" — after which the first real render read as a change and rebuilt a project that had only just been created. A missing runtime dir now has no fingerprint, and nothing records a non-answer
- `nodejs/major_version` survived when `nodejs/version` was empty, so a value left over from an older config decided which Node got installed while looking deliberate. The derived key is dropped when there is no source: the placeholder stays unsubstituted and the image build fails where the mistake is

**v3.8.26**

Fixed:
- **`config:set` was only picked up by `rebuild`.** The generator returned early whenever the `conf-cache` marker existed, and only `rebuild` and `project:clone` removed it, so `start` and `restart` kept the previously rendered Dockerfile. The change sat in `config.xml`, invisible to docker, and the environment ran the old one until somebody happened to rebuild. The compose files and build context are now rendered on every up — it is deterministic file IO over a few dozen templates
- **Rendering alone was not enough.** `docker compose start` reads the compose file for names, not for content: it wakes the existing containers and ignores a changed image or service definition. `start` now compares a fingerprint of the generated stack against the one the containers were created from, and recreates them when they differ, saying so first
- The fingerprint hashes the **generated files**, not the config, so a key no template reads — an SSH host, a cron flag — renders identically and costs nothing. Symlinks out of the runtime dir (the project source, `~/.composer`, `~/.ssh`) are skipped: following them would make every source edit look like a stack change
- A project with no recorded fingerprint is adopted silently rather than treated as changed, so upgrading madock does not rebuild every project on its next start
- The same early return silently ignored a newly added `docker-compose.<GOOS>.yml`, which [docker_compose_override.md](docs/docker_compose_override.md) says is picked up on every start. It now is

Changed:
- **`nodejs/major_version` has one owner.** It was written by twelve platform presets at setup, recomputed by two generators while rendering, and stored in `config.xml` where it could only go stale. It is now derived from `nodejs/version` on every config read, and `config:set nodejs/major_version` is refused with the option to set instead — it used to look accepted and change nothing

**v3.8.25**

Fixed:
- **The default OpenSearch version was `2.5`, a tag no registry publishes.** `opensearchproject/opensearch` ships `x.y.z` only, and the value goes straight into `image:` and into the data volume name, so a project that took the default failed its first `start` on the pull. Now `2.19.1`. This is not cosmetic: `service:enable opensearch` does not ask for a version, so whatever sits in the defaults is what gets pulled
- **A first-time setup overwrote the platform's version matrix with the embedded defaults.** `tools.PopulateFromConfig` exists to show a reconfigured project the versions it already runs, but it ran on new projects too — where the caller has no project config and passes the general config, which starts from those defaults. A Shopware 6.7 project came out pinned to the default OpenSearch and Elasticsearch versions instead of the 2.8.0 and 8.11.14 its matrix names. It is now a no-op outside reconfigure mode, which is set exactly when the project already has a `config.xml`

**v3.8.24**

Added:
- `src/helper/dbtarget` — one place that answers which container a database command runs in and with which host and credentials. `Register` lets an installation answer it instead, which is how a project whose database is owned by another project gets working `db:*` commands without every command learning about it

Fixed:
- `db:execute`, `db:export` and `db:import` built the container name from the current project and died with Docker's `No such container` on a project that does not run its own database. All three now go through `dbtarget`, so they follow the database wherever it lives
- `db:info` printed `host: db` and empty credentials for such a project instead of admitting there is nothing there. It now describes the real target, marks a shared one with its provider, and omits a `db2` block for a project that has no `db2`
- `--db-service-name db2` connected to `db2` with `db`'s root password and `db`'s schema name — the commands read `db/*` for credentials no matter which service was asked for. Each service's own keys are used now
- `snapshot:create` aborted on a project without its own `db` service. It skips the database and says so. A snapshot copies a container's data directory, so a database owned by another project is deliberately not included: restoring that copy would overwrite the data every other consumer of that server reads

**v3.8.23**

Changed:
- Builds against Go 1.25. The 1.23 line went out of support on 2025-08-12 and 1.24 on 2026-02-10 — Go keeps only the two newest majors alive and has no security-only tail, so both were already receiving nothing. The `toolchain` directive is gone with the bump: it named the same version as `go`, and `go mod tidy` drops it as redundant. Building now needs Go 1.25 or newer, which `GOTOOLCHAIN=auto` fetches by itself; the published binaries are unaffected

**v3.8.22**

Fixed:
- `remote:sync:db`, `remote:sync:media` and `remote:sync:file` could not use a passphrase-protected SSH key. The client never looked at `SSH_AUTH_SOCK`, so it asked for the passphrase itself and died with `operation not supported by device` wherever there is no TTY — a cron job, a hook, an agent-driven session. Plain `ssh host` works on the same host because ssh-agent holds the key, which is what made the gap easy to miss. The agent is now offered as the first auth method, with the key file kept as a fallback
- The key-file method is built lazily when an agent is present. It used to be constructed up front, which meant reading and parsing the key — and prompting for its passphrase — before the handshake ever reached the agent, so ordering the methods alone would have fixed nothing
- `AgentAuth()` is exported: enterprise replaces the whole `ssh.ClientConfig` through `SetSSHConfigProvider`, so a fix confined to the open-source path would have left every pro installation prompting

**v3.8.20**

Added:
- `madock version` (also `madock --version`, `madock -v`) — the version was only reachable by reading the binary's own migration output before. Supports `--json`

Fixed:
- `madock <command> --help` and `-h` printed a bare "help requested by user" instead of the command's help. go-arg answers both with a sentinel error rather than output, and it was going straight to the fatal logger. Both spellings now render the same block as `madock help <command>`

**v3.8.19**

Added:
- Memcached as a first-class service, off by default: `madock service:enable memcached` / `service:disable memcached`. Container `memcached` on port 11211, reachable from the other containers by service name, not published to the host. Joins the `isolated` network in isolation mode. Included in every platform's compose file, so enabling it is never a silent no-op
- Configurable via `memcached/repository`, `memcached/version` (`1.6.39-alpine` by default, also selectable interactively or with `service:enable memcached --version`), `memcached/memory` (`-m`, 256 MB) and `memcached/max_connections` (`-c`, 1024)
- `php{version}-memcached` is installed into the PHP image only while the service is enabled, and tolerates a missing package — ondrej lags a release behind for the newest PHP versions and a hard failure there would break the image for everyone. Documented in [docs/memcached.md](docs/memcached.md)

Fixed:
- `service:enable <svc> --version` never worked: go-arg intercepts `--version` before it reaches the struct field and aborts with "version requested by user", so valkey, artemis and xdebug could only be versioned through the interactive picker. The flag is now `--service-version`

**v3.8.17**

Fixed:
- PHP image Node.js install: fail fast when the NodeSource setup script cannot be fetched instead of silently falling back to the Debian `nodejs` package (which ships without npm and broke `npm install -g grunt-cli` with `npm: command not found`). Repo setup, `apt-get update` and the install now run in one layer, with an `apt-get install -y npm` fallback and a `node -v` / `npm -v` sanity check

**v3.7.15**

Fixed:
- Add MariaDB 11.8 to setup wizard version picker (Magento 2.4.9 default, was missing)

**v3.7.14**

Fixed:
- Add PHP 8.5 to setup wizard PHP version picker (was missing despite Magento 2.4.9 defaulting to it)

**v3.7.13**

Changed:
- Setup wizard: platform picker puts Custom first, drops recommended marker — madock is multi-platform, no single choice should be highlighted
- Setup version pickers: refresh all language/runtime/service choices after platform detection
- Interactive selector: clamp box to terminal width, truncate long options so TUI doesn't wrap on narrow terminals

Fixed:
- ProcessSnippets: support nested includes, fix cron snippet
- BigCommerce Catalyst: bump default Node to 24.10.0
- BigCommerce stencil install: run global stencil-cli install as root
- Shopify laravel-shopify Download: pass --no-scripts to composer create-project
- Setup Download: run all scaffolding inside project containers (fixes code-not-mounted race)
- Shopware: init-chown, permissive umask, scheduled-task cron, messenger sidecar

**v3.7.12**

Added:
- BigCommerce platform support with 4 SDK/framework presets:
  - `--preset catalyst` (Node 22 + Catalyst monorepo) — official headless Next.js storefront. Pnpm + turbo monorepo. Install pre-installs pnpm globally in the nodejs image, clones `bigcommerce/catalyst`, runs `pnpm install` across workspaces, rewrites root `scripts.dev` to filter to `./core` with `-H 0.0.0.0` forwarded via `--` (turbo rejects bare `-H`), and parks `scripts.dev` as `scripts.dev:catalyst` so the container stays up until the user adds real store credentials to `core/.env.local` and runs `npm run dev:catalyst` (Catalyst's pre-dev `generate` step needs the store hash to fetch the GraphQL schema)
  - `--preset stencil` (Node 22 + Stencil CLI) — Cornerstone-based theme dev. Clones `bigcommerce/cornerstone`, runs `npm install`, installs `@bigcommerce/stencil-cli` globally, parks `scripts.dev` (Stencil needs interactive `stencil init` + API token entry)
  - `--preset api-php` (PHP 8.3 + MariaDB + Redis) — `bigcommerce/api` Composer SDK for backend integrations. `composer init` scaffolds pinned to `^3.3`, `composer install` (or update when no lock yet)
  - `--preset app-node` (Node 22) — `bigcommerce/sample-app-nodejs` (Express + Next.js with OAuth handshake) clone. Parks `scripts.dev` as `scripts.dev:bc` (app dev needs interactive Developer auth + ngrok tunnel)
- `madock bigcommerce <cmd>` (alias `madock bc`) — preset-aware container exec. Routes to nodejs container for catalyst/stencil/app-node, php container for api-php
- BigCommerce env writer mirrors Shopify's preset-branching: Node-only presets drop PHP/DB/Redis; PHP preset keeps full stack. Default DB name `bigcommerce` for the PHP preset, redis on by default with project-level override honored
- Auto-detection: composer.json with `bigcommerce/api` or legacy `bigcommerce/bigcommerce-api-php` → api-php preset. package.json with `@bigcommerce/catalyst-core` / `-client` / `checkout-sdk` → catalyst, with `@bigcommerce/stencil-cli` or `name=cornerstone` → stencil
- `docker/bigcommerce/nodejs/Dockerfile` pre-installs pnpm globally so the entrypoint's `pnpm dev` works without the `corepack enable` root-permission dance
- `docker/bigcommerce/nginx/conf/default.conf` swaps between FastCGI (PHP backend) and a Node-only proxy block based on `php/enabled` / `nodejs/enabled`. Node branch uses an in-block `map $http_upgrade $node_connection_upgrade` for Connection-header handling (matches the Shopify Node-preset pattern). For catalyst / stencil / app-node `main_service_port=3000`
- `MakeConfBigcommerce` materialises only the Dockerfiles the selected preset uses — Node-only presets skip PHP/DB/Redis Dockerfiles entirely. Same pattern as `MakeConfShopify`

Docs:
- `docs/bigcommerce.md` — preset matrix, install pipeline per preset, services-per-preset table, switching presets, gotchas (Missing store hash, turbo -H bug, stencil auth, app-node CLIENT_ID)
- `README.md` — BigCommerce added to supported platforms list and key features

**v3.7.11**

Shopify presets — post-tracer hardening:
- Hydrogen now renders `/` end-to-end. Two issues that fought the
  initial tracer:
  - Hydrogen's Oxygen plugin ignores user-set `server.allowedHosts`
    in vite.config.ts. Install now also patches Vite's internal
    `isHostAllowedInternal` in `node_modules/vite/dist/node/chunks/
    node.js` to short-circuit to `true`. Marker-gated so re-running
    install is idempotent. Pure dev convenience — patches
    node_modules only
  - Hydrogen's Miniflare/undici client rejects `Connection: upgrade`
    on non-WS requests ("invalid connection header"). Project nginx
    for Node-only presets now uses an in-block `map $http_upgrade
    $node_connection_upgrade { default upgrade; '' ''; }` so the
    Connection header is empty for plain HTTP and only forwarded as
    `upgrade` for genuine WS handshakes
- app-remix template clone switched from `npm init @shopify/app@latest`
  to `git clone shopify-app-template-remix` — the npm init argument
  parser changed across CLI versions in 2024 and was producing an
  empty directory. Install also parks the template's `dev` script
  (which is `shopify app dev`, an interactive Partner-CLI command)
  as `dev:shopify` and replaces `dev` with `sleep infinity` so the
  container stays up after install; users start the real dev server
  via `madock bash` + `npm run dev:shopify` (needs interactive
  Shopify Partner auth + tunnel — can't run from a non-tty container)
- laravel-shopify install now correctly rewrites Laravel 11+
  `.env` files where the DB_* lines ship commented out by default
  (Laravel switched to SQLite default in 2024). Sed patches handle
  both `^DB_*=` and `^# *DB_*=` forms
- api-php composer require pinned to `^6.0` (v7 isn't published yet
  on Packagist). Install also picks `composer install` vs `composer
  update` based on whether composer.lock exists — fresh `composer
  init` projects only have composer.json so update is correct
- `docker/shopify/php/Dockerfile` no longer hand-rolls the yarn
  install via GPG keyserver (which was failing with `gpg: keyserver
  receive failed` on every fresh build). Uses the shared
  `snippets/dockerfile/php/nodejs` snippet that installs Node + Yarn
  via npm when `php/nodejs/enabled` / `php/yarn/enabled` are set

Added:
- Shopify platform now ships with 4 SDK/framework presets so users can pick a stack at setup time instead of inheriting the legacy PHP-only default:
- Shopify platform now ships with 4 SDK/framework presets so users can pick a stack at setup time instead of inheriting the legacy PHP-only default:
  - `--preset hydrogen` (Node 22 + Remix on Vite, TypeScript) — official headless storefront, deploys to Shopify Oxygen
  - `--preset app-remix` (Node 22 + Remix + Prisma/SQLite) — official embedded Shopify App template
  - `--preset api-php` (PHP 8.3 + MariaDB + Redis) — raw `shopify/shopify-api` Composer SDK for backend integrations
  - `--preset laravel-shopify` (PHP 8.3 + Laravel + Node + MariaDB + Redis) — full Shopify App on Laravel via `Kyon147/laravel-shopify`
  Interactive preset wizard mirrors the Medusa/Saleor/Spree/Sylius flow. Aliases honored (`node` → hydrogen, `app`/`remix` → app-remix, `php`/`api` → api-php, `laravel` → laravel-shopify)
- Shopify env writer rewires the container stack per preset. Node-only presets (hydrogen, app-remix) drop PHP/MariaDB/Redis entirely — no zombie containers and no `FROM mariadb:{{{db/version}}}` build errors when the DB block is skipped. PHP presets keep the legacy full stack
- Shopify install handler dispatches per preset:
  - Hydrogen: `npm install`, patches `package.json` (adds `--host` to the `dev` script so Vite binds 0.0.0.0 instead of 127.0.0.1), patches `vite.config.ts` (adds `server.allowedHosts: true` so the project's `*.test` host doesn't trip Vite's host-header guard), then restarts the nodejs container
  - app-remix: `npm install` + `npx prisma generate && npx prisma migrate deploy` (Prisma uses SQLite by default — no DB container needed)
  - api-php: `composer install` (or `composer update` when no lock present) against a `composer init`-generated project pinned to `shopify/shopify-api:^6.0`
  - laravel-shopify: rewrites Laravel `.env` (APP_URL, DB_CONNECTION=mysql, DB_HOST=db, DB credentials from project config), `composer install`, `composer require kyon147/laravel-shopify`, `php artisan key:generate`, `migrate`, `vendor:publish --tag=shopify-config --tag=shopify-routes`
- Per-preset `DownloadShopify` scaffolders:
  - hydrogen: `npm create -y @shopify/hydrogen@latest -- --path . --quickstart --language ts --no-install-deps`
  - app-remix: `git clone --depth 1 https://github.com/Shopify/shopify-app-template-remix.git .` (the npm init argument parser changed across CLI versions in 2024 and was producing an empty directory; cloning the upstream template is the same content without the wizard step)
  - api-php: `composer init --no-interaction --require=shopify/shopify-api:^6.0`
  - laravel-shopify: `composer create-project --no-install laravel/laravel .`

Changed:
- `docker/shopify/docker-compose.yml` wraps the DB/Redis/RabbitMQ/Grafana service block in `<<<if{{{php/enabled}}}>>>` so Node-only presets don't try to build a DB image with un-substituted `{{{db/version}}}` templates
- `docker/shopify/nginx/conf/default.conf` swaps between FastCGI (PHP backend) and a Node-only proxy server block based on `php/enabled` / `nodejs/enabled`. The Node block declares an in-block `map $http_upgrade $node_connection_upgrade { default upgrade; '' ''; }` so the `Connection` header is empty on plain HTTP (Hydrogen's Miniflare/undici rejects `Connection: upgrade` on non-WS requests with "invalid connection header") and only `upgrade` for genuine WS handshakes. For hydrogen / app-remix the env writer pins `main_service_port=3000` to match the dev server upstream
- `MakeConfShopify` only materialises the Dockerfiles the selected preset actually uses (PHP, NodeJS, DB, Redis are now conditional), so Node-only presets don't ship a half-substituted db.Dockerfile that breaks `docker compose build`
- Added `nodejs.yml` snippet include to `docker/shopify/docker-compose.yml` + `docker/shopify/nodejs/Dockerfile` so the Node service has a real Dockerfile to build from

Docs:
- `docs/shopify.md` rewritten: preset matrix, install pipeline per preset, per-preset services table, switching presets, gotchas (Hydrogen Vite allowedHosts, Remix Partner auth, Laravel routes 404)

**v3.7.10**

Added:
- Sylius platform support: `madock setup --platform sylius` (PHP 8.3 / Symfony + MariaDB + Redis + Node + Yarn baked into the PHP image for Webpack Encore). `madock sylius <cmd>` runs `php bin/console <cmd>` inside the PHP container. `madock install` writes `.env.local` (DATABASE_URL with `serverVersion=mariadb-<major.minor.patch>` so Doctrine 3 doesn't reject the lockfile, MAILER_DSN, MESSENGER_TRANSPORT_DSN, SYLIUS_STORE_URL), runs `composer install`, `doctrine:database:create`, `doctrine:migrations:migrate`, `sylius:install --no-interaction`, `sylius:fixtures:load default` (channels, taxa, products, promotions, demo customers/orders/payments — always runs because the storefront 500s with "Channel could not be found!" without it), updates `sylius_channel.hostname` to the project's nginx host (Sylius resolves channels by hostname; the default fixtures use `localhost`/wildcards that don't match `*.test`), then `yarn install` + `yarn build` for the admin/shop/app Encore bundles, plus `assets:install` and cache warmup. Auto-detection via `composer.json` / `composer.lock` declaring `sylius/sylius` or `sylius/sylius-standard`. See [docs/sylius.md](docs/sylius.md)
- Sylius presets: `--preset 2` (Latest, Sylius 2.0.x / PHP 8.3 / MariaDB 11.4 / Redis 7.4 / Node 22), `--preset 1` (Stable, Sylius 1.13.x / PHP 8.2 / MariaDB 10.11 / Redis 7.2 / Node 20). Interactive preset wizard mirrors the Medusa/Saleor/Spree flow

Sylius — post-tracer hardening:
- `--sample-data` flag now toggles the fixture suite (`default` with the flag, `minimum` without). The previous build always loaded `default`, which seeded ~87 products + sample orders even when the user just wanted a bare storefront
- Admin credentials no longer hardcoded to `sylius`/`sylius` — read from the central `magento/admin_*` config (same defaults as Magento/Shopware/PrestaShop). Install hashes the password via Symfony's `security:hash-password` (Argon2id) and updates the seeded admin row in `sylius_admin_user`. Single source of admin truth across platforms
- `madock install` is now idempotent. A `.madock-installed` marker file in the project root suppresses the first-run-only `sylius:install` + `sylius:fixtures:load` steps on subsequent runs (those commands create new rows every time without checking for existing data — re-running them duplicates the catalog). Everything else (composer, migrations, channel hostname pin, admin patch, yarn, cache warmup) still runs every time so it stays in sync with the latest config. Delete the marker to force a full re-install
- `service:enable messenger` — optional Symfony Messenger consumer container (reuses the PHP image). Auto-consumes the well-known Sylius 2 transports (`main`, `payment_request`, `catalog_promotion_removal`). Override with the `SYLIUS_MESSENGER_TRANSPORTS` env var on the service
- `service:enable encore` — optional Webpack Encore watcher container running `yarn watch` against the project src. Admin/shop/app bundles rebuild on save. `WATCHPACK_POLLING=true` keeps it responsive on macOS bind mounts
- `MAILER_DSN` now points at `smtp://host.docker.internal:1025` instead of `smtp://mailpit:1025`. Mailpit runs as a shared `aruntime-mailcatcher-1` container on the host (not on per-project networks), so the in-network hostname was unreachable from the PHP container
- PostgreSQL DSN support — install handler picks the DSN scheme + serverVersion format from `db/type` config. MariaDB stays the default (`mysql://...?serverVersion=mariadb-X.Y.Z`); PostgreSQL emits `postgresql://...?serverVersion=X.Y.Z`; plain MySQL emits `mysql://...?serverVersion=X.Y.Z` without the mariadb prefix
- Elasticsearch / OpenSearch wiring honors `search/elasticsearch/enabled` / `search/opensearch/enabled` config instead of hardcoding `false`. Project-level Dockerfile generator now materialises both engine images. Enable with `madock service:enable elasticsearch` / `opensearch`
- API Platform endpoints verified: `/api/v2/shop/products`, `/api/v2/shop/channels`, `/api/v2/shop/taxons` return 200 + JSON-LD out of the box (no manual config). `/api/v2/admin/*` returns 401 without OAuth — same as upstream

Changed:
- `php/nodejs` Dockerfile snippet now installs Yarn as well when `php/yarn/enabled=true`. PHP-based platforms with Webpack/Encore pipelines (Sylius today; Shopware/PrestaShop tomorrow if they opt in) get yarn in the same image as composer instead of needing a separate container
- `service` registry expanded with `spree/sidekiq`, `spree/storefront`, `sylius/messenger`, `sylius/encore` mappings so `madock service:enable <short>` resolves correctly

**v3.7.9**

Added:
- Spree Commerce platform support: `madock setup --platform spree` (Ruby on Rails admin + auto-provisioned Next.js storefront + PostgreSQL + Redis). `madock spree <cmd>` to run `bundle exec rails <cmd>` inside the ruby container. `madock install` writes `.env` (DATABASE_URL, REDIS_URL, RAILS_ENV, SECRET_KEY_BASE, BINDING, PORT, admin credentials), pins `.ruby-version` + Gemfile.lock RUBY VERSION line to the container's actual Ruby, patches `config/environments/development.rb` with `assume_ssl = true` for nginx TLS termination, `bundle install`, `rails db:prepare`, `spree:admin:tailwindcss:build`, `spree:search:reindex`, `spree_sample:load` (211 products, 20 customers, sample orders, publishable API key). Default admin: `admin@example.com` / `spree123`. Auto-detection via `Gemfile` / `Gemfile.lock` containing `spree`. See [docs/spree.md](docs/spree.md)
- Spree presets: `--preset 5` (Spree 5.x / Ruby 4.0 / PostgreSQL 16 / Redis 7.2 / Rails 8), `--preset 4` (Spree 4.10.x / Ruby 3.2 / PostgreSQL 15 / Redis 7.0 / Rails 7.1). Interactive preset wizard mirrors the Medusa/Saleor flow
- Spree storefront auto-provisioned. `madock setup -d` clones `spree/storefront` (official Next.js 16 / TypeScript storefront) into `./storefront/` alongside the backend. `madock install` extracts the publishable key from `Spree::ApiKey` via `rails runner`, writes `storefront/.env.local` (SPREE_API_URL=http://ruby:3000, SPREE_PUBLISHABLE_KEY, NEXT_PUBLIC_SITE_URL, country/locale/store_name), and runs `yarn install`. nginx splits the public host: `/admin|/admin_user|/api|/up|/rails|/assets|/webhooks|/oauth|/cable` to `ruby:3000`, everything else to `storefront:3001`. Lazy DNS via Docker's embedded resolver (127.0.0.11) so nginx survives early boot. Storefront defaults: `spree/storefront/enabled=true`, `path=storefront`, `workdir=/var/www/html/storefront`, `country=us`, `locale=en`, `version=22.20.0` (Node), `git_url=https://github.com/spree/storefront.git`. Override any of those in `config.xml`. Set `spree/storefront/enabled=false` to fall back to backend-only nginx config (storefront URL then 301-redirects to `/admin`)
- `service:enable sidekiq` — optional Sidekiq worker container (same ruby image as backend, runs `bundle exec sidekiq` against the project's Gemfile.lock, connects to Redis at `redisdb:6379/0`)
- Smart Ruby entrypoint in the spree ruby container: waits for `Gemfile`, then for `bundle check` to pass (gem deps fully installed for current Gemfile.lock), sources `.env` right before exec, cleans up stale `tmp/pids/server.pid`, prefers `bin/rails server` and falls back to `bundle exec rails server`. Idles with a clear message when project code or deps are missing
- Smart storefront entrypoint variant for the Spree Next.js storefront: same wait-for-install-marker pattern as the Medusa storefront entrypoint, sources both `.env` and `.env.local` before exec
- `DetectFromGemfile` — Gemfile / Gemfile.lock scanner that matches `gem "spree"` declarations and `    spree (X.Y.Z)` resolved lockfile entries. Wired into the same auto-detection chain as composer / package.json / pyproject

Fixed:
- Saleor python entrypoint sourced `.env` at the wrong point. The file is written by `madock install` AFTER the container starts, so sourcing at boot found nothing — DATABASE_URL / REDIS_URL / SECRET_KEY never landed in process env and uvicorn / runserver fell back to psycopg's localhost default, serving 502. Moved `set -a; . ./.env; set +a` to right before exec, after the wait-for-deps loop has proven the install completed. Same fix pattern as the Medusa nodejs entrypoint

**v3.7.8**

Added:
- Medusa storefront is now auto-provisioned. `madock setup -d` clones `medusajs/nextjs-starter-medusa` into `storefront/` alongside the backend; `madock install` writes `storefront/.env.local` (with `MEDUSA_BACKEND_URL=http://nodejs:9000`, `NEXT_PUBLIC_MEDUSA_BACKEND_URL=https://loc.<project>.com`, default region, publishable key) and runs `yarn install` inside the storefront container, then restarts it. nginx routes `/health|/app|/store|/admin|/auth` to the backend on `nodejs:9000` and everything else to `storefront:8000` (lazy DNS via Docker's embedded resolver so nginx survives early boot). Storefront defaults: `medusa/storefront/enabled=true`, `path=storefront`, `workdir=/var/www/html/storefront`, `region=gb`, `git_url=https://github.com/medusajs/nextjs-starter-medusa.git`. Override any of those in `config.xml`. Set `medusa/storefront/enabled=false` to fall back to the single-upstream backend-only nginx config
- Medusa publishable API key seeding. `madock install` polls `/health`, reuses the publishable key bound to the default sales channel (the one `db:setup` seeds), creates and binds one if none exist, and writes `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=…` into both the backend `.env` and `storefront/.env.local`. Eliminates the manual key-creation step that Medusa v2 otherwise requires for any `/store/*` request (would 400 with "A valid publishable key is required")
- Medusa starter seed auto-runs. When `package.json` defines a `seed` script (the default in `medusa-starter-default`), `madock install` invokes `yarn seed` after `db:setup` to populate the Europe region, sales channel, shipping options, and demo products — without it the Next.js storefront middleware errors out with "No regions found"

Fixed:
- Medusa `db:setup` left migration scripts pending. `madock install` previously called `db:migrate` and Medusa boot then hit `relation tax_provider does not exist` until a separate `db:migrate:scripts` run. Switched to `npx medusa db:setup --db <name>` (umbrella command that runs migrations, migration scripts, and link sync) and added an explicit container restart after install so PID 1 doesn't race the final migration scripts
- Medusa Admin "Blocked request" / 403 (Vite 5 host gate). `madock install` patches `medusa-config.ts` to add `admin: { vite: () => ({ server: { allowedHosts: true } }) }` if not already present, so the `*.test` project host loads the admin UI without manual config edits
- Backend chokidar reload loop triggered by storefront installs. Medusa's `develop.js` hardcodes the watcher ignore list and only ignores top-level `node_modules`. With the storefront cloned into `./storefront/`, every file written by `yarn install` in the storefront triggered a backend reload. `madock install` now patches `node_modules/@medusajs/medusa/dist/commands/develop.js` to inject regex ignores for `/storefront/` and any `/node_modules/` segment. The patch is idempotent and survives until the next backend `yarn install` (re-run `madock install` to re-apply)
- `medusa/storefront/public_backend_url` derivation. The host parser stores host strings as `domain.test:code` (where `code` namespaces nginx/hosts/<code>/name). Previously the storefront's `NEXT_PUBLIC_MEDUSA_BACKEND_URL` came out as `https://medusa.test:base` because we used the raw value. Now the trailing `:code` is stripped before building the URL
- Storefront entrypoint sources `.env` and `.env.local` right before exec so the Next.js dev server sees `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY` etc. Same wait-for-deps marker pattern as the backend entrypoint (yarn 4 install-state.gz, yarn 1 integrity, npm package-lock, pnpm modules.yaml) so the dev server starts the moment `yarn install` completes
- Backend `.env` line gluing when `medusa db:setup` rewrites the file without a trailing newline (it appends `DB_NAME=<db>`). The publishable key write now prepends `\n` so it can't end up concatenated onto the previous line

**v3.7.7**

Fixed:
- `madock setup -d -i` for Medusa and Saleor: the Node.js / Saleor python entrypoints used to `exec sleep infinity` when `node_modules` / `.venv` was missing, then `madock install` populated those folders inside the same container via `docker exec`, but PID 1 stayed asleep. The dev server never started and nginx returned 502 Bad Gateway. The entrypoint now poll-waits for deps and exec's `yarn dev` / `uvicorn` / `manage.py runserver` the moment they appear
- Medusa and Saleor `setup` controllers now honour `-d` (download) and `-i` (install) flags. Previously only Magento setup looked at them, so `madock setup -d -i -s` on a Medusa/Saleor project rebuilt containers and exited without cloning the starter or running migrations. Medusa setup clones `medusajs/medusa-starter-default`; Saleor setup clones `saleor/saleor` at the branch derived from the selected version

**v3.7.6**

Added:
- Saleor platform support: `madock setup --platform saleor` (Python 3.12 + PostgreSQL + Redis + uvicorn/runserver). `madock saleor <cmd>` to run `manage.py` inside the python container (uses `uv run` when `uv.lock` is present). `madock install` writes `.env` (SECRET_KEY, DATABASE_URL, REDIS_URL, CELERY_BROKER_URL, ALLOWED_HOSTS, PUBLIC_URL), runs `uv sync --frozen` (or `pip install` for older releases), `manage.py migrate`, and `manage.py populatedb --createsuperuser` for the default `admin@example.com` / `admin` account. Auto-detection via `pyproject.toml` / `uv.lock` / `poetry.lock` / `requirements.txt`. See [docs/saleor.md](docs/saleor.md)
- Saleor presets: `--preset latest` (Saleor 3.23 / Python 3.12 / PostgreSQL 15 / Redis 7.2), `--preset stable` (Saleor 3.20). Interactive preset wizard mirrors the Medusa flow
- `service:enable dashboard` — optional Saleor Dashboard SPA container (`ghcr.io/saleor/saleor-dashboard:3.23`), host port auto-allocated via `{{{port/saleor_dashboard}}}`, `API_URL` wired to the project nginx host
- `service:enable worker` — optional Celery worker (with beat embedded) sharing the python image, runs `celery -A saleor --app=saleor.celeryconf:app worker --loglevel=info -B`
- Smart Python entrypoint in the saleor python container: sources `.env` (Saleor reads config from `os.environ`, does NOT auto-load `.env`), detects `manage.py` + `saleor.asgi:application` and prefers `uvicorn` for ASGI, falls back to `manage.py runserver`. Idles with a clear message when dependencies are missing
- `ProxyConfTransformer` extension point (`src/helper/configs/aruntime/proxytransform/`) — lets downstream consumers post-process the fully assembled `proxy.conf` before it lands on disk. Symmetric with the existing `DockerTransformer` hook for `docker-compose.yml`. Use case: enterprise add-ons rewriting service location prefixes (e.g. suffixing `/phpmyadmin/` with a per-project hash), adding extra server blocks for cross-domain admin tools, injecting `auth_request` directives, etc. API: `proxytransform.SetProxyConfTransformer(t ProxyConfTransformer)` where `ProxyConfTransformer.TransformProxyConf(content string) string`. Default behaviour unchanged when no transformer is registered

Changed:
- `docker.Down` / `docker.Kill`: label-based fallback (`com.docker.compose.project=madock_<name>`) so containers/volumes/networks/images get cleaned even when the compose file is missing. Previously these were silent no-ops when the project state directory had already been removed
- `GetProjectName`: resolve symlinks (`filepath.EvalSymlinks`) before comparing the stored project `path` against the current working directory. On macOS `/tmp` is a symlink to `/private/tmp`, so revisiting a project through the symlinked path no longer auto-suffixes the name to `<project>-2`. Also tightens the suffix loop to actually exit on a match

**v3.7.5**

Added:
- Medusa.js platform support: `madock setup --platform medusa` (Node.js + PostgreSQL + Redis), `madock medusa <cmd>` to run the Medusa CLI inside the nodejs container, `madock install` scaffolds `.env` + runs `db:migrate` + creates an admin user. Auto-detection via `package.json` (`@medusajs/medusa` or `@medusajs/framework`). Default versions: Node 20.18, PostgreSQL 16.4, Redis 7.2, Yarn 4.5. See [docs/medusa.md](docs/medusa.md)
- Medusa setup presets: `--preset latest` (Medusa 2.x: Node 22, Postgres 17, Redis 7.4), `--preset stable` (Medusa 2.0 baseline), `--preset legacy` (Medusa 1.x: Node 18, Postgres 14, Redis 7.0). Interactive preset wizard in `madock setup --platform medusa` mirrors the Magento flow
- `service:enable meilisearch` — Meilisearch as an opt-in search engine container across all platforms (`getmeili/meilisearch:v1.11.3`, master key `masterKey`). Wired into the Medusa compose template
- `service:enable storefront` — optional Next.js storefront container for Medusa v2. Mounts the project's `<project>/storefront/` folder into `/var/www/storefront`, internal port 8000, host port auto-allocated via `{{{port/storefront}}}`. Env vars wire `MEDUSA_BACKEND_URL` to the internal `nodejs:9000`. Configurable via `medusa/storefront/*` keys in `config.xml`

Changed:
- Port allocator now also probes the host (`net.Listen`) and consults `docker inspect HostConfig.PortBindings` for every running and stopped container before handing out a port. The registry remains the primary source of truth; the extra probes defend against ports that something outside madock claimed (other docker stacks, leftover containers, non-docker listeners)
- Default DB credentials changed from `magento`/`magento`/`magento` to DDEV-style `db`/`db`/`db` (`db/root_password` stays `password`). Affects new projects only; existing projects keep their stored values
- New V375 migration backfills `db/user`/`db/password`/`db/database` = `magento` for projects whose `config.xml` relied on the previous embedded defaults, so their Docker volumes and DB users keep working after the upgrade
- Default `timezone` switched from deprecated `Europe/Kiev` to `UTC`. IANA renamed `Europe/Kiev` to `Europe/Kyiv` in tzdata 2022b; UTC is the standard server default and avoids DST surprises in logs. Existing projects keep their stored timezone
- Shared nginx `proxy.conf` no longer hardcodes upstream port `3000`. It now uses `{{{main_service_port}}}`, resolved per platform from the project config (Medusa env writer sets it to `9000`; existing custom/nodejs projects fall back to `3000`, matching the old behaviour)

**v3.7.4**

Added:
- Magento 2.4.9 support: PHP 8.5 + Xdebug 3.5.0, MariaDB 11.8, RabbitMQ 4.2, Valkey 9.0.0. OpenSearch 3.0.0 was already wired. Composer stays on the `"2"` major (ondrej apt resolves the latest 2.9.x). Project and proxy nginx bumped to 1.28
- ActiveMQ Artemis 2 as an opt-in service. Enable with `madock service:enable artemis` — wired on all platforms (magento2, shopware, prestashop, woocommerce, shopify, custom). Defaults: `apache/activemq-artemis:2.42.0`, user/password `artemis/artemis`. Not part of the `setup` wizard
- `service:enable --version <ver>` flag. For services that have a version (currently `valkey`, `artemis`, `xdebug`), enable prompts an interactive version picker (same selector as `setup`) unless `--version` is given, then persists `<service>/version` to the project config

Changed:
- `setup` wizard no longer prompts for Valkey version. The Valkey container stays opt-in via `service:enable valkey [--version <ver>]`, matching the new pattern. Existing `<valkey>` config blocks remain unchanged
- `project:clone` now requires `--domain-suffix` / `-s`. The suffix is inserted before the TLD dot of each cloned host (e.g. `shop.test` + `-update` → `shop-update.test`), so the proxy nginx no longer aborts the clone with "Duplicate domains found" right after copying the source config
- PHP 8.5 build support: `php8.5-opcache` and `php8.5-xmlrpc` (not shipped as separate packages by ondrej PPA) are now installed in their own optional `apt-get` lines that tolerate a missing package. The pecl mcrypt skip branch in the php Dockerfile snippets now covers PHP 8.4 and any newer version (`>= 8.4`) instead of being hardcoded to 8.4 only
- `setup --preset` list: new `Magento 2.4.9 (Latest)` preset (PHP 8.5, OpenSearch 3.0, MariaDB 11.8, RabbitMQ 4.2, Valkey 9.0.0). The previous "Latest" entry for 2.4.8 is relabelled to `(Previous)`
- `patch:create` now detects `cweagans/composer-patches` major version from `composer.lock` and writes the matching format: v1 keeps the existing `"vendor/pkg": { "Title": "path" }` map, v2 writes the new `"vendor/pkg": [ { "description": "Title", "url": "path" } ]` array-of-objects shape. Applies to `extra.patches` in `composer.json` and to `patches.json`

Fixed:
- Fix `host not found in upstream "php_without_xdebug:9000"` nginx error caused by the `<<<if{{{main_service_enabled}}}>>>` block in `nginx.yml` always being stripped — `main_service` and `main_service_enabled` placeholders are now substituted before `ReplaceConfigValue` runs `processConditionals`, so the conditional sees the concrete value (`true`/`false`) instead of an unresolved placeholder. Without this fix the `depends_on: php` block in nginx was always removed, letting nginx start before `php_without_xdebug` and fail upstream DNS resolution. Affects all projects on 3.7.2/3.7.3, regardless of `php/enabled` value ([#40](https://github.com/faradey/madock/issues/40))
- Unlock the ImageMagick PDF coder in php Dockerfile snippets — default Debian/Ubuntu `/etc/ImageMagick-6/policy.xml` blocks PDF reads, which breaks Imagick-based PDF preview generation in PHP apps (e.g. Magento label rendering). The `rights="none" pattern="PDF"` policy is now switched to `rights="read|write"` during image build

**v3.7.3**

Fixed:
- Fix `host not found in upstream "php_without_xdebug:9000"` nginx error after upgrading to 3.7.2 with `php/enabled=false` and `php/xdebug/enabled=true` — nginx confs in all platform templates now gate the `fastcgi_backend_xdebug_true` upstream on the same dual condition (`php/enabled` AND `php/xdebug/enabled`) used by the `php_without_xdebug` compose snippet ([#40](https://github.com/faradey/madock/issues/40))

**v3.7.2**

Fixed:
- Fix `service "nginx" depends on undefined service "php"` and `service "php_without_xdebug" depends on undefined service "php"` errors when project config lacks `php/enabled` — nginx `depends_on` is now gated by a new `main_service_enabled` placeholder, and the `php_without_xdebug` snippet now requires both `php/enabled` and `php/xdebug/enabled` ([#40](https://github.com/faradey/madock/issues/40))
- Use full `php bin/magento cache:flush` instead of the `c:f` shorthand inside `madock c:f` to avoid Symfony console ambiguity in setup-only mode

Changed:
- V366 migration now also covers `woocommerce` and `shopify` platforms
- New V372 migration backfills `php/enabled=true` for projects upgraded from versions in the 3.6.7..3.7.1 range, which the V366 trigger (`< 3.6.7`) had missed

**v3.7.1**

Fixed:
- Fix nodejs language Dockerfile build failure: `chown 501:20 /var/www` failed because `node` base image has no `/var/www` directory — `mkdir -p /var/www` added before chown
- Suppress noisy `cron: unrecognized service` stderr from cron stop probe — `service cron status` now runs silently when used as availability probe

Added:
- Cron support in nodejs language container: `apt-get install -y cron` added to nodejs Dockerfile, enabling `cron.enabled=true` and `cron/jobs/*` for nodejs-only projects

**v3.7.0**

Added:
- `madock mcp` — built-in MCP (Model Context Protocol) server for AI assistants (Claude Code, Cursor, VS Code). Provides 30 tools: container lifecycle, configuration, database operations, service management, Composer/Magento CLI, remote sync, and more. See [docs/mcp.md](docs/mcp.md)
- WooCommerce platform support: `madock setup --platform woocommerce`, WP-CLI via `madock wp`, auto-detection by `wp-config.php`
- JetBrains IDE plugin: [Madock Integration](https://plugins.jetbrains.com/plugin/31208-madock-integration)

**v3.6.9**

Added:
- `--quiet` / `-q` flag available on all commands — suppresses Docker build/pull output (useful in JediTerm and other IDEs to avoid flood output). Affects `start`, `rebuild`, `setup`, `debug:enable`, `debug:disable` and any other command that triggers Docker operations
- `db:import` now detects MySQL `GTID_PURGED cannot be changed` errors and offers an interactive resolution: run `RESET MASTER` (or `RESET BINARY LOGS AND GTIDS` on MySQL 8.4+) and retry, or retry with GTID statements stripped from the dump on the fly
- `--reset-gtid` flag for `db:import` to perform the GTID reset automatically before import (useful for CI/scripts)
- `db:import` now detects MySQL `ERROR 1062 Duplicate entry` errors and offers to retry the import in force mode (`-f`) so duplicate-row errors are skipped instead of aborting

Fixed:
- `db:import` now restores `FOREIGN_KEY_CHECKS=1` even when the import fails
- `db:import` stderr is now captured for analysis while still being streamed to the terminal

**v3.6.8**

Changed:
- Rename `DockerSecretsInjector` interface to `DockerTransformer` — more general name for the Docker file transformation hook, backward-compatible `SetSecretsInjector` wrapper retained

**v3.6.5**

Fixed:
- Fix search engine config not applied when using presets — setup controllers now pass search engine type to project config generators
- Remove trailing slash from root path variables in nginx configs
- Use host-gateway instead of outbound IP for container host resolution — removes unreliable `GetOutboundIP()` UDP dial, uses Docker's built-in `host-gateway`
- Bump version.go to 3.6.5 (was stuck at 3.6.1 since v3.6.2–v3.6.4)

**v3.6.1**

Fixed:
- Fix `MADOCK_USER` environment variable not working with `madock bash` — the bash controller now respects env overrides via `GetEnvForUserServiceWorkdir`

**v3.6.0**

Changed:
- Move database credentials from Dockerfile ENV to docker-compose environment — passwords are no longer baked into Docker image layers (visible via `docker history`), instead passed at runtime through docker-compose environment variables. Affects mysql, postgresql, mongodb, and db2 services.

**v3.5.9**

Fixed:
- Fix db/type migration not running for users upgrading from v3.4.0+ — migration was guarded by `< "3.4.0"` and version.go was not bumped, so the migration never executed for existing users
- Bump version.go to 3.5.9

**v3.5.8**

Added:
- v3.4.0 migration: adds `db/type` field to existing project configs based on `db/repository` (mysql, postgresql, mongodb)
- Sync `config_defaults.xml` with `config.xml`: add `db/type`, `db/pgadmin`, `db/mongo_express` defaults

**v3.5.7**

Fixed:
- Fix remaining `.madock/config.xml` write paths — `SetEnvForProject` (setup) and `GetCurrentProjectConfigPath` (scope:set/add) now correctly write to `projects/<projectname>/config.xml`

**v3.5.6**

Added:
- Mailpit (mailcatcher) is now a toggleable service — disabled via `madock service:disable mailpit --global`, enabled by default for backward compatibility

Changed:
- `.madock/config.xml` is now read-only for madock — all automatic config changes (`service:enable/disable`, `config:set`, `debug:enable/disable`, `cron:enable/disable`) write to `projects/<projectname>/config.xml` instead. This allows `.madock/config.xml` to be committed to the repository without unexpected modifications on servers

**v3.5.5**

Fixed:
- Fix nodejs version in PHP container ignoring project config — `customPhpConfig` used `generalConf` directly instead of `GetOption`, always defaulting to 18.x regardless of project settings

**v3.5.4**

Added:
- `llms.txt` — structured context file for AI agents (Claude Code, Cursor, Copilot) with full command reference, config format, and architecture overview

**v3.5.3**

Fixed:
- Fix XML config parser losing data when adding keys to empty scope — `<default></default>` (empty element) blocked `SetParam` from writing nested keys. Now empty leaf nodes are promoted to branch nodes when nested keys are added.

**v3.5.2**

Added:
- Embed `docker/` and `scripts/` into the binary via `go:embed` — the binary is now self-contained
- Auto-extract embedded assets to disk on first run or version change (`.embedded_version` marker)
- `src/helper/embedded` package with `ExtractIfNeeded()` for version-aware asset extraction

**v3.5.1**

Added:
- Per-option confirmation in setup reconfigure mode: when re-running `madock setup` on a project with existing config, each option shows "Current: X — Change? [y/N]" instead of re-asking everything from scratch
- `PopulateFromConfig` helper to fill ToolsVersions from existing project config
- `SetReconfigure` flag to enable/disable reconfigure mode in setup tools
- `Language()` now accepts current value parameter for correct display in reconfigure mode

Changed:
- `SelectInteractive` shows "Change?" confirmation in reconfigure mode, skipping selector if user declines
- `hostsCustom` in custom platform converted to use `SelectInteractive` for consistent reconfigure behavior
- All platform setup handlers (Magento, Custom, Shopware, Shopify, PrestaShop) call `PopulateFromConfig` before interactive questions

**v3.4.0**

Added:
- PostgreSQL support: docker-compose snippet, Dockerfile, `db:export` via `pg_dump`, `db:import` via `psql`, `db:info` with type display
- MongoDB support: docker-compose snippet, Dockerfile, `db:export` via `mongodump`, `db:import` via `mongorestore`
- Database engine selector in `madock setup`: MariaDB, MySQL, PostgreSQL, MongoDB
- `db/type` config key for explicit database type (`mysql`, `postgresql`, `mongodb`) with auto-detection from `db/repository` for backward compatibility
- Template flags `db/type_is_mysql`, `db/type_is_postgresql`, `db/type_is_mongodb` for conditional docker-compose/Dockerfile sections
- pgAdmin service (`db/pgadmin`) for PostgreSQL admin UI
- Mongo Express service (`db/mongo_express`) for MongoDB admin UI
- `remote:sync:db` support for PostgreSQL (`pg_dump`) and MongoDB (`mongodump`)
- `DbType` field in `ToolsVersions` struct
- Version selectors: MySQL (9.2, 9.1, 8.4, 8.0), PostgreSQL (17, 16, 15, 14, 13), MongoDB (8.0, 7.0, 6.0, 5.0)

Changed:
- `db:export`, `db:import`, `db:info` commands now dispatch by database type from config
- All platform env writers set `db/type` and `db/repository` based on selected engine
- `MakeDBDockerfile` skips `my.cnf` generation for non-MySQL databases
- `db.yml` docker-compose snippet wrapped in `<<<if{{{db/type_is_mysql}}}>>>` conditional

**v3.3.0**

Added:
- Exported `GetDefaultConfigXML()` in `configs` package — returns raw embedded config defaults for enterprise config layering
- Exported `version.Version` constant in `src/version/` package so downstream consumers can read the madock version without hardcoding it
- Tests for `GetOriginalGeneralConfig()` merge behavior (embedded-only, file-over-embedded, empty-value gap-fill)

Changed:
- `main.go` uses `version.Version` instead of local `appVersion` var
- `<<<else>>>` support in template engine for conditional blocks (`<<<if>>>...<<<else>>>...<<<endif>>>`)
- Centralized service credentials in `config.xml` for RabbitMQ, Grafana, Redis, Valkey, Elasticsearch, OpenSearch
- Auth config blocks (`auth/enabled`, `auth/user`, `auth/password`) for Grafana, Redis, Valkey, Elasticsearch, OpenSearch
- Secret key registration for all new service passwords
- RabbitMQ docker snippet now uses `{{{rabbitmq/user}}}` and `{{{rabbitmq/password}}}` placeholders instead of hardcoded `guest:guest`
- Grafana docker snippet uses `<<<if>>><<<else>>>` conditional for anonymous vs credential-based auth
- Grafana RabbitMQ exporter uses config placeholders for RabbitMQ credentials
- MySQL exporter config uses `{{{db/root_password}}}` placeholder instead of hardcoded password
- Migration guide for PWA Studio projects to custom+nodejs platform
- Snippet-based Dockerfiles for all languages (Python, Go, Ruby, Node.js, None) using reusable common snippets
- Common Docker snippets: `header-ubuntu`, `cron`, `mkdir`, `chown`, `cleanup`, `footer`
- `php/enabled` conditional guard for PHP services in docker-compose
- Dynamic `depends_on` in nginx with `{{{main_service}}}` placeholder
- Interactive version selectors for Python, Go, Ruby during `madock setup`
- Nginx snippet system (`php.conf`, `proxy.conf`) for language-specific configurations
- Migration v3.3.0 for automatic config key migration
- Moved PHP Dockerfile from `docker/custom/php/` to `docker/languages/php/`
- Renamed config key `php/timezone` → `timezone` across all platforms
- Split `nodejs/enabled` into standalone (language) and `php/nodejs/enabled` (embedded in PHP container)
- All languages now use unified fallback chain through `docker/languages/<language>/`
- Moved `<timezone>` from `<php>` to top level in default config.xml

Removed:
- PWA as a standalone platform (use custom+nodejs instead)

**v3.2.0**

Added:
- Multi-language support for custom platform: PHP, Node.js, Python, Go, Ruby, and language-less (`none`) projects
- Configurable cron jobs support for all platforms
- `info:ports` command to show allocated ports for the project
- File path argument support for `db:import` command
- JSON output support for CLI commands (`--output=json`)
- MySQL 8.4+ support, removed deprecated `db/type` config
- VPS installation script
- `xdg-utils` to base PHP image for all platforms
- Hot reload ports for Shopware storefront
- Port mappings for proxy services (Grafana, Kibana, OpenSearch Dashboards, phpMyAdmin, Selenium, Varnish)
- RabbitMQ monitoring dashboard to Grafana
- Deployment guide for Magento 2 and Shopware
- Documentation for Magento, PrestaShop, Shopware, custom cron jobs, JSON output

Refactored:
- Command registry pattern replacing switch statement in `main.go`
- Platform handler interface to eliminate code duplication
- Split `docker.go` into focused modules
- `SetXmlMap` refactored to use recursion instead of hardcoded switch cases
- Path builder utility to centralize path construction
- Replaced `panic(err)` with `log.Fatal` for proper error handling
- Reuse `removeCronJobsFromConfig` in install function

Fixed:
- `patch:create` command to work without TTY
- `cron:enable/disable` not saving config status
- Duplicate project entries in domain check
- Varnish proxy configuration
- Nginx proxy configuration issues
- Network configuration for dashboard services
- Grafana configuration for dashboards and networking

Improved:
- Instant key response for setup confirmation prompt
- Increased proxy rate limit defaults
- Verbose CLI output for Shopify cron enable/disable
- Auto-detect artisan location for Shopify cron setup
- Duplicate domain error messages now show all affected projects

**v3.1.0**

Added:
- Improved documentation for media sync, cron, snapshots, isolation, environment variables, and configuration
- Interactive setup wizard with ASCII banner, progress indicators, arrow keys navigation, styled selectors, configuration summary, inline validation, help hints, and confirmation prompts
- `proxy:reload` command for graceful nginx configuration reload without downtime
- `--yes` flag to setup command for auto-confirmation (skip prompts in CI/CD)
- `--preset` flag for quick setup with preset configurations (e.g., `magento-248`, `magento-247`)
- Auto-detection of Magento version from composer.json
- Progress indicator for database import
- On-demand port allocation system for better resource management
- Configurable proxy settings
- Timestamp to debug.log entries
- Magento 2.4.9 support with OpenSearch 3.0.0
- `shopware:bin` command for Shopware CLI operations
- Unit tests for core packages
- RabbitMQ monitoring dashboard in Grafana (queues, connections, channels, message rates)
- RabbitMQ exporter for Prometheus metrics collection
- Port mappings for Grafana, Kibana, OpenSearch Dashboards, phpMyAdmin, Selenium, Varnish

Improved:
- Nginx proxy security and performance
- Updated nginx from 1.21.4 to 1.26
- Grafana stack configuration with proper datasource UIDs for dashboard compatibility

Fixed:
- Section padding panic in setup wizard
- Non-deterministic XML config output order
- Nested conditional processing in config templates
- MariaDB exec file compatibility
- Composer install command for Shopify platform
- Various potential bugs across the codebase
- Nginx http2 directive deprecation warning (nginx 1.25+)
- Duplicate upstream and global directive errors in nginx proxy
- Varnish network connectivity with backend nginx
- Grafana subpath proxy configuration

**v3.0.0**
- Introduced a generic diff command: `madock diff --platform <code> --old <ver> --new <ver> [--path <publicDirFromSiteRoot>]`
- Added store scopes documentation split into a dedicated file `docs/store_scopes.md` and linked from README
- Added Valkey key-value DB
- Minor fixes and refactors in diff scripts (path handling and directory creation)

**v2.9.1**
- Added Magento 2.4.8 support
- Fixed the restart policy for aruntime containers

**v2.9.0**
- Added the env variable MADOCK_TTY_ENABLED (0/1). MADOCK_TTY_ENABLED is enabled by default
- Fixed SSH volume
- Fixed "install" command for prestashop platform
- Fixed docs
- Added logo
- Fixed GetRunDirPath function for outside executors
- Added php8.4 support
- Fixed incorrect version comparison for MariaDB
- Fixed arguments for the Setup command
- Fixed Magento2 install subcommands
- Fixed livereload
- Fixed apt-get to apt and added --allow-releaseinfo-change
- Added php-redis library to php installation
- Fixed RabbitMQ recommended version for Magento 2.4.7-p5 and later
- Added the restart policy

**v2.8.0**
- Added **PrestaShop** as a separate service
- Fixed "composer" command for Shopify service
- Improved custom commands and documentation

**v2.7.0**
- Fixed the creation of patches
- Fixed the cron for Shopify platform
- Fixed TODO comments
- Fixed NodeJs major version for php.Docker file
- Added http2 in the nginx configuration

**v2.6.0**
- Added Grafana as a service
- Added Grafana dashboards for Loki, Mysql and Redis
- Support for snippets in configuration files has been added. This has allowed us to eliminate repetitive code and settings.
- Added the new option `--shell` for `madock bash` command. It can be used `bash` or `sh` as a shell.

**v2.5.0**
- Added supporting of Shopware
- Fixed mailcatcher configuration with MP_SMTP_AUTH_ACCEPT_ANY and MP_SMTP_AUTH_ALLOW_INSECURE
- Fixed documentation
- Fixed the media synchronization public path
- Added --db-host, --db-port, --db-name, --db-user, --db-password as options for the remote:sync:db command

**v2.4.4**
- Fixed opensearch-dashboards
- Added new command `madock project:clone` [more](docs/project_clone.md)
- Added php/nodejs service to the php container
- Fixed documentation
- Fixed bug with the `madock cli` command
- Added custom commands [more](docs/custom_commands.md)

**v2.4.3**
- Added interactive options for the `madock setup` command
- Added an isolation mode [more](docs/isolation.md)
- Added Varnish cache [more](docs/varnish.md)
- Refactoring code


**v2.4.2**
- Support Magento 2.4.7 and Adobe Commerce 2.4.7
- Updated docker-compose version to 3.8
- Fixed DB host for the service db2
- Fixed GetActiveProjects method
- Fixed start/stop project
- Fixed db:export
- Fixed node grunt exec:<theme>
- Fixed documentation
- Added "RUN npm install -g grunt-cli" to docker file
- Fixed bug with "cache" folder
- Fixed if/else in config files
- Fixed project configuration
- Fixed Snapshot container
- Added snapshots functionality for the project
- Fixed .madock/config.xml
- Update PHP mcrypt version
- Fixed OpenSearch env variables



**v2.4.1**
- Added command scope:add to add a new scope and activate it
- Added the ability to store the madock configuration within a project in the .madock folder. To do this, you need to manually create a .madock folder and transfer configuration files and database backups to it, if necessary
- Added full support for creating patches for cweagans/composer-patches
- Added full support for creating patches for vaimo/composer-patches
- Added logger with stack trace
- Fixed the config cache
- Fixed the bug with the enable/disable services
- Fixed compatible version magerun n98 and PHP
- Fixed Adobe Cloud commands
- Fixed project path
- Fixed db:import
- Fixed bug with config.xml and the setup of a new project
- Fixed missing dir aruntime/projects
- Fixed working commands Start, Stop, Restart without internet
- Fixed madock info
- Fixed xdebug profile for PHP 7.1 or less


**v2.4.0**
- Added the new option PUBLIC_DIR in the project configuration. Each platform can have a different path of public folder therefore this option will be specified as a public folder in the container.
- Fixed host for phpmyadmin2
- Fixed mcrypt extension for PHP
- Fixed mail for CLI
- Improve command "madock c:f"
- Added --force option for the command "madock rebuild". Removes running containers without waiting for them to complete correctly and creates new containers.
- Added new library for CLI commands
- Replaced Mailhog to Mailpit
- The configuration file format would be changed from .txt to .xml. The project configuration file env.txt has been renamed to config.xml. The old configuration files have been preserved so that if you have problems with the new version of Madock, you can roll back to the old version.
- Configuration scopes for the project have been added. Now switching between configurations has become convenient and there is no need to create a copy of the project in a neighboring folder. The database is also separate for each scope.
- Added the new command "madock scope:list" for listing all scopes of the project.
- Added the new command "madock scope:set" for switching between scopes of the project.
- The commands "remote:sync:media", "remote:sync:db" and "remote:sync:file" have received an additional option "--ssh-type" which specifies the prefix of the name of the ssh settings in the project configuration. This way you can specify which ssh settings to use when executing the command.
- Added aruntime configuration caching. Now Madock will parse files less when starting and rebuilding a project.
- Added the new command "madock config:cache:clean" for cleaning Madock aruntime cache.
- Added the new command "madock open" for opening the project in the browser.
- Improve documentation of Madock

**2.2.0**
- Shopify support
- Custom PHP project support
- Relocated setup option "Specify Magento version" to top
- Added CONTAINER_NAME_PREFIX option in config. This option will allow you to run a madock project independently of other docker builds in the space with the default madock_ prefix. For already configured projects, the space will have an empty prefix to prevent projects from breaking.
- Added --ignore-table for "db:export" and "remote:sync:db" commands. Ignore the table when exporting. The specified table will not be included in the backup file. To specify multiple tables, specify this option multiple times.
- Updated OS Ubuntu for containers from 20.04 to 22.04. This will only affect those projects that will be installed after updating this build.
- Improve documentation for new commands
- Fixed some problems with NodeJs
- Fixed issue #9

Thanks @artmouse @serhii-chernenko

**2.1.0**
- Support the Magento Functional Testing Framework (MFTF)
- Fixed multiline commands

**2.0.1**
- Fixed the setup with Hosts
- Fixed the setup with the version Redis and rabbitMQ
- Fixed "madock status" command
- Fixed the DB host description

**2.0.0**
- PWA Studio as a separate service.
- Backward incompatible changes were made to the code. Code changes allow new platforms to be added in the future.
- At the moment, PWA Studio has been added as a separate service.
- There are plans to add Shopify and Shopware in the future.

**1.9.1**
- Fixed command project:remove
- Removed "restart: on-failure:3" from Elasticsearch service of docker-compose
- Installed libssh2-1-dev libssh2-1 php-ssh2 for PHP
- Removed the restart_if_failure option for the DB service of docker-compose
- Improved removing project. Now deletion is more transparent. Before execution, you will see the items that will be deleted and only after your confirmation will they be deleted.
- Fixed files permission with --with-chmod

**1.9.0**
- Added
  - Support Magento 2.4.6
  - Support sample data with the setup command
  - OpenSearch
  - Support PHP 8.2 and xdebug
  - Improved patcher for creating patches from the whole folder
  - Updated phpmyadmin version from 5.2.0 to 5.2.1
  - Increased UPLOAD_LIMIT for phpmyadmin. Now it is 2GB
  - Custom DB repository in the config
  - PHP 8.2 to the setup process
  - Xdebug profile
  - Increased PHP Max Input Vars Limit by default
  - Enabled log_bin_trust_function_creators for DB
  - New option for DB commands "--service-name DB container name. Optional. Default container: db. Example: db2"
  - Support overriding /docker/nginx/conf/default-proxy.conf
  - Command "install"
  - Support n98-magerun
  - Support the second DB
  - Support proxy as a service

- Fixed
  - Default_server for the nginx proxy configuration
  - Remove --single-transaction option from the mysqldump command
  - Remove the innodb_log_file_size option for MySQL 8.x
  - Improved elasticsearch plugins installation
  - Cron
  - Bug with the start/stop command of the proxy server
  - FOREIGN_KEY_CHECKS for the import DB
  - Project setup with Redis and rabbitMQ versions
  - Bug with the media synchronization
  - Proxy port and the starting script
  - Livereload location in nginx proxy
  - DEFINER for the DB import/export
  - Issue with permissions of .ssh folder #8

**1.8.2**
- Fixed generation env.php file with rabbitmq password

**1.8.1**
- Fixed starting the Nginx proxy containers

**1.8.0**
- Added a new command "patch:create"
- Added a new param "--name" for "db:export" and "remote:sync:db" 
- Added a new command setup:env for generating env.php file
- Changed domain .loc to .test by default
- Optimization for MariaDB 10.4
- Prune the volumes with option --with-volumes. For example Madock prune --with-volumes
- Added the ability to specify a custom repository and version of docker images when you set up the project
- Added "--with-chown" option for some commands. Reset permissions for files and folders
- Improved "db:import" command. Now, the Madock can read DB files from any folder of the Magento project. The name of the DB file must contain ".sql" in any part of the name
- Fixed the problem with the same project folder names from different locations
- Added a new command project:remove
- Added stopping proxy containers if there are no active projects
- Refactoring code

**1.7.4**
- Additional changing external IPs for containers from 0.0.0.0 to 127.0.0.1

**1.7.3**
- Changed external IPs for containers from 0.0.0.0 to 127.0.0.1
- Fixed bug with CLI options and arguments

**1.7.2**
- Fixed bug with the docker compose

**1.7.1**
- The internal command "docker-compose" was replaced by "docker compose"

**1.7.0**
- All commands are brought to uniformity. Now they match the Magento approach
- Added the support of Magento cloud
- Added the support of automatically creating composer patches
- Added the new command "cli"
- Fixed some bugs
- Some code improvements

**1.6.0**
- Added the LiveReload plugin and NodeJs  
- Added automatic start of containers after project setup 
- Added the ability to download a specific file from a remote server (for example: madock remote sync file --path app/etc/config.php)    
- Now changed project configuration is applied only after setup or rebuild commands   
- Fixed some bugs and added some improvements 

**1.5.0**
- Added new options for the setup command:    
  - --download - Download the specific Magento version from Composer to the container
  - --install - Install Magento, Shopware, etc. from the source code
- Added new command madock db info. This command prints data for connecting to the database. The output contains a port (permanent) for connecting such database programs as HeidiSQL, MySQL Workbench, and others
- Support Windows OS

**1.4.0**
- Added
  - Kibana  
  - CHANGELOG.md    
  - MADOCK_VERSION in global config.txt 
  - new functionality with services. For example: madock service phpmyadmin on  
- Fixed   
  - text of warning with DB import selecting

**1.3.0**
- For media, js, css requests it was added a new container without Xdebug. This improvement decreases load when you debug your code

**v1.2.0**
- Added a new command for displaying the status of the project   
  - madock status

**v1.1.0**
- Added support for PHP 8.1
- Added support for SSL certificates. Now you can use HTTPS in local development

**v1.0.3**
- Fixed remote sync DB

**v1.0.2**
- Added  
  - Additional logging for sync
  - Validation of project folder name  
- Fixed  
  - Mapping for the general config  
  - Remove compression for an image in png format   
  - Improve sync media files    

**v1.0.1**
- Remove the unison container for macOS

**v1.0.0**
- change docs