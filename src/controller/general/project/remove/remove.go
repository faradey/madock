package remove

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/ports"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool   `arg:"-f,--force" help:"Skip interactive confirmations (requires --name)"`
	Name  string `arg:"-n,--name" help:"Project name to remove (required with --force)"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"project:remove"},
		Handler:  Execute,
		Help:     "Remove project",
		Category: "project",
		ArgsType: new(ArgsStruct),
		// Runs outside a project on purpose, and only for --name.
		//
		// The scope check refuses a project command run anywhere else, which is
		// right for every other one and wrong for this: an orphan is an entry
		// whose source directory is gone, so there is nowhere to stand to remove
		// it. project:list names such entries and nothing could remove them —
		// the tool could describe the problem and not act on it.
		//
		// Without --name the handler still requires a project directory, so the
		// protection that refuses to destroy the wrong one is unchanged.
		Global: true,
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	// A name is answered by the registry, not by the working directory.
	if args.Name != "" && args.Name != configs.GetProjectName() {
		removeNamed(args.Name, args.Force)
		return
	}

	projectName := configs.GetProjectName()

	if !mayRemove(projectName) {
		os.Exit(1)
	}

	// Non-interactive mode with --force flag
	if args.Force {
		if args.Name == "" {
			fmtc.ErrorLn("--force requires --name to specify the project name")
			return
		}
		if args.Name != projectName {
			fmtc.ErrorLn("Project name mismatch. Current project: " + projectName + ", specified: " + args.Name)
			return
		}
		removeProject(projectName)
		return
	}

	// Interactive mode
	fmt.Println("Are you sure? (y/n)")
	fmt.Print("> ")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	result := strings.ToLower(strings.TrimSpace(string(sentence)))
	if result == "y" && len(projectName) > 0 {
		pp := paths.NewProjectPaths(projectName)
		fmt.Println("The following items will be removed:")
		fmt.Println(paths.GetExecDirPath() + "/projects/" + projectName + "/")
		fmt.Println(pp.RuntimeDir())
		fmt.Println(paths.GetRunDirPath())
		fmt.Println("Containers, images and volumes associated with the project.")
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm the deletion of the project")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		result = strings.TrimSpace(string(sentence))
		if result == projectName {
			removeProject(projectName)
		} else {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
		}
	}
}

// mayRemove answers whether this directory may be destroyed at all.
//
// removeProject finishes with RemoveAll on the current directory, and until now
// the only thing standing between that and the wrong directory was the name: the
// project name comes from the directory name, and --force merely checks that the
// caller repeated it. A leftover runtime directory with no configuration was
// enough to make any same-named directory look like a project — and one such
// leftover was the madock installation itself, whose runtime `src` was a symlink
// back to the source tree. `project:remove --force --name madock` there would have
// deleted madock, its repository and every other project's configuration with it.
//
// So three refusals before anything is touched: the installation, a directory that
// is not the project's own, and a project that does not exist.
func mayRemove(projectName string) bool {
	runDir := paths.GetRunDirPath()

	if configs.IsSamePath(runDir, paths.GetExecDirPath()) {
		fmtc.ErrorLn("This directory is the madock installation, not a project — refusing to remove it")
		return false
	}

	if !paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml") {
		fmtc.ErrorLn("There is no project '" + projectName + "' to remove: it has no configuration")
		fmtc.ToDoLn("Leftover generated files, if any, are under " + paths.GetExecDirPath() + "/aruntime/projects/" + projectName)
		return false
	}

	// A recorded path that disagrees with where we stand means the name resolved
	// to somebody else's project, and RemoveAll would take this directory apart on
	// its behalf.
	stored := configs.GetProjectConfigOnly(projectName)["path"]
	if stored != "" && !configs.IsSamePath(stored, runDir) {
		fmtc.ErrorLn("The project '" + projectName + "' is registered at another path, so this directory is not it")
		fmtc.ErrorLn("  registered: " + stored)
		fmtc.ErrorLn("  current:    " + runDir)
		return false
	}

	return true
}

func removeProject(projectName string) {
	// Say what is about to go even when --force skipped the interactive listing:
	// this is the one command that deletes the directory the caller is standing in.
	pp := paths.NewProjectPaths(projectName)
	fmtc.WarningLn("Removing project '" + projectName + "':")
	fmtc.WarningLn("  " + paths.GetExecDirPath() + "/projects/" + projectName + "/")
	fmtc.WarningLn("  " + pp.RuntimeDir())
	fmtc.WarningLn("  " + paths.GetRunDirPath())
	fmtc.WarningLn("  containers, images and volumes of the project")

	removeRegistered(projectName)

	// The project's own directory, and only when standing in it. Split out so
	// that removing an entry by name — where the source is gone and the caller
	// is standing somewhere else entirely — cannot reach this line.
	if err := os.RemoveAll(paths.GetRunDirPath()); err != nil {
		logger.Fatal(err)
	}
}

// removeRegistered takes down everything the installation holds for a project:
// its containers, its registry entry, its generated runtime, its block in the
// shared proxy and its port reservation. Everything except the project's own
// directory, which is the caller's to decide about.
func removeRegistered(projectName string) {
	pp := paths.NewProjectPaths(projectName)

	// Before the containers go: anything they wrote as root has to be handed
	// back, or the deletion below stops at the first such file and leaves the
	// project half removed.
	docker.ReclaimProjectFiles(projectName)

	docker.Down(projectName, true)

	if err := os.RemoveAll(paths.GetExecDirPath() + "/projects/" + projectName + "/"); err != nil {
		logger.Fatal(err)
	}

	if err := os.RemoveAll(pp.RuntimeDir()); err != nil {
		logger.Fatal(err)
	}

	// Take the project out of the shared proxy.
	//
	// Removal is the one moment this belongs to. A stopped project keeps its
	// routing on purpose — it is coming back, and rewriting every other
	// project's configuration for a pause would be churn and risk for nothing.
	// A removed project is not coming back, and its server block points at a
	// container that no longer exists.
	//
	// Nothing did this before, so the block survived until something else
	// happened to regenerate the file.
	// Regenerated on behalf of a project that still exists, never the removed
	// one: MakeConf recreates directories and allocates ports for the name it is
	// given, so naming the corpse brings its registry entry and its port
	// reservation straight back.
	remaining := paths.GetActiveProjects()
	if len(remaining) > 0 {
		nginx.MakeConf(remaining[0])
		if err := docker.ReloadNginx(); err != nil {
			logger.Println(err)
		}
	}

	// Ports come last, and the order is not cosmetic: generating the proxy
	// configuration allocates a port for the project it is given, so releasing
	// them first only meant handing them straight back. The removed project then
	// kept a reservation no other project could use.
	ports.GetRegistry().RemoveProject(projectName)

	fmtc.SuccessLn("Project was removed successfully")
	fmtc.SuccessLn("!!! Close the terminal for the changes to take effect !!!")
}

// removeNamed removes a registry entry from anywhere, by name.
//
// It exists for the orphan: an entry whose source directory is gone. project:list
// has been able to name those since 3.8.50 and nothing could remove them, because
// this command asked the working directory who the project was — and for an
// orphan there is no directory to stand in. The workaround was to recreate the
// directory with a copy of its config.xml and delete from there, which is a
// strange thing for a tool to require of the person cleaning up after it.
//
// A project whose source still exists is refused and told where it is. That keeps
// the protection this command is built around: removeProject ends with RemoveAll
// on the working directory, so removing a live project from somewhere else would
// take that somewhere else with it.
func removeNamed(projectName string, force bool) {
	var entry *configs.ProjectEntry
	for _, candidate := range configs.ListProjects() {
		if candidate.Name == projectName {
			found := candidate
			entry = &found
			break
		}
	}

	if entry == nil {
		fmtc.ErrorLn("No project named '" + projectName + "' is registered in this installation")
		fmtc.ToDoLn("madock project:list")
		os.Exit(1)
	}

	if entry.State == configs.ProjectOk {
		fmtc.ErrorLn("Project '" + projectName + "' still has its source directory: " + entry.Path)
		fmtc.ToDoLn("Remove it from there, so the directory goes with it:")
		fmtc.ToDoLn("  cd " + entry.Path + " && madock project:remove --force --name=" + projectName)
		os.Exit(1)
	}

	fmtc.WarningLn("Removing the registry entry '" + projectName + "':")
	fmtc.WarningLn("  " + paths.GetExecDirPath() + "/projects/" + projectName + "/")
	fmtc.WarningLn("  " + paths.NewProjectPaths(projectName).RuntimeDir())
	fmtc.WarningLn("  containers, images and volumes of the project")
	if entry.Path != "" {
		fmtc.WarningLn("  its source directory is already gone: " + entry.Path)
	}

	if !force {
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm")
		fmt.Print("> ")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		if strings.TrimSpace(string(sentence)) != projectName {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
			return
		}
	}

	removeRegistered(projectName)
}

// definition returns this command's registration, for tests that assert on it.
func definition() *command.Definition {
	def, _ := command.Get("project:remove")
	return def
}
