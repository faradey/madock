package clone

import (
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/create"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/restore"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/credentials"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/snapshotengine"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"project:clone"},
		Handler:  Execute,
		Help:     "Clone project. Supports --json (-j)",
		Category: "project",
		ArgsType: new(arg_struct.ControllerGeneralProjectClone),
		// Global: creates a project, so it cannot require one.
		Global:     true,
		JSONOutput: true,
	})
}

// ClonedJSON is the --json answer of project:clone.
type ClonedJSON struct {
	Name  string   `json:"name"`
	Path  string   `json:"path"`
	Hosts []string `json:"hosts"`
}

// refuse ends a clone that cannot start. Each of these used to print the
// error and return, which exited 0: to a script, a clone refused for a name
// that already exists looked like a clone made.
func refuse(w *output.Writer, msg string) {
	if w != nil {
		w.Fail("%s", msg)
	}
	fmtc.ErrorLn(msg)
	os.Exit(1)
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralProjectClone)).(*arg_struct.ControllerGeneralProjectClone)
	var w *output.Writer
	if args.Json {
		w = output.Begin()
	}

	cloneName := args.Name
	if strings.Contains(cloneName, ".") || strings.Contains(cloneName, " ") {
		refuse(w, "The project folder name cannot contain a period or space")
	}
	domainSuffix := strings.TrimSpace(args.DomainSuffix)
	if domainSuffix == "" {
		refuse(w, "--domain-suffix is required and cannot be empty")
	}
	if strings.Contains(domainSuffix, ".") || strings.Contains(domainSuffix, " ") {
		refuse(w, "--domain-suffix cannot contain a period or space")
	}
	projectsPath := paths.GetExecDirPath() + "/projects"
	dirs := paths.GetDirs(projectsPath)
	for _, val := range dirs {
		if val == cloneName {
			refuse(w, "The project with the same name is exist")
		}
	}
	projectConf := configs.GetCurrentProjectConfig()

	// A deployer-managed project is not something a file copy can reproduce.
	//
	// Cloning copies the source tree and untars it into the clone with
	// `rm -rf /var/www/html/*` first. Under deployer that tree is not the
	// application: it is `releases/` — every release still on disk — plus
	// `shared/`, `current` and deployer's own state. The copy would carry all of
	// it, the clone would inherit `deploy/enabled` pointing at the original's
	// deploy path, and the result is a project that looks cloned and is a
	// hybrid of two.
	//
	// The same shape as the other refusal in snapshot:restore, and for the same
	// reason: `/var/www/html` means the application and the mount at once on an
	// ordinary project, and two different things here.
	if configs.IsDeployManaged(projectConf) {
		if w != nil {
			w.Fail("project:clone is refused on a project managed by deployer: its source tree is the deploy layout, not the application")
		}
		fmtc.WarningLn("project:clone is refused on a project managed by deployer.")
		fmt.Println("  Its source tree is releases/, shared/ and current — the deploy layout, not the")
		fmt.Println("  application. Copying it would produce a project pointing at the original's")
		fmt.Println("  deploy path, with every old release along for the ride.")
		fmt.Println("")
		fmt.Println("  Set up the second environment from the repository instead: create the project,")
		fmt.Println("  `madock deploy:enable`, and deploy the branch you want into it.")
		os.Exit(1)
	}

	// Before anything is copied: a source set to an engine this build lacks.
	engine, engineName, engineErr := snapshotengine.For(projectConf)
	if engineErr != nil {
		refuse(w, engineErr.Error())
	}

	// --from: the clone is made from a snapshot, not from the project as it is.
	// Resolved and checked here, before anything is created, for the same
	// reason as the engine above.
	from := resolveFrom(w, configs.GetProjectName(), args.From)

	fmt.Println("Cloning the project")
	exPath := paths.GetExecDirPath()
	projectName := configs.GetProjectName()
	currentDest := paths.MakeDirsByPath(exPath + "/projects/" + projectName + "/")
	dest := paths.MakeDirsByPath(exPath + "/projects/" + cloneName + "/")
	configs.PrepareDirsForProject(cloneName)
	files := paths.GetFilesRecursively(currentDest)
	for _, val := range files {
		paths.MakeDirsByPath(dest + strings.Replace(strings.Replace(val, currentDest, "", -1), "/"+filepath.Base(val), "", -1))
		err := paths.Copy(val, dest+strings.Replace(val, currentDest, "", -1))
		if err != nil {
			logger.Fatal(err)
		}
	}

	clonePathParts := strings.Split(projectConf["path"], "/")
	clonePath := strings.Join(clonePathParts[:len(clonePathParts)-1], "/") + "/" + cloneName
	paths.MakeDirsByPath(clonePath)
	configs.SetParam(cloneName, "path", clonePath, projectConf["activeScope"], "")
	// Rewrite cloned hosts so they don't collide with the source project.
	// Suffix is inserted before the TLD dot: shop.test + "-update" -> shop-update.test
	cloneProjectConfTmp := configs.GetProjectConfig(cloneName)
	clonedHosts := []string{}
	for key, val := range cloneProjectConfTmp {
		if val == "" || !strings.Contains(key, "/hosts/") || !strings.HasSuffix(key, "/name") {
			continue
		}
		host := applyDomainSuffix(val, domainSuffix)
		configs.SetParam(cloneName, key, host, projectConf["activeScope"], "")
		clonedHosts = append(clonedHosts, host)
	}
	sort.Strings(clonedHosts)
	// Before the clone's configuration is read below, so everything built from
	// it — the compose file, cron — is built muted from the first start.
	if args.Muted {
		configs.SetParam(cloneName, "clone/muted", "true", projectConf["activeScope"], "")
	}
	// From a snapshot the clone's volumes hold the passwords the snapshot was
	// taken with, which need not be the source's any more.
	if from.dir != "" {
		credentials.Apply(cloneName, projectConf["activeScope"], from.dir, cloneProjectConfTmp)
	}
	cloneProjectConf := configs.GetProjectConfig(cloneName)

	// The source stands still while it is copied.
	//
	// It used to be copied live, on the grounds that cloning should not
	// interrupt whatever the source is doing. That cannot hold: a running InnoDB
	// writes to its log during any copy, so the archive came out torn — and once
	// the copy was checked for exactly that, clone stopped working at all for
	// any project with a database, which is every project worth cloning.
	//
	// With an engine the databases are the engine's to copy, before the clone's
	// containers exist; the files still travel as files.tar.gz.
	//
	// From a snapshot nothing about the source moves: its saved state is what
	// the clone gets, whatever the source has done since.
	clone := snapshotengine.Project{Name: cloneName, Conf: cloneProjectConf}
	if from.dir != "" {
		if from.engine != nil {
			if err := from.engine.CloneSnapshot(snapshotengine.Project{Name: projectName, Conf: projectConf}, from.ref, clone); err != nil {
				refuse(w, fmt.Sprintf("%s clone of snapshot %s: %v", from.engineName, filepath.Base(from.dir), err))
			}
		}
		if err := linkArchives(from.dir, dest); err != nil {
			refuse(w, err.Error())
		}
		// Code of a tar snapshot: another edition may give it without the
		// unpack, and then files.tar.gz has nothing left to do here.
		if from.engine == nil {
			handled, err := snapshotengine.CodeFromArchive(snapshotengine.Project{Name: projectName, Conf: projectConf}, from.dir, clone)
			if err != nil {
				refuse(w, "the clone's code: "+err.Error())
			}
			if handled {
				_ = os.Remove(dest + "files.tar.gz")
			}
		}
	} else if engine != nil {
		if err := create.CloneWithEngine(engine, engineName, projectConf, projectName, clone, dest); err != nil {
			refuse(w, err.Error())
		}
	} else {
		create.CopyStandingStill(projectConf, projectName, dest)
	}

	// Code the engine keeps: the clone of the storage already holds it, so the
	// clone's directory beside the source becomes a link to it, and no
	// files.tar.gz was written to unpack.
	codeEngine := engine
	if from.dir != "" {
		codeEngine = nil
		if from.engine != nil {
			codeEngine = from.engine.(snapshotengine.Engine)
		}
	}
	if codeEngine != nil {
		if dir := snapshotengine.CodeDirOf(codeEngine, clone); dir != "" {
			if err := linkCode(clonePath, dir); err != nil {
				refuse(w, err.Error())
			}
		}
	}

	containerName := docker.GetContainerName(cloneProjectConf, cloneName, "snapshot")
	if paths.IsFileExist(paths.CacheDir() + "/conf-cache") {
		err := os.Remove(paths.CacheDir() + "/conf-cache")
		if err != nil {
			logger.Fatal(err)
		}
	}
	docker.UpWithBuild(cloneName, false)
	docker.Down(cloneName, false)
	docker.UpSnapshot(cloneName)
	if paths.IsFileExist(dest + "db.tar.gz") {
		selectedFile, err := os.Open(dest + "db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/mysql/* && cd /var/www/mysql && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		out, err := gzip.NewReader(selectedFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer out.Close()
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	if paths.IsFileExist(dest + "db2.tar.gz") {
		selectedFileDb2, err := os.Open(dest + "db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileDb2.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/mysql2/mysql/* && cd /var/www/mysql2/mysql && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		outDb2, err := gzip.NewReader(selectedFileDb2)
		if err != nil {
			logger.Fatal(err)
		}
		defer outDb2.Close()
		cmd.Stdin = outDb2
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	// The other named volumes, unless the engine already gave the clone its own.
	// The archives sit in the clone's project directory only for this step.
	volumesEngine := engine
	if from.dir != "" {
		volumesEngine = nil
		if from.engine != nil {
			volumesEngine = from.engine.(snapshotengine.Engine)
		}
	}
	if volumesEngine == nil || !snapshotengine.TakesAll(volumesEngine) {
		others := snapshotengine.Others(cloneProjectConf)
		if err := create.RestoreVolumes(containerName, others, dest); err != nil {
			logger.Fatal(err)
		}
		for _, v := range others {
			_ = os.Remove(dest + v.Archive())
		}
	}

	if paths.IsFileExist(dest + "files.tar.gz") {
		selectedFileFiles, err := os.Open(dest + "files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileFiles.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/html/* && cd /var/www/html && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		outFiles, err := gzip.NewReader(selectedFileFiles)
		if err != nil {
			logger.Fatal(err)
		}
		defer outFiles.Close()
		cmd.Stdin = outFiles
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	docker.StopSnapshot(cloneName)

	if args.Muted {
		if err := purgeQueues(cloneProjectConf, cloneName); err != nil {
			refuse(w, "the clone was made, but its RabbitMQ still holds the source's messages: "+err.Error())
		}
	}

	if w != nil {
		w.Print(ClonedJSON{Name: cloneName, Path: clonePath, Hosts: clonedHosts})
		return
	}

	fmtc.SuccessLn("Project cloned successfully.\nThe new project name is " + cloneName + ".\nThe path to the project is " + clonePath)
}

// purgeQueues empties every queue of a muted clone's RabbitMQ, keeping the
// queues themselves.
//
// The clone's broker holds whatever the source had queued — orders to export,
// mails to send — and the clone's consumers would process each of them a second
// time. The queues stay: an application publishes to exchanges and queues it
// declared once at setup, and an empty broker without them refuses its next
// publish. Only a persisted broker carries anything (rabbitmq/persist); one
// without the volume starts empty already.
func purgeQueues(conf map[string]string, cloneName string) error {
	if conf["rabbitmq/enabled"] != "true" || conf["rabbitmq/persist"] != "true" {
		return nil
	}
	fmtc.TitleLn("Emptying the clone's RabbitMQ queues...")
	if err := docker.UpService(cloneName, "rabbitmq"); err != nil {
		return err
	}
	defer func() { _ = docker.StopService(cloneName, "rabbitmq") }()
	// As the broker's own user, not root. rabbitmqctl reads the Erlang cookie,
	// and run as root it creates one when there is none yet — owned by root,
	// unreadable to the broker, which then dies at its next start with
	// "Error when reading /var/lib/rabbitmq/.erlang.cookie: eacces". The e2e
	// run hit exactly that.
	cmd, err := docker.PrepareContainerExec(docker.GetContainerName(conf, cloneName, "rabbitmq"), "rabbitmq", false, "bash", "-c", purgeScript)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// purgeScript waits for the broker, then purges every queue of every vhost.
// A broker that never answers is a failure, not "nothing to purge".
//
// check_running, not ping: ping answers as soon as the Erlang node is up, while
// the rabbit application inside it may still be starting, and list_queues then
// fails with "this command requires the 'rabbit' app to be running".
const purgeScript = `ok=
for i in $(seq 1 90); do rabbitmq-diagnostics -q check_running >/dev/null 2>&1 && { ok=1; break; }; sleep 2; done
[ -n "$ok" ] || { echo "RabbitMQ did not come up" >&2; exit 1; }
rabbitmqctl -q --no-table-headers list_vhosts | while read -r vhost; do
  rabbitmqctl -q --no-table-headers list_queues -p "$vhost" name | while read -r queue; do
    rabbitmqctl -q purge_queue -p "$vhost" "$queue" || exit 1
  done || exit 1
done`

// fromSnapshot is the snapshot `--from` named. dir is empty without --from;
// engine is nil when the snapshot's databases are in tar archives.
type fromSnapshot struct {
	dir        string
	engine     snapshotengine.SnapshotCloner
	engineName string
	ref        string
}

// resolveFrom finds the snapshot --from names and refuses, before anything is
// created, one this build cannot clone from.
func resolveFrom(w *output.Writer, projectName, from string) fromSnapshot {
	from = strings.TrimSpace(from)
	if from == "" {
		return fromSnapshot{}
	}
	snapshots := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/snapshot"
	name := restore.MatchName(create.OrderedNames(snapshots), from)
	if name == "" {
		refuse(w, fmt.Sprintf("No snapshot named %q; snapshot:list shows the ones there are", from))
	}
	out := fromSnapshot{dir: filepath.Join(snapshots, name)}
	marker, byEngine, err := snapshotengine.ReadMarker(out.dir)
	if err != nil {
		refuse(w, err.Error())
	}
	if !byEngine {
		return out
	}
	engine, ok := snapshotengine.ByName(marker.Engine)
	if !ok {
		refuse(w, fmt.Sprintf("snapshot %s is held by %q, which this build does not have", name, marker.Engine))
	}
	cloner, ok := engine.(snapshotengine.SnapshotCloner)
	if !ok {
		refuse(w, fmt.Sprintf("snapshot %s is held by %q, which cannot clone from a snapshot in this build", name, marker.Engine))
	}
	out.engine, out.engineName, out.ref = cloner, marker.Engine, marker.Ref
	return out
}

// linkArchives puts the snapshot's archives where the clone unpacks them from.
// A hard link when it can be one: the clone removes each archive once it is
// unpacked, and removing a link leaves the snapshot's copy alone — while a copy
// of a multi-gigabyte archive would be the slowest step of the clone.
func linkArchives(snapshotDir, dest string) error {
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		src, dst := filepath.Join(snapshotDir, e.Name()), filepath.Join(dest, e.Name())
		if err := os.Link(src, dst); err == nil {
			continue
		}
		if err := paths.Copy(src, dst); err != nil {
			return fmt.Errorf("copying %s from the snapshot: %w", e.Name(), err)
		}
	}
	return nil
}

// linkCode replaces the clone's freshly made, empty directory with a link to
// where its engine keeps the code. A directory that is not empty is refused:
// something is there that the link would hide.
func linkCode(clonePath, codeDir string) error {
	entries, err := os.ReadDir(clonePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty; the clone's code is in %s and cannot be linked there", clonePath, codeDir)
	}
	if err := os.Remove(clonePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(codeDir, clonePath)
}

// applyDomainSuffix inserts suffix before the last dot of domain. If domain
// has no dot the suffix is appended. shop.test + "-update" -> shop-update.test
func applyDomainSuffix(domain, suffix string) string {
	idx := strings.LastIndex(domain, ".")
	if idx == -1 {
		return domain + suffix
	}
	return domain[:idx] + suffix + domain[idx:]
}
