**v3.9.3**

Added:
- **`db/memory`, one budget that sizes whichever database engine the project runs.** The numbers used to be written into `my.cnf` by hand — `innodb_buffer_pool_size = 512M` and `innodb_log_buffer_size = 256M`, 768 MB reserved by every MySQL before it holds a single row — and there was no setting for them at all. The only way to change one was to copy the shipped `my.cnf` into a project's `.madock/docker/db/`, which is a frozen copy that then drifts from the file it was taken from, silently and for as long as nobody compares them. Measured on a demo server: two `mysqld` holding 732 and 409 MB of 3819, with the container limits unable to reach them
- **The engines are not the same problem, which is why one number is divided rather than passed through.** MySQL takes two thirds as its buffer pool and one third as its log buffer, so the default of `768M` renders exactly the two lines that were there before — the golden `my.cnf` does not move. PostgreSQL had never been given anything and sat on its stock 128MB `shared_buffers`; it now takes a quarter of the budget there and three quarters as `effective_cache_size`, which buys no memory but tells the planner what to expect. MongoDB is the one that had to be told: left alone, WiredTiger sizes its cache from the RAM it can see, which is the **host's** and not the container's limit, so one `mongod` on a large machine quietly reserves gigabytes. It takes half the budget, with MongoDB's own 0.25 GB floor
- Templates can do the arithmetic themselves — `memShare` for a fraction of the budget in a given unit, `memShareGB` for MongoDB's bare decimal. Fractions rather than percentages so the numbers land where a person would have written them: two thirds of 768M is exactly 512M, where 67% is 514M and every generated file would differ from the one before it for no reason
- A golden fixture for MongoDB, which had none. Rendering `--wiredTigerCacheSizeGB` correctly is exactly the kind of thing that is wrong once and then wrong forever
- **`--db-type`, so a scripted setup can choose an engine.** `--db` names a version — or a repository and a version, when it carries a colon — and the engine was asked for interactively, which under `--yes` meant every scripted project got MariaDB. It also made an end-to-end test of PostgreSQL or MongoDB impossible: changing `db/type` afterwards leaves the first engine's data directory in the volume and the second refuses to start on it. Three e2e tests come with it, one per engine, each asking the running server what it actually got
- The engine flag works alongside an explicit version, which took two fixes of its own. The platform setups asked for the engine only when no version was given, on the reasoning that `--db` answers the whole question — it does not, a version says nothing about an engine. And a version carried over from the general configuration belongs to the engine that was there before: asking for MongoDB produced `FROM mongo:10.6`, the MariaDB default and a tag that does not exist, so a default from another engine's list is now dropped rather than reused

Fixed:
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