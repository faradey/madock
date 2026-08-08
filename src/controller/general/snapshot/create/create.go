package create

import (
	"compress/gzip"
	"errors"
	"fmt"
	"os/exec"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/dbtarget"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"os"
	"strings"
	"time"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"snapshot:create"},
		Handler:  Execute,
		Help:     "Create snapshot",
		Category: "snapshot",
		ArgsType: new(arg_struct.ControllerGeneralSnapshotCreate),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSnapshotCreate)).(*arg_struct.ControllerGeneralSnapshotCreate)
	projectConf := configs.GetCurrentProjectConfig()
	exPath := paths.GetExecDirPath()
	projectName := configs.GetProjectName()
	dest := paths.MakeDirsByPath(exPath + "/projects/" + projectName + "/backup/snapshot")

	name := "snapshot-"
	if args.Name != "" {
		name += args.Name + "-"
	}
	name += time.Now().Format("2006-01-02-15-04-05")

	dbsPath := paths.MakeDirsByPath(dest + "/" + name + "/")

	// The snapshot is taken with the project's own containers stopped, from the
	// helper container that mounts the same volumes — the mirror of what
	// snapshot:restore already does.
	//
	// Copying a database's data directory out of a running server produces a
	// torn set of pages: an archive that looks like a backup and may refuse to
	// start. There is no way around that from inside the running container, so
	// the project stands still for the length of the copy.
	//
	// `stop` rather than `down`: the containers stay, so bringing the project
	// back afterwards is a start and not a rebuild.
	wasRunning := isRunning(projectName)
	if wasRunning {
		fmtc.TitleLn("Stopping containers for a consistent copy...")
		if err := docker.Stop(projectName); err != nil {
			logger.Fatal(err)
		}
	}

	docker.UpSnapshot(projectName)
	container := docker.GetContainerName(projectConf, projectName, "snapshot")

	archiveDatabases(projectConf, projectName, container, snapshotVolumePaths, dbsPath)
	archiveFiles(container, snapshotVolumePaths.files, dbsPath)

	docker.StopSnapshot(projectName)

	if wasRunning {
		fmtc.TitleLn("Starting containers again...")
		if err := docker.Start(projectName); err != nil {
			logger.Fatal(err)
		}
	}

	fmt.Println("Snapshot completed successfully")
}

// volumePaths names where the data lives inside the container being read.
type volumePaths struct {
	db, db2, files string
}

// snapshotVolumePaths are the mounts of the helper `snapshot` container, which
// snapshot:restore untars into. Create and restore have to agree on them.
var snapshotVolumePaths = volumePaths{
	db:    "/var/www/mysql",
	db2:   "/var/www/mysql2/mysql",
	files: "/var/www/html",
}

// liveVolumePaths are the same data seen from the project's own running
// containers. project:clone reads the source project this way because it must
// not take the source down.
var liveVolumePaths = volumePaths{
	db:    "/var/lib/mysql",
	db2:   "/var/lib/mysql",
	files: "/var/www/html",
}

// isRunning reports whether the project has containers up right now, so a
// snapshot can put them back the way it found them.
// CopyStandingStill writes the project's databases and files into dest with the
// project stopped, then starts it again if it was running.
//
// It exists because project:clone needs precisely what snapshot:create does, and
// the first attempt to give it that stopped the containers and then tried to
// read from them — "container is not running", reported as a torn copy. The
// data is read from the helper container that mounts the same volumes, which is
// the only place it can be read from while the servers are down.
func CopyStandingStill(projectConf map[string]string, projectName, dest string) {
	wasRunning := isRunning(projectName)
	if wasRunning {
		fmtc.TitleLn("Stopping the source for a consistent copy...")
		if err := docker.Stop(projectName); err != nil {
			logger.Fatal(err)
		}
	}

	docker.UpSnapshot(projectName)
	container := docker.GetContainerName(projectConf, projectName, "snapshot")

	archiveDatabases(projectConf, projectName, container, snapshotVolumePaths, dest)
	archiveFiles(container, snapshotVolumePaths.files, dest)

	docker.StopSnapshot(projectName)

	if wasRunning {
		fmtc.TitleLn("Starting the source again...")
		if err := docker.Start(projectName); err != nil {
			logger.Fatal(err)
		}
	}
}

func isRunning(projectName string) bool {
	for _, active := range paths.GetActiveProjects() {
		if strings.EqualFold(active, projectName) {
			return true
		}
	}
	return false
}

// GetDB copies the project's databases out of its own running containers.
// project:clone uses it: cloning must not take the source project down, so it
// accepts the torn-copy risk that snapshot:create refuses.
func GetDB(projectConf map[string]string, projectName string, dbsPath string) {
	archiveDatabases(projectConf, projectName, "", liveVolumePaths, dbsPath)
}

