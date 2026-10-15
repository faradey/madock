package clone

import (
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/snapshot/create"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
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
	for key, val := range cloneProjectConfTmp {
		if val == "" || !strings.Contains(key, "/hosts/") || !strings.HasSuffix(key, "/name") {
			continue
		}
		configs.SetParam(cloneName, key, applyDomainSuffix(val, domainSuffix), projectConf["activeScope"], "")
	}
	cloneProjectConf := configs.GetProjectConfig(cloneName)

	// The source stands still while it is copied.
	//
	// It used to be copied live, on the grounds that cloning should not
	// interrupt whatever the source is doing. That cannot hold: a running InnoDB
	// writes to its log during any copy, so the archive came out torn — and once
	// the copy was checked for exactly that, clone stopped working at all for
	// any project with a database, which is every project worth cloning.
	create.CopyStandingStill(projectConf, projectName, dest)

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

	if w != nil {
		cloned := ClonedJSON{Name: cloneName, Path: clonePath, Hosts: []string{}}
		for key, val := range configs.GetProjectConfig(cloneName) {
			if val != "" && strings.Contains(key, "/hosts/") && strings.HasSuffix(key, "/name") {
				cloned.Hosts = append(cloned.Hosts, val)
			}
		}
		sort.Strings(cloned.Hosts)
		w.Print(cloned)
		return
	}

	fmtc.SuccessLn("Project cloned successfully.\nThe new project name is " + cloneName + ".\nThe path to the project is " + clonePath)
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