// archiveDatabases writes db.tar.gz and, when the project has one, db2.tar.gz.
// container names the container to read from; empty means the project's own db
// and db2 containers.
func archiveDatabases(projectConf map[string]string, projectName, container string, dirs volumePaths, dbsPath string) {
	// A snapshot copies the data directory of a container this project owns.
	// A project without its own db service has nothing to copy — and if its data
	// lives on another project's server, that directory holds every other
	// consumer's data too, so restoring a copy taken from here would overwrite
	// all of them.
	if !dbtarget.HasLocal(projectConf, "db") {
		fmtc.WarningLn("Skipping the database: this project does not run its own db service.")
		if target, ok := dbtarget.Resolve(projectConf, projectName, "db"); ok && target.Shared {
			fmtc.WarningLn("Its data lives on project \"" + target.Project + "\" — snapshot it there.")
			fmtc.WarningLn("A snapshot copies the whole data directory, which on that server holds every consumer.")
		}
		return
	}

	dbContainer := container
	if dbContainer == "" {
		dbContainer = docker.GetContainerName(projectConf, projectName, "db")
	}
	archiveDataDir(dbContainer, dirs.db, dbsPath+"db.tar.gz")

	if projectConf["db2/enabled"] == "true" {
		db2Container := container
		if db2Container == "" {
			db2Container = docker.GetContainerName(projectConf, projectName, "db2")
		}
		archiveDataDir(db2Container, dirs.db2, dbsPath+"db2.tar.gz")
	}
}

// archiveDataDir copies a database container's data directory into an archive.
//
// A file that changed mid-copy is fatal here, unlike for the project files. The
// data directory of a running server is a set of pages that only make sense
// together: a torn copy of it is not "slightly out of date", it is an archive
// that may refuse to start. Better to stop and say so than to store something
// that looks like a backup.
func archiveDataDir(containerName, dir, archivePath string) {
	err := archive(containerName, dir, archivePath)
	if errors.Is(err, errFilesChanged) {
		// archive keeps the file so the caller can decide. This one cannot be
		// kept: left in the snapshot directory it is indistinguishable from a
		// good copy, and snapshot:restore would happily write it back.
		_ = os.Remove(archivePath)
		fmtc.ErrorLn("The server wrote to its data directory while it was being copied, so the copy is torn.")
		fmtc.ToDoLn("Use 'madock db:export' for a dump that can be relied on.")
		logger.Fatal(err)
	}
	if err != nil {
		logger.Fatal(err)
	}
}

// GetFiles copies the project's files out of its own running container.
// project:clone uses it, for the same reason it uses GetDB.
func GetFiles(projectConf map[string]string, projectName string, dbsPath string) {
	// The service running the application code, not "php". Every platform mounts
	// the project at /var/www/html, but only a PHP one has a php container —
	// snapshot:create died with "No such container" on all the others.
	mainService := configs.ResolveMainService(projectConf, "php")
	archiveFiles(docker.GetContainerName(projectConf, projectName, mainService), liveVolumePaths.files, dbsPath)
}

func archiveFiles(containerName, dir, dbsPath string) {
	err := archive(containerName, dir, dbsPath+"files.tar.gz")
	if errors.Is(err, errFilesChanged) {
		// Expected of any project whose containers are up: a log rotates, a
		// cache file is written. The archive is complete and every other file
		// in it is intact, so this is worth saying and not worth stopping for.
		fmtc.WarningLn("Some files changed while they were being archived — a running project writes as it is copied.")
		fmtc.WarningLn("The snapshot is complete; those files are as they were partway through the copy.")
		return
	}
	if err != nil {
		logger.Fatal(err)
	}
}

// errFilesChanged is tar's exit status 1: the archive was written in full, but
// something under it changed while it was being read, so those members do not
// match any single moment. Fatal errors are 2 and above, and stay errors.
var errFilesChanged = errors.New("files changed while being archived")

// isFilesChanged reports tar's exit status 1 — "some files differ", which
// during --create means the archive is complete but some members were written
// while they were being read.
//
// Only 1. Status 2 and above are tar's fatal errors (unreadable member, no
// space, broken pipe), and --ignore-failed-read is deliberately not passed, so
// a permission problem cannot arrive here disguised as a changed file.
func isFilesChanged(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

// archive streams a gzipped tar of dir out of a container and into archivePath.
//
// tar writes to stdout rather than to a file inside the container: its exit
// status then reaches this function instead of being swallowed by a shell
// chain, and the container does not need room for a second copy of the data.
// The old form was `tar -czf /tmp/x.tar.gz . && cat /tmp/x.tar.gz`, where exit
// 1 skipped the cat and produced an empty archive plus an error.
//
// The archive is gzipped by tar and again by the writer here. That is one layer
// too many, and it stays: snapshot:restore feeds the outer-ungzipped stream to
// `tar -zxf -`, so removing it would make every existing snapshot unrestorable.
func archive(containerName, dir, archivePath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}

	writer := gzip.NewWriter(file)

	// `cd || exit 2` keeps a missing directory out of the exit-1 case, where it
	// would read as "some files changed" and pass off an empty archive as one.
	cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c",
		"cd "+dir+" || exit 2; tar -czf - .")
	if prepErr != nil {
		writer.Close()
		file.Close()
		_ = os.Remove(archivePath)
		return prepErr
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()

	if isFilesChanged(runErr) {
		// The archive is complete — keep it and let the caller decide.
		writer.Close()
		file.Close()
		return errFilesChanged
	}

	if runErr != nil {
		writer.Close()
		file.Close()
		_ = os.Remove(archivePath)
		return runErr
	}

	if err := writer.Close(); err != nil {
		file.Close()
		_ = os.Remove(archivePath)
		return err
	}

	return file.Close()
}
